package main

type schema = map[string]any

func str(desc string) schema { return schema{"type": "string", "description": desc} }

func intRange(desc string, min, max int) schema {
	return schema{"type": "integer", "description": desc, "minimum": min, "maximum": max}
}

func objectSchema(required []string, props schema) schema {
	s := schema{"type": "object", "properties": props}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}

var (
	limitProp  = intRange("Max results", 1, 50)
	limit100   = intRange("Max results", 1, 100)
	offsetProp = schema{"type": "integer", "description": "Results to skip", "minimum": 0}
	maxChars   = schema{"type": "integer", "description": "Truncate body at this many characters", "minimum": 1}
)

// toolSchemas holds the JSON Schema of each tool's arguments, keyed by tool name.
var toolSchemas = map[string]schema{
	"search_documents": objectSchema([]string{"query"}, schema{
		"query":        str("Search text"),
		"collectionId": str("Restrict to one collection"),
		"limit":        limitProp,
		"offset":       offsetProp,
		"contextLimit": intRange("Snippet length in characters", 0, 1000),
	}),
	"search_document_titles": objectSchema([]string{"query"}, schema{
		"query":        str("Title text to look for"),
		"collectionId": str("Restrict to one collection"),
		"limit":        limitProp,
	}),
	"get_document": objectSchema([]string{"idOrUrl"}, schema{
		"idOrUrl":  str("Document URL, urlId or UUID"),
		"maxChars": maxChars,
	}),
	"list_collections": objectSchema(nil, schema{
		"limit":  limit100,
		"offset": offsetProp,
	}),
	"list_documents": objectSchema(nil, schema{
		"collectionId":     str("List documents of this collection"),
		"parentDocumentId": str("List children of this document"),
		"limit":            limit100,
		"offset":           offsetProp,
	}),
	"list_revisions": objectSchema([]string{"documentId"}, schema{
		"documentId": str("Document id"),
		"limit":      limitProp,
		"offset":     offsetProp,
	}),
	"get_revision": objectSchema([]string{"revisionId"}, schema{
		"revisionId": str("Revision id from list_revisions"),
		"maxChars":   maxChars,
	}),
	"list_comments": objectSchema([]string{"documentId"}, schema{
		"documentId": str("UUID of the document"),
		"limit":      intRange("Max results", 1, 100),
		"offset":     offsetProp,
	}),
	"create_document": objectSchema([]string{"title", "collectionId"}, schema{
		"title":            str("Document title"),
		"text":             str("Markdown body"),
		"collectionId":     str("Collection to create the document in"),
		"parentDocumentId": str("Nest under this document"),
		"publish":          schema{"type": "boolean", "description": "Publish immediately (default) or leave as draft"},
	}),
	"update_document": objectSchema([]string{"documentId"}, schema{
		"documentId": str("Document URL, urlId, or UUID"),
		"text":       str("Markdown to write"),
		"title":      str("New title"),
		"append":     schema{"type": "boolean", "description": "Append (default) or replace the body"},
	}),
	"move_document": objectSchema([]string{"documentId"}, schema{
		"documentId":       str("Document URL, urlId, or UUID"),
		"collectionId":     str("Destination collection"),
		"parentDocumentId": str("Destination parent document"),
	}),
	"archive_document": objectSchema([]string{"documentId"}, schema{"documentId": str("Document URL, urlId, or UUID")}),
	"delete_document":  objectSchema([]string{"documentId"}, schema{"documentId": str("Document URL, urlId, or UUID")}),
}
