package tfidf

import (
	"errors"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"

	"nyaya-backend/internal/corpus"
	"nyaya-backend/internal/retrieval"
)

var tokenRegex = regexp.MustCompile(`[a-z0-9]+`)

type Retriever struct {
	mu         sync.RWMutex
	chunks     []corpus.Chunk
	chunkTFIDF []map[string]float64
	chunkNorms []float64
	idf        map[string]float64
}

func New() *Retriever {
	return &Retriever{}
}

func (r *Retriever) Name() string {
	return "tfidf"
}

func (r *Retriever) Index(chunks []corpus.Chunk) error {
	if len(chunks) == 0 {
		return errors.New("no chunks to index")
	}

	idf, vectors, norms := buildVectors(chunks)

	r.mu.Lock()
	defer r.mu.Unlock()
	r.chunks = chunks
	r.idf = idf
	r.chunkTFIDF = vectors
	r.chunkNorms = norms
	return nil
}

func (r *Retriever) Search(query retrieval.Query) ([]retrieval.Result, error) {
	if strings.TrimSpace(query.Text) == "" {
		return nil, errors.New("query is required")
	}
	if query.TopK <= 0 {
		query.TopK = 3
	}
	if query.TopK > 15 {
		query.TopK = 15
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.chunks) == 0 {
		return nil, errors.New("index is empty")
	}

	qVec, qNorm := queryVector(query.Text, r.idf)
	if qNorm == 0 {
		return nil, nil
	}

	type scored struct {
		idx   int
		score float64
	}
	scoredChunks := make([]scored, 0, len(r.chunks))
	for i, chunkVec := range r.chunkTFIDF {
		score := cosineSimilarity(qVec, qNorm, chunkVec, r.chunkNorms[i])
		if score <= 0 {
			continue
		}
		scoredChunks = append(scoredChunks, scored{idx: i, score: score})
	}

	sort.Slice(scoredChunks, func(a, b int) bool {
		if scoredChunks[a].score == scoredChunks[b].score {
			return r.chunks[scoredChunks[a].idx].Title < r.chunks[scoredChunks[b].idx].Title
		}
		return scoredChunks[a].score > scoredChunks[b].score
	})

	if len(scoredChunks) > query.TopK {
		scoredChunks = scoredChunks[:query.TopK]
	}

	results := make([]retrieval.Result, 0, len(scoredChunks))
	for _, s := range scoredChunks {
		chunk := r.chunks[s.idx]
		results = append(results, retrieval.Result{
			Chunk:   chunk,
			Score:   math.Round(s.score*10000) / 10000,
			Excerpt: bestExcerpt(chunk.Text, query.Text),
		})
	}

	return results, nil
}

func (r *Retriever) Stats() map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return map[string]any{
		"retriever":       r.Name(),
		"indexedChunks":   len(r.chunks),
		"indexedKeywords": len(r.idf),
	}
}

func buildVectors(chunks []corpus.Chunk) (map[string]float64, []map[string]float64, []float64) {
	chunkTF := make([]map[string]float64, len(chunks))
	df := make(map[string]float64)

	for i, chunk := range chunks {
		tokens := tokenize(chunk.Text)
		tf := termFrequency(tokens)
		chunkTF[i] = tf
		for token := range tf {
			df[token]++
		}
	}

	total := float64(len(chunks))
	idf := make(map[string]float64, len(df))
	for token, docFreq := range df {
		idf[token] = math.Log((total+1)/(docFreq+1)) + 1
	}

	vectors := make([]map[string]float64, len(chunks))
	norms := make([]float64, len(chunks))
	for i, tf := range chunkTF {
		vec := make(map[string]float64, len(tf))
		var sumSquares float64
		for token, tVal := range tf {
			w := tVal * idf[token]
			vec[token] = w
			sumSquares += w * w
		}
		vectors[i] = vec
		norms[i] = math.Sqrt(sumSquares)
	}

	return idf, vectors, norms
}

func tokenize(text string) []string {
	return tokenRegex.FindAllString(strings.ToLower(text), -1)
}

func termFrequency(tokens []string) map[string]float64 {
	tf := make(map[string]float64)
	if len(tokens) == 0 {
		return tf
	}
	for _, t := range tokens {
		tf[t]++
	}
	total := float64(len(tokens))
	for token, count := range tf {
		tf[token] = count / total
	}
	return tf
}

func queryVector(query string, idf map[string]float64) (map[string]float64, float64) {
	tf := termFrequency(tokenize(query))
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

func cosineSimilarity(qVec map[string]float64, qNorm float64, cVec map[string]float64, cNorm float64) float64 {
	if qNorm == 0 || cNorm == 0 {
		return 0
	}
	var dot float64
	for token, qWeight := range qVec {
		dot += qWeight * cVec[token]
	}
	return dot / (qNorm * cNorm)
}

func bestExcerpt(content, query string) string {
	queryTokens := tokenize(query)
	if len(queryTokens) == 0 {
		return trim(content, 240)
	}

	sentences := splitSentences(content)
	if len(sentences) == 0 {
		return trim(content, 240)
	}

	querySet := make(map[string]struct{}, len(queryTokens))
	for _, t := range queryTokens {
		querySet[t] = struct{}{}
	}

	bestIdx := -1
	bestScore := 0
	for i, sentence := range sentences {
		score := 0
		for _, token := range tokenize(sentence) {
			if _, ok := querySet[token]; ok {
				score++
			}
		}
		if score > bestScore {
			bestScore = score
			bestIdx = i
		}
	}

	if bestIdx == -1 {
		return trim(sentences[0], 240)
	}
	return trim(sentences[bestIdx], 240)
}

func splitSentences(text string) []string {
	separators := regexp.MustCompile(`[.!?]\s+`)
	parts := separators.Split(text, -1)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func trim(text string, max int) string {
	text = strings.TrimSpace(text)
	if len(text) <= max {
		return text
	}
	return strings.TrimSpace(text[:max]) + "..."
}
