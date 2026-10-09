package main

import (
	"context"
	"fmt"
	"os"
	"strings"
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
	{
		Name: "list_comments",
		Description: `List review comments left on one Outline document. Use this when a spec
has open questions or reviewer pushback that is not reflected in the body
text itself.`,
		Handler: handleListComments,
	},
	{
		Name: "create_document",
		Description: `Create a new Outline document. Pass collectionId (and optionally
parentDocumentId to nest it). Published immediately unless publish=false.`,
		Handler: handleCreateDocument,
	},
	{
		Name: "update_document",
		Description: `Edit an Outline document's body and/or title. Appends text by default;
pass append=false to replace the body. Accepts a URL, urlId or UUID.`,
		Handler: handleUpdateDocument,
	},
	{
		Name:        "move_document",
		Description: `Move an Outline document to another collection and/or under another parent document.`,
		Handler:     handleMoveDocument,
	},
	{
		Name:        "archive_document",
		Description: `Archive an Outline document (recoverable from the archive).`,
		Handler:     handleArchiveDocument,
	},
	{
		Name: "delete_document",
		Description: `Delete an Outline document. It goes to the trash and can be restored;
this tool never deletes permanently.`,
		Handler: handleDeleteDocument,
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
		"query":        query,
		"limit":        limit,
		"offset":       offset,
		"contextLimit": contextLimit,
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
		return ref.Err
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

	return FormatList(result.Raw, func(c obj) string {
		return fmt.Sprintf("%s  [%s]", asStr(c["name"]), asStr(c["id"]))
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

	return FormatList(result.Raw, func(d obj) string {
		return fmt.Sprintf("%s\n   %s", asStr(d["title"]), AbsURL(client.baseUrl, asStr(d["url"])))
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

	return FormatList(result.Raw, func(r obj) string {
		return fmt.Sprintf("%s  %s  [%s]", asStr(r["createdAt"]), asStr(r["title"]), asStr(r["id"]))
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

func handleListComments(ctx context.Context, args map[string]interface{}, client *Client) string {
	docID, ok := args["documentId"].(string)
	if !ok || docID == "" {
		return "Error: documentId parameter is required"
	}

	limit := getInt(args, "limit", 25)
	if limit < 1 || limit > 100 {
		return "Error: limit must be between 1 and 100"
	}

	offset := getInt(args, "offset", 0)
	if offset < 0 {
		return "Error: offset must be >= 0"
	}

	result, _ := client.Call(ctx, "comments.list", map[string]interface{}{
		"documentId": docID,
		"limit":      limit,
		"offset":     offset,
	})
	if !result.Ok {
		return fmt.Sprintf("Error: %s", result.Error)
	}

	return FormatList(result.Raw, func(c obj) string {
		return fmt.Sprintf("%s  %s  [%s]", asStr(c["createdAt"]), who(c["createdBy"], ""), asStr(c["id"]))
	})
}

// writeBlocked implements the optional OUTLINE_WRITE_DOC_ID restriction: when
// set, writes are confined to update_document on that single document. The
// allowed id and the target may be written differently (UUID vs urlId); when
// they are, Outline is asked to resolve the target so both names compare equal.
func writeBlocked(ctx context.Context, client *Client, tool, documentID string) string {
	allowed := strings.TrimSpace(os.Getenv("OUTLINE_WRITE_DOC_ID"))
	if allowed == "" {
		return ""
	}
	if tool != "update_document" {
		return fmt.Sprintf("Error: writes are restricted to one document (OUTLINE_WRITE_DOC_ID); %s is not allowed.", tool)
	}
	want, got := ParseDocRef(allowed), ParseDocRef(documentID)
	valid := want.Kind != "invalid" && got.Kind != "invalid"
	ok := valid && want.ID == got.ID
	if valid && !ok && want.Kind != got.Kind {
		if info, _ := client.Call(ctx, "documents.info", map[string]interface{}{"id": got.ID}); info.Ok {
			d := asObj(info.Data)
			ok = want.ID == asStr(d["id"]) || want.ID == asStr(d["urlId"])
		}
	}
	if !ok {
		return "Error: this server may only update the document named by OUTLINE_WRITE_DOC_ID"
	}
	return ""
}

// withDocument resolves the document argument, applies the write guard and runs the call.
func withDocument(ctx context.Context, client *Client, args map[string]interface{}, tool string, run func(id string) string) string {
	docID, ok := args["documentId"].(string)
	if !ok || docID == "" {
		return "Error: documentId parameter is required"
	}
	ref := ParseDocRef(docID)
	if ref.Kind == "invalid" {
		return ref.Err
	}
	if msg := writeBlocked(ctx, client, tool, docID); msg != "" {
		return msg
	}
	return run(ref.ID)
}

func describeDoc(data interface{}, baseURL string) string {
	d := asObj(data)
	return fmt.Sprintf("\"%s\" (%d characters)\n%s",
		asStr(d["title"]), len([]rune(asStr(d["text"]))), AbsURL(baseURL, asStr(d["url"])))
}

func handleCreateDocument(ctx context.Context, args map[string]interface{}, client *Client) string {
	if msg := writeBlocked(ctx, client, "create_document", ""); msg != "" {
		return msg
	}
	title, ok := args["title"].(string)
	collID, ok2 := args["collectionId"].(string)
	if !ok || !ok2 || title == "" || collID == "" {
		return "Error: title and collectionId parameters are required"
	}
	text, _ := args["text"].(string)
	publish := true
	if v, ok := args["publish"].(bool); ok {
		publish = v
	}
	params := map[string]interface{}{
		"title": title, "text": text, "collectionId": collID, "publish": publish,
	}
	if parent, ok := args["parentDocumentId"].(string); ok && parent != "" {
		params["parentDocumentId"] = parent
	}
	result, _ := client.Call(ctx, "documents.create", params)
	if !result.Ok {
		return fmt.Sprintf("Error: %s", result.Error)
	}
	return "Created " + describeDoc(result.Data, client.baseUrl)
}

func handleUpdateDocument(ctx context.Context, args map[string]interface{}, client *Client) string {
	return withDocument(ctx, client, args, "update_document", func(id string) string {
		appendMode := true
		if v, ok := args["append"].(bool); ok {
			appendMode = v
		}
		params := map[string]interface{}{"id": id, "append": appendMode}
		if text, ok := args["text"].(string); ok {
			params["text"] = text
		}
		if title, ok := args["title"].(string); ok {
			params["title"] = title
		}
		result, _ := client.Call(ctx, "documents.update", params)
		if !result.Ok {
			return fmt.Sprintf("Error: %s", result.Error)
		}
		return "Updated " + describeDoc(result.Data, client.baseUrl)
	})
}

func handleMoveDocument(ctx context.Context, args map[string]interface{}, client *Client) string {
	return withDocument(ctx, client, args, "move_document", func(id string) string {
		params := map[string]interface{}{"id": id}
		if v, ok := args["collectionId"].(string); ok && v != "" {
			params["collectionId"] = v
		}
		if v, ok := args["parentDocumentId"].(string); ok && v != "" {
			params["parentDocumentId"] = v
		}
		result, _ := client.Call(ctx, "documents.move", params)
		if !result.Ok {
			return fmt.Sprintf("Error: %s", result.Error)
		}
		return "Moved document " + id
	})
}

func handleArchiveDocument(ctx context.Context, args map[string]interface{}, client *Client) string {
	return withDocument(ctx, client, args, "archive_document", func(id string) string {
		result, _ := client.Call(ctx, "documents.archive", map[string]interface{}{"id": id})
		if !result.Ok {
			return fmt.Sprintf("Error: %s", result.Error)
		}
		return "Archived document " + id
	})
}

func handleDeleteDocument(ctx context.Context, args map[string]interface{}, client *Client) string {
	return withDocument(ctx, client, args, "delete_document", func(id string) string {
		result, _ := client.Call(ctx, "documents.delete", map[string]interface{}{"id": id})
		if !result.Ok {
			return fmt.Sprintf("Error: %s", result.Error)
		}
		return "Moved document " + id + " to the trash"
	})
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
