package corpus

import "time"

type SourceType string

const (
	SourceIPC      SourceType = "ipc"
	SourceJudgment SourceType = "judgment"
)

type LegalDocument struct {
	ID        string            `json:"id"`
	Source    SourceType        `json:"source"`
	Title     string            `json:"title"`
	Path      string            `json:"path"`
	Text      string            `json:"text"`
	WordCount int               `json:"wordCount"`
	Checksum  string            `json:"checksum"`
	LawName   string            `json:"lawName,omitempty"`
	Section   string            `json:"section,omitempty"`
	Court     string            `json:"court,omitempty"`
	Date      string            `json:"date,omitempty"`
	Citation  string            `json:"citation,omitempty"`
	SourceURL string            `json:"sourceUrl,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type Chunk struct {
	ID          string            `json:"id"`
	DocumentID  string            `json:"documentId"`
	Source      SourceType        `json:"source"`
	Title       string            `json:"title"`
	Path        string            `json:"path"`
	Text        string            `json:"text"`
	ChunkIndex  int               `json:"chunkIndex"`
	StartOffset int               `json:"startOffset"`
	EndOffset   int               `json:"endOffset"`
	LawName     string            `json:"lawName,omitempty"`
	Section     string            `json:"section,omitempty"`
	Court       string            `json:"court,omitempty"`
	Date        string            `json:"date,omitempty"`
	Citation    string            `json:"citation,omitempty"`
	SourceURL   string            `json:"sourceUrl,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type Snapshot struct {
	Documents  []LegalDocument
	Chunks     []Chunk
	LoadedAt   time.Time
	IndexedAt  time.Time
	DataRoot   string
	DocByID    map[string]LegalDocument
	ChunkByID  map[string]Chunk
	DocCount   int
	ChunkCount int
}
