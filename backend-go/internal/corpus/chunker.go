package corpus

import (
	"fmt"
	"unicode/utf8"
)

type ChunkerConfig struct {
	MaxChars int
	Overlap  int
}

type Chunker struct {
	cfg ChunkerConfig
}

func NewChunker(cfg ChunkerConfig) *Chunker {
	if cfg.MaxChars <= 0 {
		cfg.MaxChars = 900
	}
	if cfg.Overlap < 0 {
		cfg.Overlap = 0
	}
	if cfg.Overlap >= cfg.MaxChars {
		cfg.Overlap = cfg.MaxChars / 3
	}
	return &Chunker{cfg: cfg}
}

func (c *Chunker) ChunkDocuments(docs []LegalDocument) []Chunk {
	chunks := make([]Chunk, 0, len(docs))
	for _, doc := range docs {
		chunks = append(chunks, c.chunkDocument(doc)...)
	}
	return chunks
}

func (c *Chunker) chunkDocument(doc LegalDocument) []Chunk {
	if doc.Text == "" {
		return nil
	}
	text := doc.Text
	if utf8.RuneCountInString(text) <= c.cfg.MaxChars {
		return []Chunk{
			c.makeChunk(doc, 0, 0, len(text), text),
		}
	}

	step := c.cfg.MaxChars - c.cfg.Overlap
	if step <= 0 {
		step = c.cfg.MaxChars
	}

	chunks := make([]Chunk, 0, (len(text)/step)+1)
	chunkIdx := 0
	for start := 0; start < len(text); start += step {
		end := start + c.cfg.MaxChars
		if end > len(text) {
			end = len(text)
		}
		fragment := text[start:end]
		chunks = append(chunks, c.makeChunk(doc, chunkIdx, start, end, fragment))
		chunkIdx++
		if end == len(text) {
			break
		}
	}
	return chunks
}

func (c *Chunker) makeChunk(doc LegalDocument, idx, start, end int, text string) Chunk {
	return Chunk{
		ID:          fmt.Sprintf("%s#%d", doc.ID, idx),
		DocumentID:  doc.ID,
		Source:      doc.Source,
		Title:       doc.Title,
		Path:        doc.Path,
		Text:        text,
		ChunkIndex:  idx,
		StartOffset: start,
		EndOffset:   end,
		LawName:     doc.LawName,
		Section:     doc.Section,
		Court:       doc.Court,
		Date:        doc.Date,
		Citation:    doc.Citation,
		SourceURL:   doc.SourceURL,
		Metadata:    doc.Metadata,
	}
}
