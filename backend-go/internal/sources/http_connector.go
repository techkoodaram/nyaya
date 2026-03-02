package sources

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"nyaya-backend/internal/corpus"
)

type HTTPJSONConnector struct {
	name       string
	endpoint   string
	apiKey     string
	authType   string
	sourceType corpus.SourceType
	client     *http.Client
	userAgent  string
}

func NewHTTPJSONConnector(cfg ConnectorConfig, timeout time.Duration, userAgent string) (*HTTPJSONConnector, error) {
	if strings.TrimSpace(cfg.Name) == "" {
		return nil, errors.New("connector name is required")
	}
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return nil, fmt.Errorf("connector %s endpoint is required", cfg.Name)
	}
	sourceType := corpus.SourceType(strings.ToLower(strings.TrimSpace(cfg.Source)))
	if sourceType != corpus.SourceIPC && sourceType != corpus.SourceJudgment {
		return nil, fmt.Errorf("connector %s source must be ipc or judgment", cfg.Name)
	}
	if timeout <= 0 {
		timeout = 45 * time.Second
	}
	if strings.TrimSpace(userAgent) == "" {
		userAgent = "nyaya-backend/1.0 (+scheduled-sync)"
	}

	return &HTTPJSONConnector{
		name:       cfg.Name,
		endpoint:   cfg.Endpoint,
		apiKey:     cfg.APIKey,
		authType:   strings.ToLower(strings.TrimSpace(cfg.AuthType)),
		sourceType: sourceType,
		client:     &http.Client{Timeout: timeout},
		userAgent:  userAgent,
	}, nil
}

func (c *HTTPJSONConnector) Name() string {
	return c.name
}

func (c *HTTPJSONConnector) TargetSource() corpus.SourceType {
	return c.sourceType
}

func (c *HTTPJSONConnector) Fetch(ctx context.Context) ([]corpus.LegalDocument, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("%s build request: %w", c.name, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if strings.TrimSpace(c.apiKey) != "" {
		switch c.authType {
		case "query":
			q := req.URL.Query()
			q.Set("api_key", c.apiKey)
			req.URL.RawQuery = q.Encode()
		case "x-api-key":
			req.Header.Set("X-API-Key", c.apiKey)
		default:
			req.Header.Set("Authorization", "Bearer "+c.apiKey)
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s request failed: %w", c.name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		return nil, fmt.Errorf("%s returned status %d: %s", c.name, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("%s decode response: %w", c.name, err)
	}

	items := findRecordList(payload)
	out := make([]corpus.LegalDocument, 0, len(items))
	for _, item := range items {
		doc, ok := c.normalize(item)
		if ok {
			out = append(out, doc)
		}
	}
	return out, nil
}

func findRecordList(payload any) []map[string]any {
	switch x := payload.(type) {
	case []any:
		out := make([]map[string]any, 0, len(x))
		for _, row := range x {
			if m, ok := row.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	case map[string]any:
		keys := []string{"items", "results", "data", "documents", "records", "hits"}
		for _, key := range keys {
			v, ok := x[key]
			if !ok {
				continue
			}
			if arr, ok := v.([]any); ok {
				out := make([]map[string]any, 0, len(arr))
				for _, row := range arr {
					if m, ok := row.(map[string]any); ok {
						out = append(out, m)
					}
				}
				if len(out) > 0 {
					return out
				}
			}
		}
		return []map[string]any{x}
	default:
		return nil
	}
}

func (c *HTTPJSONConnector) normalize(item map[string]any) (corpus.LegalDocument, bool) {
	title := firstNonEmpty(item, "title", "case_name", "name", "heading", "section")
	text := firstNonEmpty(item, "text", "content", "body", "summary", "judgment", "judgement", "description")
	if strings.TrimSpace(text) == "" {
		return corpus.LegalDocument{}, false
	}
	if strings.TrimSpace(title) == "" {
		title = "untitled"
	}

	url := firstNonEmpty(item, "url", "source_url", "link")
	id := firstNonEmpty(item, "id", "doc_id", "slug")
	if strings.TrimSpace(id) == "" {
		id = "remote:" + c.name + ":" + checksumHex(title+"|"+text+"|"+url)
	}

	path := fmt.Sprintf("remote/%s/%s.json", strings.ToLower(c.name), sanitizeFileName(id))

	meta := map[string]string{
		"connector":  c.name,
		"law_name":   firstNonEmpty(item, "law_name", "act", "law"),
		"section":    firstNonEmpty(item, "section", "section_no"),
		"court":      firstNonEmpty(item, "court", "bench"),
		"date":       firstNonEmpty(item, "date", "judgment_date", "published_at"),
		"citation":   firstNonEmpty(item, "citation", "neutral_citation"),
		"source_url": url,
	}

	return corpus.LegalDocument{
		ID:        id,
		Source:    c.sourceType,
		Title:     title,
		Path:      path,
		Text:      strings.TrimSpace(text),
		WordCount: len(strings.Fields(text)),
		Checksum:  checksumHex(text),
		LawName:   meta["law_name"],
		Section:   meta["section"],
		Court:     meta["court"],
		Date:      meta["date"],
		Citation:  meta["citation"],
		SourceURL: meta["source_url"],
		Metadata:  meta,
	}, true
}

func firstNonEmpty(item map[string]any, keys ...string) string {
	for _, k := range keys {
		v, ok := item[k]
		if !ok {
			continue
		}
		s := strings.TrimSpace(toString(v))
		if s != "" {
			return s
		}
	}
	return ""
}

func toString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return fmt.Sprintf("%.0f", x)
	case int:
		return fmt.Sprintf("%d", x)
	default:
		return ""
	}
}

func checksumHex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func sanitizeFileName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return checksumHex(time.Now().String())
	}
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.ReplaceAll(value, ":", "-")
	value = strings.ReplaceAll(value, " ", "-")
	return value
}
