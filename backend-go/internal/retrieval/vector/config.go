package vector

import "strings"

type Config struct {
	Backend        string
	DSN            string
	EmbeddingModel string
}

func (c Config) Enabled() bool {
	return strings.TrimSpace(c.Backend) != ""
}

func (c Config) Stats() map[string]any {
	return map[string]any{
		"backend":        c.Backend,
		"dsnConfigured":  strings.TrimSpace(c.DSN) != "",
		"embeddingModel": c.EmbeddingModel,
	}
}
