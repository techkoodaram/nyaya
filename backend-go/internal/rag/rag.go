package rag

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

var tokenRegex = regexp.MustCompile(`[a-z0-9]+`)

type Document struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Title    string `json:"title"`
	Path     string `json:"path"`
	Content  string `json:"content"`
	WordSize int    `json:"wordSize"`
}

type SearchResult struct {
	ID      string  `json:"id"`
	Source  string  `json:"source"`
	Title   string  `json:"title"`
	Path    string  `json:"path"`
	Score   float64 `json:"score"`
	Excerpt string  `json:"excerpt"`
}

type Index struct {
	mu       sync.RWMutex
	docs     []Document
	docTFIDF []map[string]float64
	docNorms []float64
	idf      map[string]float64

	dataRoot   string
	lastLoaded time.Time
}

func NewIndex(dataRoot string) (*Index, error) {
	idx := &Index{dataRoot: dataRoot}
	if err := idx.Reload(); err != nil {
		return nil, err
	}
	return idx, nil
}

func (i *Index) Reload() error {
	docs, err := loadCorpus(i.dataRoot)
	if err != nil {
		return err
	}
	if len(docs) == 0 {
		return fmt.Errorf("no corpus documents found in %s (expected files under data/ipc and data/judgments)", i.dataRoot)
	}

	idf, tfidfVectors, norms := buildVectors(docs)

	i.mu.Lock()
	defer i.mu.Unlock()
	i.docs = docs
	i.idf = idf
	i.docTFIDF = tfidfVectors
	i.docNorms = norms
	i.lastLoaded = time.Now()
	return nil
}

func (i *Index) Stats() map[string]any {
	i.mu.RLock()
	defer i.mu.RUnlock()

	var ipcCount int
	var judgmentCount int
	for _, d := range i.docs {
		switch d.Source {
		case "ipc":
			ipcCount++
		case "judgment":
			judgmentCount++
		}
	}

	return map[string]any{
		"documents":       len(i.docs),
		"ipcDocuments":    ipcCount,
		"judgmentDocs":    judgmentCount,
		"lastLoaded":      i.lastLoaded.Format(time.RFC3339),
		"dataRoot":        i.dataRoot,
		"indexedKeywords": len(i.idf),
	}
}

func (i *Index) Search(query string, topK int) ([]SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("query is required")
	}
	if topK <= 0 {
		topK = 3
	}
	if topK > 10 {
		topK = 10
	}

	i.mu.RLock()
	defer i.mu.RUnlock()

	if len(i.docs) == 0 {
		return nil, errors.New("index is empty")
	}

	qVec, qNorm := queryVector(query, i.idf)
	if qNorm == 0 {
		return nil, nil
	}

	type scoredDoc struct {
		idx   int
		score float64
	}
	scored := make([]scoredDoc, 0, len(i.docs))
	for docIdx, docVec := range i.docTFIDF {
		score := cosineSimilarity(qVec, qNorm, docVec, i.docNorms[docIdx])
		if score > 0 {
			scored = append(scored, scoredDoc{idx: docIdx, score: score})
		}
	}

	sort.Slice(scored, func(a, b int) bool {
		if scored[a].score == scored[b].score {
			return i.docs[scored[a].idx].Title < i.docs[scored[b].idx].Title
		}
		return scored[a].score > scored[b].score
	})

	if len(scored) > topK {
		scored = scored[:topK]
	}

	results := make([]SearchResult, 0, len(scored))
	for _, s := range scored {
		doc := i.docs[s.idx]
		results = append(results, SearchResult{
			ID:      doc.ID,
			Source:  doc.Source,
			Title:   doc.Title,
			Path:    doc.Path,
			Score:   math.Round(s.score*10000) / 10000,
			Excerpt: bestExcerpt(doc.Content, query),
		})
	}

	return results, nil
}

func GenerateAnswer(query string, hits []SearchResult) string {
	if len(hits) == 0 {
		return "No relevant IPC sections or court judgments were found for this query in the current corpus."
	}

	var b strings.Builder
	b.WriteString("Based on the indexed IPC provisions and court judgments, here are the most relevant references:\n\n")
	for idx, h := range hits {
		b.WriteString(fmt.Sprintf("%d. %s (%s)\n", idx+1, h.Title, strings.ToUpper(h.Source)))
		b.WriteString(fmt.Sprintf("   %s\n", h.Excerpt))
	}
	b.WriteString("\nUse these references as starting points and validate against full case text and current law.")
	return b.String()
}

func loadCorpus(dataRoot string) ([]Document, error) {
	sources := map[string]string{
		"ipc":      filepath.Join(dataRoot, "ipc"),
		"judgment": filepath.Join(dataRoot, "judgments"),
	}

	var docs []Document
	for source, baseDir := range sources {
		entries, err := os.ReadDir(baseDir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("read source dir %s: %w", baseDir, err)
		}
		if len(entries) == 0 {
			continue
		}

		if err := filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}

			doc, ok, err := parseFile(path, source)
			if err != nil {
				return err
			}
			if ok {
				docs = append(docs, doc)
			}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("walk %s: %w", baseDir, err)
		}
	}

	return docs, nil
}

func parseFile(path, source string) (Document, bool, error) {
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return Document{}, false, fmt.Errorf("read file %s: %w", path, err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	raw := strings.TrimSpace(string(contentBytes))
	if raw == "" {
		return Document{}, false, nil
	}

	var title string
	var content string
	switch ext {
	case ".json":
		title, content = extractFromJSON(raw, filepath.Base(path))
	default:
		title = titleFromText(raw, filepath.Base(path))
		content = raw
	}

	clean := normalizeWS(content)
	if clean == "" {
		return Document{}, false, nil
	}

	id := fmt.Sprintf("%s:%s", source, filepath.ToSlash(path))
	return Document{
		ID:       id,
		Source:   source,
		Title:    title,
		Path:     filepath.ToSlash(path),
		Content:  clean,
		WordSize: len(tokenize(clean)),
	}, true, nil
}

func extractFromJSON(raw, fallbackTitle string) (title, content string) {
	var parsed any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return titleFromText(raw, fallbackTitle), raw
	}

	title = fallbackTitle
	parts := make([]string, 0, 8)

	var walk func(v any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			for k, val := range x {
				lk := strings.ToLower(k)
				if title == fallbackTitle && (lk == "title" || lk == "section" || lk == "case_name" || lk == "name") {
					if s, ok := val.(string); ok && strings.TrimSpace(s) != "" {
						title = strings.TrimSpace(s)
					}
				}
				if lk == "text" || lk == "content" || lk == "body" || lk == "judgment" || lk == "judgement" || lk == "facts" || lk == "decision" || lk == "holding" || lk == "summary" || lk == "description" {
					if s, ok := val.(string); ok {
						parts = append(parts, s)
					}
				}
				walk(val)
			}
		case []any:
			for _, el := range x {
				walk(el)
			}
		case string:
			if len(strings.Fields(x)) >= 8 {
				parts = append(parts, x)
			}
		}
	}
	walk(parsed)

	if len(parts) == 0 {
		parts = append(parts, raw)
	}
	return strings.TrimSpace(title), normalizeWS(strings.Join(parts, "\n"))
}

func titleFromText(text, fallback string) string {
	lines := strings.Split(text, "\n")
	for _, ln := range lines {
		c := strings.TrimSpace(ln)
		if c != "" {
			if len(c) > 100 {
				return fallback
			}
			return c
		}
	}
	return fallback
}

func normalizeWS(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

func tokenize(text string) []string {
	text = strings.ToLower(text)
	return tokenRegex.FindAllString(text, -1)
}

func termFrequency(tokens []string) map[string]float64 {
	tf := make(map[string]float64)
	if len(tokens) == 0 {
		return tf
	}

	for _, t := range tokens {
		tf[t]++
	}

	n := float64(len(tokens))
	for token, count := range tf {
		tf[token] = count / n
	}
	return tf
}

func buildVectors(docs []Document) (map[string]float64, []map[string]float64, []float64) {
	docTF := make([]map[string]float64, len(docs))
	df := make(map[string]float64)

	for i, doc := range docs {
		tokens := tokenize(doc.Content)
		tf := termFrequency(tokens)
		docTF[i] = tf
		for token := range tf {
			df[token]++
		}
	}

	totalDocs := float64(len(docs))
	idf := make(map[string]float64, len(df))
	for token, docFreq := range df {
		idf[token] = math.Log((totalDocs+1)/(docFreq+1)) + 1
	}

	docTFIDF := make([]map[string]float64, len(docs))
	norms := make([]float64, len(docs))
	for i, tf := range docTF {
		vec := make(map[string]float64, len(tf))
		var sumSquares float64
		for token, tVal := range tf {
			w := tVal * idf[token]
			vec[token] = w
			sumSquares += w * w
		}
		docTFIDF[i] = vec
		norms[i] = math.Sqrt(sumSquares)
	}

	return idf, docTFIDF, norms
}

func queryVector(query string, idf map[string]float64) (map[string]float64, float64) {
	qTokens := tokenize(query)
	tf := termFrequency(qTokens)
	vec := make(map[string]float64, len(tf))
	var sumSquares float64
	for token, tVal := range tf {
		idfVal, ok := idf[token]
		if !ok {
			continue
		}
		w := tVal * idfVal
		vec[token] = w
		sumSquares += w * w
	}
	return vec, math.Sqrt(sumSquares)
}

func cosineSimilarity(qVec map[string]float64, qNorm float64, docVec map[string]float64, docNorm float64) float64 {
	if qNorm == 0 || docNorm == 0 {
		return 0
	}
	var dot float64
	for token, qWeight := range qVec {
		dot += qWeight * docVec[token]
	}
	return dot / (qNorm * docNorm)
}

func bestExcerpt(content, query string) string {
	queryTokens := tokenize(query)
	if len(queryTokens) == 0 {
		return trimForExcerpt(content, 220)
	}

	sentences := splitSentences(content)
	if len(sentences) == 0 {
		return trimForExcerpt(content, 220)
	}

	querySet := make(map[string]struct{}, len(queryTokens))
	for _, t := range queryTokens {
		querySet[t] = struct{}{}
	}

	bestIdx := -1
	bestScore := 0
	for idx, sent := range sentences {
		score := 0
		for _, token := range tokenize(sent) {
			if _, ok := querySet[token]; ok {
				score++
			}
		}
		if score > bestScore {
			bestScore = score
			bestIdx = idx
		}
	}

	if bestIdx == -1 {
		return trimForExcerpt(sentences[0], 220)
	}
	return trimForExcerpt(sentences[bestIdx], 220)
}

func splitSentences(text string) []string {
	seps := regexp.MustCompile(`[.!?]\s+`)
	raw := seps.Split(text, -1)
	s := make([]string, 0, len(raw))
	for _, part := range raw {
		part = strings.TrimSpace(part)
		if len(part) > 0 {
			s = append(s, part)
		}
	}
	return s
}

func trimForExcerpt(text string, max int) string {
	text = strings.TrimSpace(text)
	if len(text) <= max {
		return text
	}
	return strings.TrimSpace(text[:max]) + "..."
}
