.PHONY: test build run clean

test:
	go test -v ./...

build:
	go build -o outline-mcp-go

run: build
	OUTLINE_API_URL=http://localhost:3000 OUTLINE_API_KEY=test_key ./outline-mcp-go

clean:
	rm -f outline-mcp-go
	go clean

fmt:
	go fmt ./...

vet:
	go vet ./...
