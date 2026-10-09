package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// MockClient for testing
type MockClient struct {
	CallFn func(ctx context.Context, method string, params map[string]interface{}) (*Result, error)
}

func (m *MockClient) Call(ctx context.Context, method string, params map[string]interface{}) (*Result, error) {
	if m.CallFn != nil {
		return m.CallFn(ctx, method, params)
	}
	return &Result{Ok: false, Error: "mock not configured"}, nil
}

func TestClientCall_Success(t *testing.T) {
	c := NewClient("https://outline.example.com", "test_key_123")

	mockHTTP := &MockHTTPClient{
		DoFn: func(req interface{}) ([]byte, int, error) {
			return []byte(`{"data": {"results": [{"title": "Test", "url": "/doc/test", "snippet": "test content"}]}}`), 200, nil
		},
	}
	c.httpClient = mockHTTP

	result, err := c.Call(context.Background(), "documents.search", map[string]interface{}{
		"query": "test",
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !result.Ok {
		t.Errorf("expected ok=true, got %v", result.Ok)
	}
	if result.Error != "" {
		t.Errorf("expected no error message, got: %s", result.Error)
	}
}

func TestClientCall_Unauthorized(t *testing.T) {
	c := NewClient("https://outline.example.com", "bad_key")

	mockHTTP := &MockHTTPClient{
		DoFn: func(req interface{}) ([]byte, int, error) {
			return []byte(`{"error": "Invalid token"}`), 401, nil
		},
	}
	c.httpClient = mockHTTP

	result, err := c.Call(context.Background(), "documents.search", map[string]interface{}{})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Ok {
		t.Errorf("expected ok=false for 401")
	}
	if result.Error == "" {
		t.Errorf("expected error message")
	}
	// API key must not leak
	if strings.Contains(result.Error, "bad_key") {
		t.Errorf("API key leaked in error: %s", result.Error)
	}
}

func TestClientCall_Timeout(t *testing.T) {
	c := NewClient("https://outline.example.com", "test_key")
	c.timeoutMs = 100

	mockHTTP := &MockHTTPClient{
		DoFn: func(req interface{}) ([]byte, int, error) {
			time.Sleep(200 * time.Millisecond)
			return nil, 0, nil
		},
	}
	c.httpClient = mockHTTP

	result, err := c.Call(context.Background(), "documents.search", map[string]interface{}{})

	if result.Ok {
		t.Errorf("expected timeout error")
	}
	if !strings.Contains(result.Error, "timeout") {
		t.Errorf("expected timeout in error, got: %s", result.Error)
	}
}

func TestClientCall_NoCredentials(t *testing.T) {
	c := NewClient("https://outline.example.com", "")

	result, _ := c.Call(context.Background(), "documents.search", map[string]interface{}{})

	if result.Ok {
		t.Errorf("expected error for missing credentials")
	}
	if !strings.Contains(result.Error, "OUTLINE_API_KEY") {
		t.Errorf("expected OUTLINE_API_KEY in error, got: %s", result.Error)
	}
}

func TestParseDocRef_URL(t *testing.T) {
	tests := []struct {
		input string
		kind  string
		id    string
	}{
		{"https://outline.example.com/doc/api-a1B2c3D4e5", "url", "a1B2c3D4e5"},
		{"http://localhost:3000/doc/title-xyz123", "url", "xyz123"},
		{"a1B2c3D4e5", "id", "a1B2c3D4e5"},
		{"550e8400-e29b-41d4-a716-446655440000", "uuid", "550e8400-e29b-41d4-a716-446655440000"},
		{"invalid!", "invalid", ""},
	}

	for _, tt := range tests {
		ref := ParseDocRef(tt.input)
		if ref.Kind != tt.kind {
			t.Errorf("ParseDocRef(%q) kind = %q, want %q", tt.input, ref.Kind, tt.kind)
		}
		if ref.ID != tt.id {
			t.Errorf("ParseDocRef(%q) id = %q, want %q", tt.input, ref.ID, tt.id)
		}
	}
}

func TestClientCall_ResponseUnmarshal(t *testing.T) {
	c := NewClient("https://outline.example.com", "test_key")

	mockHTTP := &MockHTTPClient{
		DoFn: func(req interface{}) ([]byte, int, error) {
			data := map[string]interface{}{
				"data": map[string]interface{}{
					"id":    "doc-123",
					"title": "Test Doc",
					"text":  "Content",
				},
			}
			body, _ := json.Marshal(data)
			return body, 200, nil
		},
	}
	c.httpClient = mockHTTP

	result, _ := c.Call(context.Background(), "documents.info", map[string]interface{}{
		"id": "doc-123",
	})

	if !result.Ok {
		t.Errorf("expected ok=true, got error: %s", result.Error)
	}

	// Data should be unmarshaled
	dataMap, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Errorf("expected Data to be map[string]interface{}")
	}
	if dataMap["title"] != "Test Doc" {
		t.Errorf("expected title='Test Doc', got %v", dataMap["title"])
	}
}

// Helper mock for HTTP
type MockHTTPClient struct {
	DoFn func(req interface{}) ([]byte, int, error)
}

