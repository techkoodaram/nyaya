package sources

import "time"

type Config struct {
	Enabled        bool
	Interval       time.Duration
	RequestTimeout time.Duration
	UserAgent      string
	Connectors     []ConnectorConfig
}

type ConnectorConfig struct {
	Name      string
	Endpoint  string
	APIKey    string
	Source    string
	AuthType  string
	TargetDir string
}
