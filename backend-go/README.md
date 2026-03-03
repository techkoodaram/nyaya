# nyaya-backend

Go backend for Nyaya legal retrieval.
It implements a lightweight RAG-style pipeline over IPC sections and court judgments.

## Features

- Loads legal corpus from `data/ipc` and `data/judgments`
- Normalizes documents into a canonical legal schema (law/section/court/citation/url fields)
- Chunks long documents for retrieval-friendly indexing
- Uses pluggable retrieval with lexical TF-IDF active by default
- Includes vector-retriever wiring via env config for future embedding backends
- Supports corpus reload via API without restarting server

## Tech stack

- Go 1.24.4
- Standard `net/http` server
- In-memory indexing (no external database required)

## Project structure

- `cmd/server/main.go` - app entry point
- `internal/server` - HTTP routes and handlers
- `internal/rag` - orchestration layer: ingestion + chunking + retriever + answer synthesis
- `internal/corpus` - canonical document model and filesystem ingestion/chunking
- `internal/retrieval` - retriever contracts and implementations (hybrid, tfidf, vector wiring)
- `data/ipc` - IPC source documents
- `data/judgments` - court judgment source documents

## Prerequisites

- Go 1.24+

## Run

```bash
go run ./cmd/server
```

Environment variables:

- `PORT` (default: `5000`)
- `NYAYA_DATA_DIR` (default: `./data`)
- `VECTOR_BACKEND` (optional, e.g. `pgvector`)
- `VECTOR_DSN` (optional, connection string for vector store)
- `EMBEDDING_MODEL` (optional, default: `text-embedding-3-small`)
- `SOURCE_SYNC_ENABLED` (optional, `true/false`, default `false`)
- `SOURCE_SYNC_INTERVAL` (optional, Go duration, default `24h`)
- `SOURCE_SYNC_TIMEOUT` (optional, Go duration, default `45s`)
- `SUPREME_COURT_FEED_URL` (optional JSON endpoint for Supreme Court judgments)
- `SUPREME_COURT_API_KEY` / `SUPREME_COURT_AUTH_TYPE` (optional auth)
- `ECOURTS_FEED_URL` (optional JSON endpoint for eCourts judgments)
- `ECOURTS_API_KEY` / `ECOURTS_AUTH_TYPE` (optional auth)
- `OFFICIAL_LAW_FEED_URL` (optional JSON endpoint for official law/act sections)
- `OFFICIAL_LAW_API_KEY` / `OFFICIAL_LAW_AUTH_TYPE` (optional auth)

Windows example:

```powershell
$env:PORT="5001"; go run ./cmd/server
```

## API endpoints

### `GET /health`

Returns service status and index statistics.

Example response (shape):

```json
{
  "status": "ok",
  "stats": {
    "documents": 4,
    "ipcDocuments": 2,
    "judgmentDocs": 2,
    "indexedKeywords": 87
  },
  "time": "2026-02-28T12:38:34+05:30"
}
```

### `POST /api/retrieve`

Request body:

```json
{
  "query": "difference between culpable homicide and murder",
  "topK": 3
}
```

Response fields:

- `query`
- `answer`
- `references`
- `retrievedCount`
- `generatedAt`

### `POST /api/ingest`

Reloads all files from data directory and rebuilds index.

## Corpus format

Supported file types:

- `.txt`
- `.md`
- `.json`

Place files under:

- `data/ipc`
- `data/judgments`

For JSON files, common fields such as `title`, `content`, `text`, `judgment`, `summary`, `facts`, `decision` are extracted automatically.
If present, metadata fields such as `law_name`, `section`, `court`, `date`, `citation`, `source_url` are preserved and surfaced in retrieval results.

## Scheduled source sync

When `SOURCE_SYNC_ENABLED=true`, backend can pull remote legal data from configured JSON feeds and write normalized records to:

- `data/judgments` (Supreme Court + eCourts connectors)
- `data/ipc` (official law connector)

Startup behavior:

- Attempts one initial sync (best effort)
- Builds index from local + synced files
- Starts periodic sync ticker
- Reloads retrieval index automatically after successful sync

## Quick test with curl

Health:

```bash
curl http://localhost:5000/health
```

Retrieve:

```bash
curl -X POST http://localhost:5000/api/retrieve \
  -H "Content-Type: application/json" \
  -d "{\"query\":\"What is the difference between Section 299 and 300?\",\"topK\":3}"
```

Reload corpus:

```bash
curl -X POST http://localhost:5000/api/ingest
```

## Troubleshooting

- `bind: Only one usage of each socket address...`:
  another process is already using that port. Stop existing process or change `PORT`.
- `no corpus documents found`:
  ensure files exist under `data/ipc` and `data/judgments`.
