package corpus

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var tokenRegex = regexp.MustCompile(`[a-z0-9]+`)

type FileLoader struct {
	dataRoot string
}

func NewFileLoader(dataRoot string) *FileLoader {
	return &FileLoader{dataRoot: dataRoot}
}

func (l *FileLoader) LoadDocuments() ([]LegalDocument, error) {
	sources := map[SourceType]string{
		SourceIPC:      filepath.Join(l.dataRoot, "ipc"),
		SourceJudgment: filepath.Join(l.dataRoot, "judgments"),
	}

	var docs []LegalDocument
	for source, baseDir := range sources {
		entries, err := os.ReadDir(baseDir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("read source dir %s: %w", baseDir, err)
		}
		if len(entries) == 0 {
			continue
		}

		if err := filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			doc, ok, err := parseFile(path, source)
			if err != nil {
				return err
			}
			if ok {
				docs = append(docs, doc)
			}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("walk %s: %w", baseDir, err)
		}
	}

	if len(docs) == 0 {
		return nil, fmt.Errorf("no corpus documents found in %s (expected files under data/ipc and data/judgments)", l.dataRoot)
	}

	return docs, nil
}

func parseFile(path string, source SourceType) (LegalDocument, bool, error) {
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return LegalDocument{}, false, fmt.Errorf("read file %s: %w", path, err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	raw := strings.TrimSpace(string(contentBytes))
	if raw == "" {
		return LegalDocument{}, false, nil
	}

	title := filepath.Base(path)
	text := raw
	metadata := map[string]string{}

	if ext == ".json" {
		title, text, metadata = extractFromJSON(raw, filepath.Base(path))
	} else {
		title = titleFromText(raw, filepath.Base(path))
	}

	clean := normalizeWS(text)
	if clean == "" {
		return LegalDocument{}, false, nil
	}

	relPath := filepath.ToSlash(path)
	id := fmt.Sprintf("%s:%s", source, relPath)

	doc := LegalDocument{
		ID:        id,
		Source:    source,
		Title:     title,
		Path:      relPath,
		Text:      clean,
		WordCount: len(tokenize(clean)),
		Checksum:  sha256Hex(clean),
		Metadata:  metadata,
	}
	doc.LawName = metadata["law_name"]
	doc.Section = metadata["section"]
	doc.Court = metadata["court"]
	doc.Date = metadata["date"]
	doc.Citation = metadata["citation"]
	doc.SourceURL = metadata["source_url"]

	return doc, true, nil
}

func extractFromJSON(raw, fallbackTitle string) (title, content string, metadata map[string]string) {
	metadata = map[string]string{}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return titleFromText(raw, fallbackTitle), raw, metadata
	}

	title = fallbackTitle
	if s := firstNonEmptyString(parsed, "title", "section", "case_name", "name"); s != "" {
		title = s
	}

	copyIfPresent(metadata, parsed, "law_name", "section", "court", "date", "citation", "source_url")
	if metadata["source_url"] == "" {
		copyIfPresent(metadata, parsed, "url", "source")
	}

	fields := []string{
		"text", "content", "body", "judgment", "judgement",
		"facts", "decision", "holding", "summary", "description",
	}
	parts := make([]string, 0, len(fields))
	for _, k := range fields {
		if s := strings.TrimSpace(anyToString(parsed[k])); s != "" {
			parts = append(parts, s)
		}
	}
	if len(parts) == 0 {
		parts = append(parts, raw)
	}

	return strings.TrimSpace(title), strings.Join(parts, "\n"), metadata
}

func copyIfPresent(dst map[string]string, src map[string]any, keys ...string) {
	for _, k := range keys {
		if v := strings.TrimSpace(anyToString(src[k])); v != "" {
			if k == "url" || k == "source" {
				if dst["source_url"] == "" {
					dst["source_url"] = v
				}
				continue
			}
			dst[k] = v
		}
	}
}

func firstNonEmptyString(src map[string]any, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(anyToString(src[k])); v != "" {
			return v
		}
	}
	return ""
}

func anyToString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case fmt.Stringer:
		return x.String()
	default:
		return ""
	}
}

func titleFromText(text, fallback string) string {
	lines := strings.Split(text, "\n")
	for _, ln := range lines {
		c := strings.TrimSpace(ln)
		if c == "" {
			continue
		}
		if len(c) > 100 {
			return fallback
		}
		return c
	}
	return fallback
}

func normalizeWS(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

func tokenize(text string) []string {
	return tokenRegex.FindAllString(strings.ToLower(text), -1)
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
