# nyaya-client

React frontend for the Nyaya legal assistant demo.
It sends user queries to the backend RAG API and renders the generated answer and retrieved references.

## Tech stack

- React 18
- Vite 5

## Prerequisites

- Node.js 20+
- npm 10+
- Running `nyaya-backend` service

## Environment

Create `.env` from `.env.example`:

```bash
VITE_API_URL=/api/retrieve
```

Default value uses the Vite dev proxy to forward requests to `http://localhost:5000`.

## Install

```bash
npm install
```

## Run (development)

Start backend in one terminal:

```bash
cd ../nyaya-backend
go run ./cmd/server
```

Start frontend in another terminal:

```bash
npm run dev
```

Open the URL shown by Vite (usually `http://localhost:5173`).

## Build

```bash
npm run build
```

## API contract used by client

Endpoint:

- `POST /api/retrieve`

Request body:

```json
{
  "query": "difference between culpable homicide and murder",
  "topK": 3
}
```

Expected response fields:

- `answer`
- `references` (array)
- `retrievedCount`
- `generatedAt`

## Demo sample queries

- `What is the difference between culpable homicide and murder under IPC?`
- `Explain IPC Section 299 with a simple example.`
- `How does Virsa Singh v. State of Punjab interpret intention and injury?`

## Troubleshooting

- If `vite is not recognized`, run `npm install` again in this folder.
- If frontend cannot reach backend, verify backend is running on port `5000` or adjust `VITE_API_URL`.
