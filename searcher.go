package searchx

import "context"

// Searcher defines the core search interface.
type Searcher interface {
	// Search executes a search with the given query and options.
	Search(ctx context.Context, query string, opts ...SearchOption) (*Results, error)
}

// SearcherFunc is a function type that implements the Searcher interface.
// This allows using a function as a Searcher, similar to http.HandlerFunc.
type SearcherFunc func(context.Context, string, ...SearchOption) (*Results, error)

// Search implements the Searcher interface for SearcherFunc.
func (f SearcherFunc) Search(ctx context.Context, query string, opts ...SearchOption) (*Results, error) {
	return f(ctx, query, opts...)
}
