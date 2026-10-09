package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
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

	result, _ := c.Call(context.Background(), "documents.search", map[string]interface{}{})

	if result.Ok {
		t.Errorf("expected timeout error")
	}
	if !strings.Contains(result.Error, "timed out") {
		t.Errorf("expected timed out in error, got: %s", result.Error)
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

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	body, status, err := m.DoFn(req)
	if err != nil {
		return nil, err
	}
	// like http.Client: fail once the request context is done
	if err := req.Context().Err(); err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{},
		Body:       io.NopCloser(bytes.NewReader(body)),
	}, nil
}
