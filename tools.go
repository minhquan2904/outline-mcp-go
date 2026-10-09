package main

import (
	"context"
	"encoding/json"
	"fmt"
)

type ToolDef struct {
	Name        string
	Description string
	Handler     func(ctx context.Context, args map[string]interface{}, client *Client) string
}

var Tools = []ToolDef{
	{
		Name: "search_documents",
		Description: `Full-text search across your Outline wiki. Returns each hit as
title, absolute URL and a matching snippet — NOT the full document body.
Use this when you do not yet know which document you need. Once you have
a URL or id, call get_document to read the body.`,
		Handler: handleSearchDocuments,
	},
	{
		Name: "search_document_titles",
		Description: `Search Outline document TITLES only. Cheaper and more precise than
search_documents when you already know roughly what the page is called
and only need to locate it. Returns titles and absolute URLs, no snippets.`,
		Handler: handleSearchTitles,
	},
	{
		Name: "get_document",
		Description: `Read one Outline document as markdown. Accepts a full document URL
(https://host/doc/some-title-a1B2c3D4e5), a bare urlId, or a UUID. Long
documents are truncated at maxChars with a marker stating the true total
length — raise maxChars and call again if you need the rest.`,
		Handler: handleGetDocument,
	},
	{
		Name: "list_collections",
		Description: `List the collections (top-level spaces) in the Outline wiki. Use this
first when you need to scope a search or browse, since collectionId is
what search_documents and list_documents filter on.`,
		Handler: handleListCollections,
	},
	{
		Name: "list_documents",
		Description: `Browse Outline documents without searching. Pass collectionId to list a
collection, or parentDocumentId to walk into one document's children.
Returns titles and absolute URLs only — call get_document to read one.`,
		Handler: handleListDocuments,
	},
	{
		Name: "list_revisions",
		Description: `List the saved revisions of one Outline document, newest first. Use this
to see how a spec changed over time; pair it with get_revision to read a
specific earlier version.`,
		Handler: handleListRevisions,
	},
	{
		Name: "get_revision",
		Description: `Read one earlier version of an Outline document as markdown. Get the
revision id from list_revisions first. Use this to diff what a spec said
before against what it says now.`,
		Handler: handleGetRevision,
	},
}

func handleSearchDocuments(ctx context.Context, args map[string]interface{}, client *Client) string {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return "Error: query parameter is required and must be a string"
	}

	limit := getInt(args, "limit", 10)
	if limit < 1 || limit > 50 {
		return "Error: limit must be between 1 and 50"
	}

	offset := getInt(args, "offset", 0)
	if offset < 0 {
		return "Error: offset must be >= 0"
	}

	contextLimit := getInt(args, "contextLimit", 200)
	if contextLimit < 0 || contextLimit > 1000 {
		return "Error: contextLimit must be between 0 and 1000"
	}

	params := map[string]interface{}{
		"query":          query,
		"limit":          limit,
		"offset":         offset,
		"contextLimit":   contextLimit,
	}
	if collID, ok := args["collectionId"].(string); ok && collID != "" {
		params["collectionId"] = collID
	}

	result, _ := client.Call(ctx, "documents.search", params)
	if !result.Ok {
		return fmt.Sprintf("Error: %s", result.Error)
	}

	baseURL := client.baseUrl
	return FormatSearchResults(result.Data, baseURL)
}

func handleSearchTitles(ctx context.Context, args map[string]interface{}, client *Client) string {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return "Error: query parameter is required and must be a string"
	}

	limit := getInt(args, "limit", 15)
	if limit < 1 || limit > 50 {
		return "Error: limit must be between 1 and 50"
	}

	params := map[string]interface{}{
		"query": query,
		"limit": limit,
	}

	result, _ := client.Call(ctx, "documents.search_titles", params)
	if !result.Ok {
		return fmt.Sprintf("Error: %s", result.Error)
	}

	return FormatTitleResults(result.Data, client.baseUrl)
}

func handleGetDocument(ctx context.Context, args map[string]interface{}, client *Client) string {
	idOrUrl, ok := args["idOrUrl"].(string)
	if !ok || idOrUrl == "" {
		return "Error: idOrUrl parameter is required"
	}

	ref := ParseDocRef(idOrUrl)
	if ref.Kind == "invalid" {
		return fmt.Sprintf("Error: %s", ref.Err)
	}

	maxChars := getInt(args, "maxChars", 40000)
	if maxChars < 0 || maxChars > 400000 {
		return "Error: maxChars must be between 0 and 400000"
	}

	result, _ := client.Call(ctx, "documents.info", map[string]interface{}{
		"id": ref.ID,
	})
	if !result.Ok {
		return fmt.Sprintf("Error: %s", result.Error)
	}

	return FormatDocument(result.Data, client.baseUrl, maxChars)
}

func handleListCollections(ctx context.Context, args map[string]interface{}, client *Client) string {
	limit := getInt(args, "limit", 25)
	if limit < 1 || limit > 100 {
		return "Error: limit must be between 1 and 100"
	}

	offset := getInt(args, "offset", 0)
	if offset < 0 {
		return "Error: offset must be >= 0"
	}

	result, _ := client.Call(ctx, "collections.list", map[string]interface{}{
		"limit":  limit,
		"offset": offset,
	})
	if !result.Ok {
		return fmt.Sprintf("Error: %s", result.Error)
	}

	bytes, _ := json.Marshal(result.Data)
	var collData CollectionsData
	_ = json.Unmarshal(bytes, &collData)

	return FormatList(toInterfaces(collData.Results), func(item interface{}) string {
		c := item.(Collection)
		return fmt.Sprintf("%s  [%s]", c.Name, c.ID)
	})
}

func handleListDocuments(ctx context.Context, args map[string]interface{}, client *Client) string {
	limit := getInt(args, "limit", 25)
	if limit < 1 || limit > 100 {
		return "Error: limit must be between 1 and 100"
	}

	offset := getInt(args, "offset", 0)
	if offset < 0 {
		return "Error: offset must be >= 0"
	}

	params := map[string]interface{}{
		"limit":  limit,
		"offset": offset,
	}
	if collID, ok := args["collectionId"].(string); ok && collID != "" {
		params["collectionId"] = collID
	}
	if parentID, ok := args["parentDocumentId"].(string); ok && parentID != "" {
		params["parentDocumentId"] = parentID
	}

	result, _ := client.Call(ctx, "documents.list", params)
	if !result.Ok {
		return fmt.Sprintf("Error: %s", result.Error)
	}

	bytes, _ := json.Marshal(result.Data)
	var docsData struct {
		Results []Document `json:"results"`
	}
	_ = json.Unmarshal(bytes, &docsData)

	return FormatList(toInterfaces(docsData.Results), func(item interface{}) string {
		d := item.(Document)
		return fmt.Sprintf("%s\n   %s", d.Title, AbsURL(client.baseUrl, d.URL))
	})
}

func handleListRevisions(ctx context.Context, args map[string]interface{}, client *Client) string {
	docID, ok := args["documentId"].(string)
	if !ok || docID == "" {
		return "Error: documentId parameter is required"
	}

	limit := getInt(args, "limit", 10)
	if limit < 1 || limit > 50 {
		return "Error: limit must be between 1 and 50"
	}

	offset := getInt(args, "offset", 0)
	if offset < 0 {
		return "Error: offset must be >= 0"
	}

	result, _ := client.Call(ctx, "revisions.list", map[string]interface{}{
		"documentId": docID,
		"limit":      limit,
		"offset":     offset,
	})
	if !result.Ok {
		return fmt.Sprintf("Error: %s", result.Error)
	}

	bytes, _ := json.Marshal(result.Data)
	var revData RevisionsData
	_ = json.Unmarshal(bytes, &revData)

	return FormatList(toInterfaces(revData.Results), func(item interface{}) string {
		r := item.(Revision)
		return fmt.Sprintf("%s  %s  [%s]", r.CreatedAt, r.Title, r.ID)
	})
}

func handleGetRevision(ctx context.Context, args map[string]interface{}, client *Client) string {
	revID, ok := args["revisionId"].(string)
	if !ok || revID == "" {
		return "Error: revisionId parameter is required"
	}

	maxChars := getInt(args, "maxChars", 40000)
	if maxChars < 0 || maxChars > 400000 {
		return "Error: maxChars must be between 0 and 400000"
	}

	result, _ := client.Call(ctx, "revisions.info", map[string]interface{}{
		"id": revID,
	})
	if !result.Ok {
		return fmt.Sprintf("Error: %s", result.Error)
	}

	return FormatRevision(result.Data, maxChars)
}

// Helpers
func getInt(args map[string]interface{}, key string, defaultVal int) int {
	if v, ok := args[key]; ok {
		switch vv := v.(type) {
		case float64:
			return int(vv)
		case int:
			return vv
		case string:
			var i int
			fmt.Sscanf(vv, "%d", &i)
			return i
		}
	}
	return defaultVal
}

func toInterfaces[T any](slice []T) []interface{} {
	result := make([]interface{}, len(slice))
	for i, v := range slice {
		result[i] = v
	}
	return result
}
