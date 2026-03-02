package sources

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"nyaya-backend/internal/corpus"
)

type FileWriter struct {
	dataRoot string
}

func NewFileWriter(dataRoot string) *FileWriter {
	return &FileWriter{dataRoot: dataRoot}
}

func (w *FileWriter) Persist(connectorName string, source corpus.SourceType, docs []corpus.LegalDocument) (int, error) {
	if len(docs) == 0 {
		return 0, nil
	}
	targetDir := filepath.Join(w.dataRoot, sourceDir(source))
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return 0, fmt.Errorf("create target dir %s: %w", targetDir, err)
	}

	manifest := make(map[string]struct{}, len(docs))
	written := 0
	for _, doc := range docs {
		fileName := fmt.Sprintf("remote-%s-%s.json", sanitizeFileName(connectorName), sanitizeFileName(doc.ID))
		fullPath := filepath.Join(targetDir, fileName)
		payload := map[string]any{
			"id":         doc.ID,
			"title":      doc.Title,
			"text":       doc.Text,
			"law_name":   doc.LawName,
			"section":    doc.Section,
			"court":      doc.Court,
			"date":       doc.Date,
			"citation":   doc.Citation,
			"source_url": doc.SourceURL,
			"metadata":   doc.Metadata,
		}
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return written, fmt.Errorf("marshal %s: %w", doc.ID, err)
		}
		data = append(data, '\n')
		if err := os.WriteFile(fullPath, data, 0o644); err != nil {
			return written, fmt.Errorf("write %s: %w", fullPath, err)
		}
		manifest[fileName] = struct{}{}
		written++
	}

	prefix := "remote-" + sanitizeFileName(connectorName) + "-"
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return written, nil
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ".json") {
			continue
		}
		if _, ok := manifest[name]; ok {
			continue
		}
		_ = os.Remove(filepath.Join(targetDir, name))
	}

	return written, nil
}

func sourceDir(source corpus.SourceType) string {
	switch source {
	case corpus.SourceIPC:
		return "ipc"
	default:
		return "judgments"
	}
}
