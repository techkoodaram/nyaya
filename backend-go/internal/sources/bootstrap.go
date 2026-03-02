package sources

import (
	"os"
	"strconv"
	"strings"
	"time"
)

func ConfigFromEnv() Config {
	cfg := Config{
		Enabled:        parseBoolEnv("SOURCE_SYNC_ENABLED", false),
		Interval:       parseDurationEnv("SOURCE_SYNC_INTERVAL", 24*time.Hour),
		RequestTimeout: parseDurationEnv("SOURCE_SYNC_TIMEOUT", 45*time.Second),
		UserAgent:      env("SOURCE_SYNC_USER_AGENT", "nyaya-backend/1.0 (+scheduled-sync)"),
	}

	connectors := []ConnectorConfig{
		{
			Name:     "supreme-court",
			Endpoint: strings.TrimSpace(os.Getenv("SUPREME_COURT_FEED_URL")),
			APIKey:   strings.TrimSpace(os.Getenv("SUPREME_COURT_API_KEY")),
			Source:   "judgment",
			AuthType: strings.TrimSpace(os.Getenv("SUPREME_COURT_AUTH_TYPE")),
		},
		{
			Name:     "ecourts",
			Endpoint: strings.TrimSpace(os.Getenv("ECOURTS_FEED_URL")),
			APIKey:   strings.TrimSpace(os.Getenv("ECOURTS_API_KEY")),
			Source:   "judgment",
			AuthType: strings.TrimSpace(os.Getenv("ECOURTS_AUTH_TYPE")),
		},
		{
			Name:     "official-law",
			Endpoint: strings.TrimSpace(os.Getenv("OFFICIAL_LAW_FEED_URL")),
			APIKey:   strings.TrimSpace(os.Getenv("OFFICIAL_LAW_API_KEY")),
			Source:   "ipc",
			AuthType: strings.TrimSpace(os.Getenv("OFFICIAL_LAW_AUTH_TYPE")),
		},
	}

	cfg.Connectors = make([]ConnectorConfig, 0, len(connectors))
	for _, c := range connectors {
		if c.Endpoint == "" {
			continue
		}
		cfg.Connectors = append(cfg.Connectors, c)
	}
	return cfg
}

func env(key, fallback string) string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	return val
}

func parseBoolEnv(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return v
}

func parseDurationEnv(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}
