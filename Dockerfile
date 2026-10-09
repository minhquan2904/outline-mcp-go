# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/outline-mcp-go .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/outline-mcp-go /outline-mcp-go
ENTRYPOINT ["/outline-mcp-go"]
