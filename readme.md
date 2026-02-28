# Nyaya

Nyaya is a lightweight legal assistant demo with:

- A Go backend that performs retrieval over IPC sections and judgments
- A React + Vite frontend to submit queries and display grounded answers

## Repository layout

- `backend-go` - RAG-style retrieval API (`/health`, `/api/retrieve`, `/api/ingest`)
- `client` - web UI for querying the backend

## Prerequisites

- Go 1.24+
- Node.js 20+
- npm 10+

## Quick start

### 1. Start backend

```powershell
cd backend-go
go run ./cmd/server
```

Defaults:

- `PORT=5000`
- `NYAYA_DATA_DIR=./data`

### 2. Start frontend

Open a new terminal:

```powershell
cd client
npm install
npm run dev
```

Vite dev server usually runs at `http://localhost:5173` and proxies API requests to backend port `5000`.

## API summary

- `GET /health` - service health and index stats
- `POST /api/retrieve` - retrieve answer for query
- `POST /api/ingest` - reload corpus files and rebuild in-memory index

Sample retrieve request:

```json
{
  "query": "difference between culpable homicide and murder",
  "topK": 3
}
```

## Corpus data

Backend loads corpus from:

- `backend-go/data/ipc`
- `backend-go/data/judgments`

Supported file types: `.txt`, `.md`, `.json`.

## Build frontend

```powershell
cd client
npm run build
```
