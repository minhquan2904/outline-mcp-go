package main

// Outline API response types (flat RPC POST /api/{method})

type APIResponse struct {
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

type Result struct {
	Ok    bool
	Data  interface{}
	Raw   interface{}
	Error string
}

// Search results
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

type SearchData struct {
	Results    []SearchResult `json:"results"`
	Pagination struct {
		Offset int `json:"offset"`
		Limit  int `json:"limit"`
	} `json:"pagination"`
}

// Document
type Document struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
	Text  string `json:"text"`
}

// Collection
type Collection struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CollectionsData struct {
	Results    []Collection `json:"results"`
	Pagination struct {
		Offset int `json:"offset"`
		Limit  int `json:"limit"`
	} `json:"pagination"`
}

// Revision
type Revision struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"createdAt"`
}

type RevisionsData struct {
	Results    []Revision `json:"results"`
	Pagination struct {
		Offset int `json:"offset"`
		Limit  int `json:"limit"`
	} `json:"pagination"`
}

// Comment
type Comment struct {
	ID        string `json:"id"`
	CreatedAt string `json:"createdAt"`
	CreatedBy struct {
		Name string `json:"name"`
	} `json:"createdBy"`
}

type CommentsData struct {
	Results    []Comment `json:"results"`
	Pagination struct {
		Offset int `json:"offset"`
		Limit  int `json:"limit"`
	} `json:"pagination"`
}

// Document reference (URL, UUID, or bare ID)
type DocRef struct {
	Kind string // "url", "uuid", "id", "invalid"
	ID   string
	Err  string
}
