package retrieval

import "nyaya-backend/internal/corpus"

type Query struct {
	Text    string
	TopK    int
	Filters map[string]string
}

type Result struct {
	Chunk   corpus.Chunk `json:"chunk"`
	Score   float64      `json:"score"`
	Excerpt string       `json:"excerpt"`
}

type Retriever interface {
	Name() string
	Index(chunks []corpus.Chunk) error
	Search(query Query) ([]Result, error)
	Stats() map[string]any
}
