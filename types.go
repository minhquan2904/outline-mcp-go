package main

// Outline API response types (flat RPC POST /api/{method}).
// Payloads are kept generic: Outline's shapes differ per endpoint and the
// formatters in format.go pick out only the fields they render.

type APIResponse struct {
	Data       interface{}            `json:"data,omitempty"`
	Pagination map[string]interface{} `json:"pagination,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

type Result struct {
	Ok    bool
	Data  interface{}
	Raw   APIResponse
	Error string
}

// Document reference (URL, UUID, or bare urlId)
type DocRef struct {
	Kind string // "uuid", "urlId", "invalid"
	ID   string
	Err  string
}
