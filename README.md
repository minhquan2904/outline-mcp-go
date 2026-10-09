# outline-mcp-go

MCP server (read and write) for Outline wiki, written in Go using the official MCP SDK.

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

## Docker

```bash
cp .env.example .env        # điền OUTLINE_API_URL và OUTLINE_API_KEY
docker compose build
docker compose run --rm -T outline-mcp
```

Server nói chuyện qua stdio nên không có cổng để publish. Để dùng với MCP client,
trỏ lệnh khởi động của client tới `docker compose run --rm -T outline-mcp`
(đặt thư mục làm việc là thư mục repo này).

## Tools

- `search_documents` — Full-text search
- `search_document_titles` — Search titles only
- `get_document` — Read one document
- `list_collections` — List wiki spaces
- `list_documents` — Browse documents
- `list_revisions` — List document versions
- `get_revision` — Read an older version
- `list_comments` — List review comments on a document
- `create_document` — Create a document
- `update_document` — Edit a document (append by default)
- `move_document` — Move a document to another collection or parent
- `archive_document` — Archive a document
- `delete_document` — Move a document to the trash (never permanent)

Writes are bounded by the API key's own permissions. Set `OUTLINE_WRITE_DOC_ID` to
confine every write to `update_document` on that one document.

## Architecture

- `types.go` — API response and document-reference types
- `client.go` — HTTP RPC client (POST /api/{method})
- `format.go` — Output rendering (mirrors the JS reference)
- `tools.go` — MCP tool definitions & handlers
- `main.go` — Server entry point

## Implementation Notes

- Zero dependencies beyond MCP SDK
- Manual parameter validation (limit, offset, contextLimit)
- API key redaction in errors
- Timeout: 30s per request
- Retry on 429 (rate limit): max 10s wait

## Benchmark

The three servers (JS, Go, Python) expose the same tools and return byte-identical output; this was
checked on every operation below. Numbers are per MCP tool call over stdio against a local mock of the
Outline API with realistic payloads, 200 timed calls per operation after warm-up, on one 4-CPU machine.
Method, harness and raw results: [benchmark/](https://github.com/minhquan2904/outline-mcp-js/tree/main/benchmark).

| Operation (p50 ms) | JS | Go | Python |
|---|---:|---:|---:|
| get_document (small page) | 0.62 | 0.14 | 1.45 |
| get_document (200 KB page) | 3.68 | 7.58 | 3.97 |
| search_documents (10 hits) | 0.59 | 0.34 | 1.89 |
| search_document_titles (15) | 0.59 | 0.21 | 1.60 |
| list_documents (25) | 0.58 | 0.27 | 1.72 |
| list_revisions (10) | 0.44 | 0.17 | 1.58 |
| list_comments (25) | 0.53 | 0.18 | 1.57 |
| update_document (append) | 0.47 | 0.15 | 1.70 |

| Resource | JS | Go | Python |
|---|---:|---:|---:|
| Startup to `initialize` (ms, median) | 91 | 2 | 513 |
| RSS idle (MB) | 71 | 8 | 64 |
| RSS peak after the run (MB) | 246 | 40 | 102 |
| 200 KB page, 16 in flight (req/s) | 113 | 394 | 343 |
| On disk | 26 MB `node_modules` + Node | 7.8 MB static binary | 76 MB virtualenv |

| Real instance, one page (p50 ms, n=10) | JS | Go | Python |
|---|---:|---:|---:|
| get_document | 36 | 23 | 29 |
| list_revisions | 29 | 23 | 30 |
| list_comments | 26 | 22 | 27 |
| update_document (append) | 45 | 40 | 46 |

What it means:

- On small calls Go is 2-4x faster than JS and 5-10x faster than Python, but that is 0.1-0.3 ms vs
  0.4-0.6 ms vs 1.4-1.9 ms. Against a real Outline instance (20-45 ms per call, last table) the server's own cost is lost
  in the network time, and the three are indistinguishable.
- Go starts in 2 ms and idles at 8 MB, against about 90 ms / 70 MB for Node and 510 ms / 64 MB for Python.
  That matters if a client launches the server per session.
- On the 200 KB page JS and Python (about 4 ms) beat Go (7.6 ms): Go decodes into generic maps and counts
  characters as runes. Under 16 concurrent requests Go and Python hold 340-390 req/s, while Node, which is
  single-threaded, drops to about 113.
- Peak memory under large pages: Go 40 MB, Python 102 MB, JS 246 MB.

Limits: single run, one machine, mock server in Python (see the caveats in `benchmark/README.md`).
The real-instance table uses only 10 calls per cell, so read it as "same ballpark", not as a ranking.

## TODO

- [ ] Integration tests with mock Outline API
- [ ] Typed (non-generic) response structs where the shape is stable
