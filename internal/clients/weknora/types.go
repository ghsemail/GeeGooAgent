package weknora

// KnowledgeBase is a WeKnora knowledge base summary.
type KnowledgeBase struct {
	ID               string
	Name             string
	Description      string
	EmbeddingModelID string
	ChatModelID      string
}

// Model is a WeKnora model record.
type Model struct {
	ID   string
	Name string
	Type string
}

// Document is one knowledge file.
type Document struct {
	ID          string
	FileName    string
	Title       string
	FolderPath  string
	FileSize    int64
	ParseStatus string
	UpdatedAt   string
}

// FolderNode is one node in the folder tree.
type FolderNode struct {
	Path          string       `json:"path"`
	Name          string       `json:"name"`
	DocumentCount int          `json:"document_count"`
	TotalCount    int          `json:"total_count"`
	Children      []FolderNode `json:"children"`
}

// FolderTree is the aggregated folder view.
type FolderTree struct {
	RootDocumentCount  int          `json:"root_document_count"`
	TotalDocumentCount int          `json:"total_document_count"`
	Folders            []FolderNode `json:"folders"`
}

// SearchHit is one hybrid-search chunk.
type SearchHit struct {
	Content  string  `json:"content"`
	Filename string  `json:"filename"`
	Title    string  `json:"title"`
	Folder   string  `json:"folder"`
	Score    float64 `json:"score"`
}

// ListDocumentsOpts filters knowledge list.
type ListDocumentsOpts struct {
	FolderPath   string
	FilterFolder bool
	Page         int
	PageSize     int
}
