package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

func FormatSearchResults(data interface{}, baseURL string) string {
	bytes, _ := json.Marshal(data)
	var searchData SearchData
	if err := json.Unmarshal(bytes, &searchData); err != nil {
		return fmt.Sprintf("Error parsing search results: %v", err)
	}

	var buf strings.Builder
	for _, result := range searchData.Results {
		absURL := AbsURL(baseURL, result.URL)
		fmt.Fprintf(&buf, "%s\n%s\n%s\n\n", result.Title, absURL, result.Snippet)
	}
	return buf.String()
}

func FormatDocument(data interface{}, baseURL string, maxChars int) string {
	bytes, _ := json.Marshal(data)
	var doc Document
	if err := json.Unmarshal(bytes, &doc); err != nil {
		return fmt.Sprintf("Error parsing document: %v", err)
	}

	text := doc.Text
	truncated := false
	if maxChars > 0 && len(text) > maxChars {
		text = text[:maxChars]
		truncated = true
	}

	var buf strings.Builder
	fmt.Fprintf(&buf, "# %s\n\n", doc.Title)
	buf.WriteString(text)

	if truncated {
		fmt.Fprintf(&buf, "\n\n_(truncated; true length: %d characters)_", len(doc.Text))
	}

	return buf.String()
}

func FormatList(items []interface{}, itemFmt func(item interface{}) string) string {
	var buf strings.Builder
	for i, item := range items {
		if i > 0 {
			buf.WriteString("\n")
		}
		buf.WriteString(itemFmt(item))
	}
	return buf.String()
}

func AbsURL(baseURL, relPath string) string {
	if strings.HasPrefix(relPath, "http://") || strings.HasPrefix(relPath, "https://") {
		return relPath
	}

	u, _ := url.Parse(baseURL)
	u.Path = relPath
	return u.String()
}

func FormatTitleResults(data interface{}, baseURL string) string {
	// Search titles returns same structure as search
	return FormatSearchResults(data, baseURL)
}

func FormatRevision(data interface{}, maxChars int) string {
	bytes, _ := json.Marshal(data)
	var doc Document
	if err := json.Unmarshal(bytes, &doc); err != nil {
		return fmt.Sprintf("Error parsing revision: %v", err)
	}

	text := doc.Text
	if maxChars > 0 && len(text) > maxChars {
		text = text[:maxChars]
	}

	var buf strings.Builder
	fmt.Fprintf(&buf, "# %s (Revision)\n\n", doc.Title)
	buf.WriteString(text)

	if maxChars > 0 && len(doc.Text) > maxChars {
		fmt.Fprintf(&buf, "\n\n_(truncated; true length: %d characters)_", len(doc.Text))
	}

	return buf.String()
}
