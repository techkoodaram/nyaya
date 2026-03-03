package sources

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type Service struct {
	cfg        Config
	connectors []Connector
	writer     *FileWriter
	onSynced   func() error

	mu            sync.RWMutex
	lastRun       time.Time
	lastSuccess   time.Time
	lastError     string
	lastConnector []ConnectorResult
}

func NewService(cfg Config, dataRoot string, onSynced func() error) (*Service, error) {
	if cfg.Interval <= 0 {
		cfg.Interval = 24 * time.Hour
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = 45 * time.Second
	}
	connectors := make([]Connector, 0, len(cfg.Connectors))
	for _, cc := range cfg.Connectors {
		c, err := NewHTTPJSONConnector(cc, cfg.RequestTimeout, cfg.UserAgent)
		if err != nil {
			return nil, err
		}
		connectors = append(connectors, c)
	}
	return &Service{
		cfg:        cfg,
		connectors: connectors,
		writer:     NewFileWriter(dataRoot),
		onSynced:   onSynced,
	}, nil
}

func (s *Service) SetOnSynced(fn func() error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onSynced = fn
}

func (s *Service) SyncOnce(ctx context.Context) error {
	s.mu.Lock()
	s.lastRun = time.Now()
	s.lastError = ""
	s.mu.Unlock()

	results := make([]ConnectorResult, 0, len(s.connectors))
	for _, connector := range s.connectors {
		docs, err := connector.Fetch(ctx)
		if err != nil {
			s.recordError(fmt.Sprintf("%s: %v", connector.Name(), err))
			return err
		}
		persisted, err := s.writer.Persist(connector.Name(), connector.TargetSource(), docs)
		if err != nil {
			s.recordError(fmt.Sprintf("%s persist: %v", connector.Name(), err))
			return err
		}
		results = append(results, ConnectorResult{
			Name:      connector.Name(),
			Source:    string(connector.TargetSource()),
			Fetched:   len(docs),
			Persisted: persisted,
		})
	}

	if s.onSynced != nil {
		if err := s.onSynced(); err != nil {
			s.recordError(fmt.Sprintf("reload index: %v", err))
			return err
		}
	}

	s.mu.Lock()
	s.lastSuccess = time.Now()
	s.lastConnector = results
	s.lastError = ""
	s.mu.Unlock()

	return nil
}

func (s *Service) Start(ctx context.Context) {
	if !s.cfg.Enabled || len(s.connectors) == 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(s.cfg.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.SyncOnce(ctx); err != nil {
					log.Printf("scheduled sync failed: %v", err)
					continue
				}
				log.Printf("scheduled sync succeeded: %+v", s.Stats())
			}
		}
	}()
}

func (s *Service) Stats() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]any{
		"enabled":       s.cfg.Enabled,
		"interval":      s.cfg.Interval.String(),
		"connectors":    len(s.connectors),
		"lastRun":       formatTime(s.lastRun),
		"lastSuccess":   formatTime(s.lastSuccess),
		"lastError":     s.lastError,
		"connectorRuns": s.lastConnector,
	}
}

func (s *Service) recordError(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastError = msg
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
