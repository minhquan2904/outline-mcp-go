package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Rendering helpers. Output mirrors the JavaScript reference implementation
// (outline-mcp/format.js) so all three servers answer identically.

const noResults = "No results."

type obj = map[string]interface{}

func asObj(v interface{}) obj {
	m, _ := v.(obj)
	return m
}

func asStr(v interface{}) string {
	switch s := v.(type) {
	case string:
		return s
	case nil:
		return ""
	case float64:
		return numStr(s)
	}
	return fmt.Sprint(v)
}

func numStr(f float64) string {
	if f == float64(int64(f)) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// AbsURL turns Outline's relative document paths into absolute links.
func AbsURL(baseURL, path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(path), "http://") || strings.HasPrefix(strings.ToLower(path), "https://") {
		return path
	}
	base := strings.TrimRight(baseURL, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}

// truncate cuts the body at maxChars characters and reports the original length.
func truncate(text string, maxChars int) (string, int) {
	runes := []rune(text)
	total := len(runes)
	if total <= maxChars {
		return text, total
	}
	marker := fmt.Sprintf("\n\n[... truncated — showing %d of %d characters. "+
		"Call again with a larger maxChars to read the rest.]", maxChars, total)
	return string(runes[:maxChars]) + marker, total
}

func docLine(doc obj, baseURL string, index int) string {
	return fmt.Sprintf("%d. %s\n   %s", index+1,
		orDefault(asStr(doc["title"]), "(untitled)"), AbsURL(baseURL, asStr(doc["url"])))
}

// FormatSearchResults renders documents.search hits ({context, document, ranking}).
func FormatSearchResults(data interface{}, baseURL string) string {
	hits, _ := data.([]interface{})
	if len(hits) == 0 {
		return noResults
	}
	parts := make([]string, len(hits))
	for i, h := range hits {
		hit := asObj(h)
		line := docLine(asObj(hit["document"]), baseURL, i)
		if ctx := strings.TrimSpace(asStr(hit["context"])); ctx != "" {
			line += "\n   " + ctx
		}
		parts[i] = line
	}
	return strings.Join(parts, "\n\n")
}

// FormatTitleResults renders documents.search_titles (bare document objects).
func FormatTitleResults(data interface{}, baseURL string) string {
	docs, _ := data.([]interface{})
	if len(docs) == 0 {
		return noResults
	}
	parts := make([]string, len(docs))
	for i, d := range docs {
		parts[i] = docLine(asObj(d), baseURL, i)
	}
	return strings.Join(parts, "\n")
}

func who(person interface{}, fallback string) string {
	if name := asStr(asObj(person)["name"]); name != "" {
		return name
	}
	return orDefault(fallback, "unknown")
}

func FormatDocument(data interface{}, baseURL string, maxChars int) string {
	d := asObj(data)
	body, _ := truncate(asStr(d["text"]), maxChars)
	header := strings.Join([]string{
		"# " + orDefault(asStr(d["title"]), "(untitled)"),
		fmt.Sprintf("id: %s · urlId: %s · collection: %s", asStr(d["id"]), asStr(d["urlId"]), asStr(d["collectionId"])),
		"url: " + AbsURL(baseURL, asStr(d["url"])),
		fmt.Sprintf("updated: %s by %s", orDefault(asStr(d["updatedAt"]), "unknown"), who(d["updatedBy"], "")),
	}, "\n")
	return header + "\n---\n" + body
}

func FormatRevision(data interface{}, maxChars int) string {
	r := asObj(data)
	body, _ := truncate(asStr(r["text"]), maxChars)
	header := strings.Join([]string{
		fmt.Sprintf("# %s (revision %s)", orDefault(asStr(r["title"]), "(untitled)"), asStr(r["id"])),
		"document: " + asStr(r["documentId"]),
		fmt.Sprintf("created: %s by %s", orDefault(asStr(r["createdAt"]), "unknown"), who(r["createdBy"], asStr(r["name"]))),
	}, "\n")
	return header + "\n---\n" + body
}

// FormatList renders a list payload and keeps the pagination cursor visible.
func FormatList(raw APIResponse, render func(item obj) string) string {
	items, _ := raw.Data.([]interface{})
	if len(items) == 0 {
		return noResults
	}
	lines := make([]string, 0, len(items)+2)
	for i, it := range items {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, render(asObj(it))))
	}
	if p := raw.Pagination; p != nil {
		parts := []string{"limit=" + asStr(p["limit"]), "offset=" + asStr(p["offset"])}
		if t, ok := p["total"]; ok {
			parts = append(parts, "total="+asStr(t))
		}
		lines = append(lines, "", "["+strings.Join(parts, " · ")+"]")
	}
	return strings.Join(lines, "\n")
}
