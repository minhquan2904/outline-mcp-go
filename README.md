# outline-mcp-go

Read-only MCP server for Outline wiki, written in Go using the official MCP SDK.

## Setup

```bash
go mod tidy
```

## Build

```bash
make build
# or
go build -o outline-mcp-go
```

## Test

```bash
make test
# or
go test -v ./...
```

## Run

```bash
export OUTLINE_API_URL=https://outline.example.com
export OUTLINE_API_KEY=your_api_key
./outline-mcp-go
```

## Tools

- `search_documents` — Full-text search
- `search_document_titles` — Search titles only
- `get_document` — Read one document
- `list_collections` — List wiki spaces
- `list_documents` — Browse documents
- `list_revisions` — List document versions
- `get_revision` — Read an older version

## Architecture

- `types.go` — Outline API response types
- `client.go` — HTTP RPC client (POST /api/{method})
- `format.go` — Output rendering
- `tools.go` — MCP tool definitions & handlers
- `main.go` — Server entry point

## Implementation Notes

- Zero dependencies beyond MCP SDK
- Manual parameter validation (limit, offset, contextLimit)
- API key redaction in errors
- Timeout: 30s per request
- Retry on 429 (rate limit): max 10s wait

## TODO

- [ ] Full JSON Schema for tool parameters
- [ ] Integration tests with mock Outline API
- [ ] Handle pagination in list operations
- [ ] Add list_comments tool
