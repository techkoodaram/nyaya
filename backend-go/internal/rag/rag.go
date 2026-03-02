package rag

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"nyaya-backend/internal/corpus"
	"nyaya-backend/internal/retrieval"
	"nyaya-backend/internal/retrieval/hybrid"
	"nyaya-backend/internal/retrieval/tfidf"
	"nyaya-backend/internal/retrieval/vector"
)

type Config struct {
	DataRoot string
	Vector   vector.Config
}

type SearchResult struct {
	ID        string  `json:"id"`
	Source    string  `json:"source"`
	Title     string  `json:"title"`
	Path      string  `json:"path"`
	Score     float64 `json:"score"`
	Excerpt   string  `json:"excerpt"`
	LawName   string  `json:"lawName,omitempty"`
	Section   string  `json:"section,omitempty"`
	Court     string  `json:"court,omitempty"`
	Date      string  `json:"date,omitempty"`
	Citation  string  `json:"citation,omitempty"`
	SourceURL string  `json:"sourceUrl,omitempty"`
}

type Index struct {
	mu       sync.RWMutex
	cfg      Config
	loader   *corpus.FileLoader
	chunker  *corpus.Chunker
	retrieve retrieval.Retriever

	docs       []corpus.LegalDocument
	chunks     []corpus.Chunk
	lastLoaded time.Time
}

func NewIndex(cfg Config) (*Index, error) {
	if strings.TrimSpace(cfg.DataRoot) == "" {
		return nil, errors.New("data root is required")
	}

	lexical := tfidf.New()
	vectorRetriever := vector.NewRetriever(cfg.Vector)
	idx := &Index{
		cfg:      cfg,
		loader:   corpus.NewFileLoader(cfg.DataRoot),
		chunker:  corpus.NewChunker(corpus.ChunkerConfig{MaxChars: 900, Overlap: 120}),
		retrieve: hybrid.New(lexical, vectorRetriever),
	}
	if err := idx.Reload(); err != nil {
		return nil, err
	}
	return idx, nil
}

func (i *Index) Reload() error {
	docs, err := i.loader.LoadDocuments()
	if err != nil {
		return err
	}
	chunks := i.chunker.ChunkDocuments(docs)
	if len(chunks) == 0 {
		return fmt.Errorf("no indexable chunks found in %s", i.cfg.DataRoot)
	}
	if err := i.retrieve.Index(chunks); err != nil {
		return err
	}

	i.mu.Lock()
	defer i.mu.Unlock()
	i.docs = docs
	i.chunks = chunks
	i.lastLoaded = time.Now()
	return nil
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

	results, err := i.retrieve.Search(retrieval.Query{Text: query, TopK: topK})
	if err != nil {
		return nil, err
	}
	if len(results) > topK {
		results = results[:topK]
	}

	out := make([]SearchResult, 0, len(results))
	for _, r := range results {
		out = append(out, SearchResult{
			ID:        r.Chunk.DocumentID,
			Source:    string(r.Chunk.Source),
			Title:     r.Chunk.Title,
			Path:      r.Chunk.Path,
			Score:     math.Round(r.Score*10000) / 10000,
			Excerpt:   r.Excerpt,
			LawName:   r.Chunk.LawName,
			Section:   r.Chunk.Section,
			Court:     r.Chunk.Court,
			Date:      r.Chunk.Date,
			Citation:  r.Chunk.Citation,
			SourceURL: r.Chunk.SourceURL,
		})
	}
	return out, nil
}

func (i *Index) Stats() map[string]any {
	i.mu.RLock()
	defer i.mu.RUnlock()

	var ipcCount int
	var judgmentCount int
	for _, d := range i.docs {
		switch d.Source {
		case corpus.SourceIPC:
			ipcCount++
		case corpus.SourceJudgment:
			judgmentCount++
		}
	}

	return map[string]any{
		"documents":        len(i.docs),
		"chunks":           len(i.chunks),
		"ipcDocuments":     ipcCount,
		"judgmentDocs":     judgmentCount,
		"lastLoaded":       i.lastLoaded.Format(time.RFC3339),
		"dataRoot":         i.cfg.DataRoot,
		"retriever":        i.retrieve.Name(),
		"retrieverStats":   i.retrieve.Stats(),
		"vectorConfigured": i.cfg.Vector.Enabled(),
	}
}

func GenerateAnswer(query string, hits []SearchResult) string {
	if len(hits) == 0 {
		return "No relevant legal references were found for this query in the indexed corpus."
	}

	var b strings.Builder
	b.WriteString("Retrieved legal references relevant to your query:\n\n")
	for idx, h := range hits {
		b.WriteString(fmt.Sprintf("%d. %s (%s)\n", idx+1, h.Title, strings.ToUpper(h.Source)))
		if h.Citation != "" || h.Section != "" {
			meta := make([]string, 0, 2)
			if h.Section != "" {
				meta = append(meta, "Section: "+h.Section)
			}
			if h.Citation != "" {
				meta = append(meta, "Citation: "+h.Citation)
			}
			b.WriteString("   " + strings.Join(meta, " | ") + "\n")
		}
		b.WriteString(fmt.Sprintf("   %s\n", h.Excerpt))
	}
	b.WriteString("\nValidate against the full text and latest law before relying on the result.")
	return b.String()
}
