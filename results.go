package searchx

// Result represents a single search result.
type Result struct {
	// ID is the unique identifier of the result.
	ID string

	// Score represents the relevance score of this result.
	Score float64

	// Fields contains the document fields as key-value pairs.
	Fields map[string]interface{}
}

// Results represents a collection of search results with metadata.
type Results struct {
	// Items contains the individual search results.
	Items []Result

	// Total is the total number of matching documents.
	Total int64

	// Took is the time taken to execute the search in milliseconds.
	Took int64

	// MaxScore is the maximum relevance score across all results.
	MaxScore float64

	// Query is the original query string for reference.
	Query string

	// NextOffset can be used for pagination.
	NextOffset *int
}
