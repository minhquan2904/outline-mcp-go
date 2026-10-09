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
		"limit":  limitProp,
		"offset": offsetProp,
	}),
	"list_documents": objectSchema(nil, schema{
		"collectionId":     str("List documents of this collection"),
		"parentDocumentId": str("List children of this document"),
		"limit":            limitProp,
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
}
