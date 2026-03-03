package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"nyaya-backend/internal/rag"
	"nyaya-backend/internal/retrieval/vector"
	"nyaya-backend/internal/server"
	"nyaya-backend/internal/sources"
)

func main() {
	port := getEnv("PORT", "5000")
	dataRoot := getEnv("NYAYA_DATA_DIR", "./data")
	vectorBackend := getEnv("VECTOR_BACKEND", "")
	vectorDSN := getEnv("VECTOR_DSN", "")
	embeddingModel := getEnv("EMBEDDING_MODEL", "text-embedding-3-small")
	sourceSyncCfg := sources.ConfigFromEnv()
	var err error

	var syncService *sources.Service
	if sourceSyncCfg.Enabled && len(sourceSyncCfg.Connectors) > 0 {
		syncService, err = sources.NewService(sourceSyncCfg, dataRoot, nil)
		if err != nil {
			log.Fatalf("failed to initialize source sync service: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		err = syncService.SyncOnce(ctx)
		cancel()
		if err != nil {
			log.Printf("initial source sync failed, continuing with local corpus: %v", err)
		} else {
			log.Printf("initial source sync complete: %+v", syncService.Stats())
		}
	}

	index, err := rag.NewIndex(rag.Config{
		DataRoot: dataRoot,
		Vector: vector.Config{
			Backend:        vectorBackend,
			DSN:            vectorDSN,
			EmbeddingModel: embeddingModel,
		},
	})
	if err != nil {
		log.Fatalf("failed to build index: %v", err)
	}

	if syncService != nil {
		syncService.SetOnSynced(index.Reload)
	}

	srv := server.New(index)
	log.Printf("nyaya-backend started on :%s", port)
	log.Printf("index stats: %+v", index.Stats())
	if syncService != nil {
		syncService.Start(context.Background())
		log.Printf("source sync scheduler started: %+v", syncService.Stats())
	}

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
