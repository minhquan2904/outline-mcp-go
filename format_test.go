package main

import (
	"context"
	"strings"
	"testing"
)

func TestParseDocRef(t *testing.T) {
	uuid := "550e8400-e29b-41d4-a716-446655440000"
	tests := []struct{ in, kind, id string }{
		{uuid, "uuid", uuid},
		{"a1B2c3D4e5", "urlId", "a1B2c3D4e5"},           // 10 chars: lower boundary
		{"a1B2c3D4e5f6g7h", "urlId", "a1B2c3D4e5f6g7h"}, // 15 chars: upper boundary
		{"https://outline.example.com/doc/api-a1B2c3D4e5", "urlId", "a1B2c3D4e5"},
		{"/doc/api-a1B2c3D4e5?x=1#top", "urlId", "a1B2c3D4e5"},
		{"https://outline.example.com/doc/benchmark-mcp-x7Yq2Lm9Kp/edit", "urlId", "x7Yq2Lm9Kp"},
		{"title-a1B2c3D4e5", "urlId", "a1B2c3D4e5"},
		{"a1B2c3D4e", "invalid", ""},        // 9 chars
		{"a1B2c3D4e5f6g7h8", "invalid", ""}, // 16 chars
		{"title-short", "invalid", ""},
		{"https://example.com/other/page", "invalid", ""},
		{"", "invalid", ""},
		{"   ", "invalid", ""},
		{"invalid!", "invalid", ""},
	}
	for _, tt := range tests {
		ref := ParseDocRef(tt.in)
		if ref.Kind != tt.kind || ref.ID != tt.id {
			t.Errorf("ParseDocRef(%q) = {%s %q}, want {%s %q}", tt.in, ref.Kind, ref.ID, tt.kind, tt.id)
		}
		if ref.Kind == "invalid" && ref.Err == "" {
			t.Errorf("ParseDocRef(%q): invalid result has no reason", tt.in)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got, total := truncate("hello", 5); got != "hello" || total != 5 {
		t.Errorf("exact limit must not cut, got %q %d", got, total)
	}
	got, total := truncate("hello!", 5)
	if total != 6 || !strings.HasPrefix(got, "hello\n\n[... truncated — showing 5 of 6 characters.") {
		t.Errorf("one over the limit: %q %d", got, total)
	}
	if got, _ := truncate("", 0); got != "" {
		t.Errorf("empty text: %q", got)
	}
	if got, _ := truncate("tiếng Việt", 5); !strings.HasPrefix(got, "tiếng") {
		t.Errorf("must cut by characters, not bytes: %q", got)
	}
}

func TestAbsURL(t *testing.T) {
	cases := []struct{ base, path, want string }{
		{"https://o.example.com", "/doc/x-abc", "https://o.example.com/doc/x-abc"},
		{"https://o.example.com/", "/doc/x-abc", "https://o.example.com/doc/x-abc"},
		{"https://o.example.com", "https://other/doc", "https://other/doc"},
		{"", "/doc/x", "/doc/x"},
		{"https://o.example.com", "", ""},
	}
	for _, c := range cases {
		if got := AbsURL(c.base, c.path); got != c.want {
			t.Errorf("AbsURL(%q,%q) = %q, want %q", c.base, c.path, got, c.want)
		}
	}
}

func TestFormatSearchResults(t *testing.T) {
	hits := []interface{}{
		obj{"context": " snippet one ", "document": obj{"title": "First", "url": "/doc/first-abc"}},
		obj{"document": obj{"url": "/doc/second-abc"}},
	}
	want := "1. First\n   https://o.example.com/doc/first-abc\n   snippet one\n\n" +
		"2. (untitled)\n   https://o.example.com/doc/second-abc"
	if got := FormatSearchResults(hits, "https://o.example.com"); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	if got := FormatSearchResults([]interface{}{}, ""); got != "No results." {
		t.Errorf("empty: %q", got)
	}
}

func TestFormatTitleResults(t *testing.T) {
	docs := []interface{}{obj{"title": "A", "url": "/doc/a"}, obj{"title": "B", "url": "/doc/b"}}
	want := "1. A\n   https://h/doc/a\n2. B\n   https://h/doc/b"
	if got := FormatTitleResults(docs, "https://h"); got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestFormatDocument(t *testing.T) {
	doc := obj{
		"title": "Spec", "id": "uuid-1", "urlId": "abc1234567", "collectionId": "col-1",
		"url": "/doc/spec-abc1234567", "updatedAt": "2026-01-02T03:04:05Z",
		"updatedBy": obj{"name": "Quan"}, "text": "body text",
	}
	want := "# Spec\nid: uuid-1 · urlId: abc1234567 · collection: col-1\n" +
		"url: https://h/doc/spec-abc1234567\nupdated: 2026-01-02T03:04:05Z by Quan\n---\nbody text"
	if got := FormatDocument(doc, "https://h", 100); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	if got := FormatDocument(obj{"text": "x"}, "", 10); !strings.Contains(got, "by unknown") {
		t.Errorf("missing updatedBy must not break: %s", got)
	}
}

func TestFormatList(t *testing.T) {
	raw := APIResponse{
		Data:       []interface{}{obj{"name": "Eng", "id": "c1"}, obj{"name": "Ops", "id": "c2"}},
		Pagination: map[string]interface{}{"limit": float64(25), "offset": float64(0), "total": float64(2)},
	}
	want := "1. Eng  [c1]\n2. Ops  [c2]\n\n[limit=25 · offset=0 · total=2]"
	got := FormatList(raw, func(c obj) string { return asStr(c["name"]) + "  [" + asStr(c["id"]) + "]" })
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
	if got := FormatList(APIResponse{}, nil); got != "No results." {
		t.Errorf("empty: %q", got)
	}
}

func TestWriteBlocked(t *testing.T) {
	const doc = "3f9a2c10-1b2c-4d5e-8f90-abcdef123456"
	t.Setenv("OUTLINE_WRITE_DOC_ID", "")
	for _, tool := range []string{"create_document", "update_document", "move_document", "archive_document", "delete_document"} {
		if msg := writeBlocked(context.Background(), nil, tool, doc); msg != "" {
			t.Errorf("%s must be allowed without a restriction, got %q", tool, msg)
		}
	}

	t.Setenv("OUTLINE_WRITE_DOC_ID", doc)
	if msg := writeBlocked(context.Background(), nil, "update_document", doc); msg != "" {
		t.Errorf("update of the allowed document must pass, got %q", msg)
	}
	if msg := writeBlocked(context.Background(), nil, "update_document", "11111111-2222-3333-4444-555555555555"); !strings.Contains(msg, "may only update") {
		t.Errorf("other document must be refused, got %q", msg)
	}
	for _, tool := range []string{"create_document", "move_document", "archive_document", "delete_document"} {
		if msg := writeBlocked(context.Background(), nil, tool, doc); !strings.Contains(msg, "writes are restricted") {
			t.Errorf("%s must be refused under a restriction, got %q", tool, msg)
		}
	}
}

func TestWriteBlockedResolvesUUIDAndUrlID(t *testing.T) {
	const doc = "3f9a2c10-1b2c-4d5e-8f90-abcdef123456"
	t.Setenv("OUTLINE_WRITE_DOC_ID", doc)

	lookup := func(body string) *Client {
		c := NewClient("https://h", "k")
		c.httpClient = &MockHTTPClient{DoFn: func(req interface{}) ([]byte, int, error) {
			return []byte(body), 200, nil
		}}
		return c
	}
	same := lookup(`{"data": {"id": "` + doc + `", "urlId": "x7Yq2Lm9Kp"}}`)
	if msg := writeBlocked(context.Background(), same, "update_document", "x7Yq2Lm9Kp"); msg != "" {
		t.Errorf("urlId of the allowed document must pass, got %q", msg)
	}
	other := lookup(`{"data": {"id": "11111111-2222-3333-4444-555555555555", "urlId": "zzzzzzzzzz"}}`)
	if msg := writeBlocked(context.Background(), other, "update_document", "x7Yq2Lm9Kp"); !strings.Contains(msg, "may only update") {
		t.Errorf("urlId of another document must be refused, got %q", msg)
	}
}
