package vector

import (
	"errors"

	"nyaya-backend/internal/corpus"
	"nyaya-backend/internal/retrieval"
)

type Retriever struct {
	cfg Config
}

func NewRetriever(cfg Config) *Retriever {
	return &Retriever{cfg: cfg}
}

func (r *Retriever) Name() string {
	return "vector"
}

func (r *Retriever) Index(_ []corpus.Chunk) error {
	if !r.cfg.Enabled() {
		return nil
	}
	return errors.New("vector retriever wiring is configured but no backend implementation is registered yet")
}

func (r *Retriever) Search(_ retrieval.Query) ([]retrieval.Result, error) {
	if !r.cfg.Enabled() {
		return nil, nil
	}
	return nil, errors.New("vector retriever is not implemented")
}

func (r *Retriever) Stats() map[string]any {
	stats := r.cfg.Stats()
	stats["retriever"] = r.Name()
	stats["implemented"] = false
	return stats
}
