package searchx

// SearchOption represents a search configuration option.
type SearchOption interface {
	Apply(*SearchConfig)
}

// SearchConfig holds all search configuration parameters.
type SearchConfig struct {
	// Limit specifies the maximum number of results to return.
	Limit int

	// Offset specifies the number of results to skip for pagination.
	Offset int

	// Sort specifies sorting configuration.
	Sort []SortField

	// Filters contains filter expressions to apply.
	Filters []Expression
}

// SortField represents a field to sort by.
type SortField struct {
	// Field is the name of the field to sort by.
	Field string
	// Desc indicates whether to sort in descending order (true) or ascending order (false).
	Desc bool
}

// optionFunc is a function that implements SearchOption.
type optionFunc func(*SearchConfig)

// Apply implements the SearchOption interface for optionFunc.
func (f optionFunc) Apply(cfg *SearchConfig) {
	f(cfg)
}

// WithLimit sets the maximum number of results to return.
func WithLimit(n int) SearchOption {
	return optionFunc(func(cfg *SearchConfig) {
		cfg.Limit = n
	})
}

// WithOffset sets the number of results to skip for pagination.
func WithOffset(n int) SearchOption {
	return optionFunc(func(cfg *SearchConfig) {
		cfg.Offset = n
	})
}

// WithSort adds a sort field to the search.
func WithSort(field string, desc bool) SearchOption {
	return optionFunc(func(cfg *SearchConfig) {
		cfg.Sort = append(cfg.Sort, SortField{Field: field, Desc: desc})
	})
}
