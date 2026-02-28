package main

import (
	"log"
	"net/http"
	"os"

	"nyaya-backend/internal/rag"
	"nyaya-backend/internal/server"
)

func main() {
	port := getEnv("PORT", "5000")
	dataRoot := getEnv("NYAYA_DATA_DIR", "./data")

	index, err := rag.NewIndex(dataRoot)
	if err != nil {
		log.Fatalf("failed to build index: %v", err)
	}

	srv := server.New(index)
	log.Printf("nyaya-backend started on :%s", port)
	log.Printf("index stats: %+v", index.Stats())

	if err := http.ListenAndServe(":"+port, srv.Routes()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}
