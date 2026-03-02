package sources

import (
	"context"

	"nyaya-backend/internal/corpus"
)

type Connector interface {
	Name() string
	TargetSource() corpus.SourceType
	Fetch(ctx context.Context) ([]corpus.LegalDocument, error)
}

type ConnectorResult struct {
	Name      string `json:"name"`
	Source    string `json:"source"`
	Fetched   int    `json:"fetched"`
	Persisted int    `json:"persisted"`
}
