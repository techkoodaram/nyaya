# nyaya-backend

Go backend for Nyaya legal retrieval.
It implements a lightweight RAG-style pipeline over IPC sections and court judgments.

## Features

- Loads corpus from `data/ipc` and `data/judgments`
- Builds in-memory TF-IDF vectors
- Retrieves top relevant passages for each query
- Returns generated answer grounded in retrieved references
- Supports corpus reload via API without restarting server

## Tech stack

- Go 1.24.4
- Standard `net/http` server
- In-memory indexing (no external database required)

## Project structure

- `cmd/server/main.go` - app entry point
- `internal/server` - HTTP routes and handlers
- `internal/rag` - corpus loading, indexing, retrieval, answer synthesis
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
