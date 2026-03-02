package hybrid

import (
	"nyaya-backend/internal/corpus"
	"nyaya-backend/internal/retrieval"
)

type Retriever struct {
	primary  retrieval.Retriever
	semantic retrieval.Retriever
}

func New(primary, semantic retrieval.Retriever) *Retriever {
	return &Retriever{primary: primary, semantic: semantic}
}

func (r *Retriever) Name() string {
	return "hybrid"
}

func (r *Retriever) Index(chunks []corpus.Chunk) error {
	if err := r.primary.Index(chunks); err != nil {
		return err
	}
	if r.semantic != nil {
		_ = r.semantic.Index(chunks)
	}
	return nil
}

func (r *Retriever) Search(query retrieval.Query) ([]retrieval.Result, error) {
	lexical, err := r.primary.Search(query)
	if err != nil {
		return nil, err
	}
	if r.semantic == nil {
		return lexical, nil
	}
	semantic, err := r.semantic.Search(query)
	if err != nil || len(semantic) == 0 {
		return lexical, nil
	}
	return mergeScores(lexical, semantic), nil
}

func (r *Retriever) Stats() map[string]any {
	stats := map[string]any{
		"retriever": r.Name(),
		"primary":   r.primary.Name(),
	}
	if r.semantic != nil {
		stats["semantic"] = r.semantic.Name()
		stats["semanticStats"] = r.semantic.Stats()
	}
	stats["primaryStats"] = r.primary.Stats()
	return stats
}

func mergeScores(lexical, semantic []retrieval.Result) []retrieval.Result {
	byChunkID := make(map[string]retrieval.Result, len(lexical)+len(semantic))
	for _, item := range lexical {
		item.Score = item.Score * 0.75
		byChunkID[item.Chunk.ID] = item
	}
	for _, item := range semantic {
		if current, ok := byChunkID[item.Chunk.ID]; ok {
			current.Score += item.Score * 0.25
			byChunkID[item.Chunk.ID] = current
			continue
		}
		item.Score = item.Score * 0.25
		byChunkID[item.Chunk.ID] = item
	}

	out := make([]retrieval.Result, 0, len(byChunkID))
	for _, v := range byChunkID {
		out = append(out, v)
	}
	retrieval.SortResults(out)
	return out
}
