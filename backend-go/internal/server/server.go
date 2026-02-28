package server

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"nyaya-backend/internal/rag"
)

type RetrieveRequest struct {
	Query string `json:"query"`
	TopK  int    `json:"topK"`
}

type RetrieveResponse struct {
	Query          string             `json:"query"`
	Answer         string             `json:"answer"`
	References     []rag.SearchResult `json:"references"`
	RetrievedCount int                `json:"retrievedCount"`
	GeneratedAt    string             `json:"generatedAt"`
}

type APIError struct {
	Error string `json:"error"`
}

type Server struct {
	index *rag.Index
}

func New(idx *rag.Index) *Server {
	return &Server{index: idx}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/retrieve", s.handleRetrieve)
	mux.HandleFunc("/api/ingest", s.handleIngest)
	return withCORS(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"stats":  s.index.Stats(),
		"time":   time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleRetrieve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}

	var req RetrieveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIError{Error: "invalid JSON body"})
		return
	}

	results, err := s.index.Search(req.Query, req.TopK)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
		return
	}

	resp := RetrieveResponse{
		Query:          req.Query,
		Answer:         rag.GenerateAnswer(req.Query, results),
		References:     results,
		RetrievedCount: len(results),
		GeneratedAt:    time.Now().Format(time.RFC3339),
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIError{Error: "method not allowed"})
		return
	}

	if err := s.index.Reload(); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIError{Error: err.Error()})
		return
	}
	log.Printf("corpus reloaded: %+v", s.index.Stats())
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "corpus reloaded",
		"stats":   s.index.Stats(),
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
