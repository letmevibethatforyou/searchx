package algolia

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/algolia/algoliasearch-client-go/v3/algolia/opt"
	"github.com/cockroachdb/errors"
	"github.com/letmevibethatforyou/searchx"
)

// Searcher implements the searchx.Searcher interface using Algolia.
type Searcher struct {
	client    *Client
	indexName string
}

// NewSearcher creates a new Algolia searcher for the specified index.
func NewSearcher(client *Client, indexName string) *Searcher {
	return &Searcher{
		client:    client,
		indexName: indexName,
	}
}

// Search implements the searchx.Searcher interface using Algolia search.
func (s *Searcher) Search(ctx context.Context, query string, opts ...searchx.SearchOption) (*searchx.Results, error) {
	startTime := time.Now()

	// Check context
	select {
	case <-ctx.Done():
		return nil, searchx.ErrCanceled
	default:
	}

	// Note: Unlike the inmemory searcher, we don't return ErrEmptyQuery for empty strings
	// because Algolia can handle empty queries and return all documents

	// Parse options
	cfg := &searchx.SearchConfig{}
	for _, opt := range opts {
		opt.Apply(cfg)
	}

	// Set defaults
	if cfg.Limit == 0 {
		cfg.Limit = 10
	}

	// Get Algolia client
	algoliaClient, err := s.client.getClient()
	if err != nil {
		return nil, errors.WithSecondaryError(
			searchx.ErrBackendUnavailable,
			errors.Wrapf(err, "failed to get Algolia client"),
		)
	}

	// Get index
	index := algoliaClient.InitIndex(s.indexName)

	// Build search parameters
	params := buildSearchParams(cfg)

	// Execute search
	res, err := index.Search(query, params...)
	if err != nil {
		// Check if this is a timeout or cancellation error
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, searchx.ErrTimeout
		}
		if errors.Is(err, context.Canceled) {
			return nil, searchx.ErrCanceled
		}

		// For other Algolia errors, treat as backend unavailable
		return nil, errors.WithSecondaryError(
			searchx.ErrBackendUnavailable,
			errors.Wrapf(err, "Algolia search failed"),
		)
	}

	// Convert results
	results := &searchx.Results{
		Items:    make([]searchx.Result, 0, len(res.Hits)),
		Total:    int64(res.NbHits),
		Query:    query,
		Took:     time.Since(startTime).Milliseconds(),
		MaxScore: 0.0,
	}

	// Convert hits to results
	for _, hit := range res.Hits {
		// Extract objectID
		objectID, ok := hit["objectID"].(string)
		if !ok {
			objectID = ""
		}

		// Calculate score (Algolia doesn't provide scores directly, use rank-based scoring)
		score := calculateScore(len(res.Hits), len(results.Items))
		if score > results.MaxScore {
			results.MaxScore = score
		}

		// Create result
		result := searchx.Result{
			ID:     objectID,
			Score:  score,
			Fields: hit,
		}

		results.Items = append(results.Items, result)
	}

	// Set next offset for pagination
	nextPage := res.Page + 1
	if nextPage < res.NbPages {
		nextOffset := nextPage * cfg.Limit
		results.NextOffset = &nextOffset
	}

	return results, nil
}

// buildSearchParams converts searchx.SearchConfig to Algolia search parameters
func buildSearchParams(cfg *searchx.SearchConfig) []interface{} {
	var params []interface{}

	// Set pagination
	params = append(params, opt.HitsPerPage(cfg.Limit))
	if cfg.Offset > 0 {
		page := cfg.Offset / cfg.Limit
		params = append(params, opt.Page(page))
	}

	// Convert filters
	if len(cfg.Filters) > 0 {
		filterStrings := make([]string, 0, len(cfg.Filters))
		for _, expr := range cfg.Filters {
			if filterStr := convertExpressionToFilter(expr); filterStr != "" {
				filterStrings = append(filterStrings, filterStr)
			}
		}
		if len(filterStrings) > 0 {
			params = append(params, opt.Filters(strings.Join(filterStrings, " AND ")))
		}
	}

	// Convert sorting
	if len(cfg.Sort) > 0 {
		sortFields := make([]string, 0, len(cfg.Sort))
		for _, sort := range cfg.Sort {
			if sort.Field == "_score" {
				// Algolia handles relevance sorting automatically
				continue
			}
			if sort.Desc {
				sortFields = append(sortFields, sort.Field+":desc")
			} else {
				sortFields = append(sortFields, sort.Field+":asc")
			}
		}
		if len(sortFields) > 0 {
			// For Algolia, we would need to use replica indices for custom sorting
			// For now, we'll add this as a comment for future enhancement
			// This would require creating replica indices with custom ranking
		}
	}

	return params
}

// calculateScore creates a rank-based score for Algolia results
// Since Algolia doesn't provide relevance scores directly, we use position-based scoring
func calculateScore(totalResults, position int) float64 {
	if totalResults == 0 {
		return 1.0
	}
	// Higher positions get higher scores (inverse rank)
	return float64(totalResults-position) / float64(totalResults)
}

// convertExpressionToFilter converts a searchx expression to an Algolia filter string
func convertExpressionToFilter(expr searchx.Expression) string {
	switch e := expr.(type) {
	case searchx.AndExpr:
		return convertAndExpression(e)
	case searchx.OrExpr:
		return convertOrExpression(e)
	case searchx.NotExpr:
		return convertNotExpression(e)
	case searchx.EqExpr:
		return convertEqExpression(e)
	case searchx.NeExpr:
		return convertNeExpression(e)
	case searchx.GtExpr:
		return convertGtExpression(e)
	case searchx.GteExpr:
		return convertGteExpression(e)
	case searchx.LtExpr:
		return convertLtExpression(e)
	case searchx.LteExpr:
		return convertLteExpression(e)
	case searchx.RangeExpr:
		return convertRangeExpression(e)
	case searchx.ExistsExpr:
		return convertExistsExpression(e)
	default:
		return ""
	}
}

// convertAndExpression converts an AND expression to Algolia filter syntax
func convertAndExpression(expr searchx.AndExpr) string {
	filters := make([]string, 0, len(expr.Exprs))
	for _, e := range expr.Exprs {
		if filter := convertExpressionToFilter(e); filter != "" {
			filters = append(filters, "("+filter+")")
		}
	}
	if len(filters) == 0 {
		return ""
	}
	return strings.Join(filters, " AND ")
}

// convertOrExpression converts an OR expression to Algolia filter syntax
func convertOrExpression(expr searchx.OrExpr) string {
	filters := make([]string, 0, len(expr.Exprs))
	for _, e := range expr.Exprs {
		if filter := convertExpressionToFilter(e); filter != "" {
			filters = append(filters, "("+filter+")")
		}
	}
	if len(filters) == 0 {
		return ""
	}
	return strings.Join(filters, " OR ")
}

// convertNotExpression converts a NOT expression to Algolia filter syntax
func convertNotExpression(expr searchx.NotExpr) string {
	inner := convertExpressionToFilter(expr.Inner)
	if inner == "" {
		return ""
	}
	return "NOT (" + inner + ")"
}

// convertEqExpression converts an equality expression to Algolia filter syntax
func convertEqExpression(expr searchx.EqExpr) string {
	return fmt.Sprintf("%s:%s", escapeField(expr.Field), escapeValue(expr.Value))
}

// convertNeExpression converts a not-equal expression to Algolia filter syntax
func convertNeExpression(expr searchx.NeExpr) string {
	return fmt.Sprintf("NOT %s:%s", escapeField(expr.Field), escapeValue(expr.Value))
}

// convertGtExpression converts a greater-than expression to Algolia filter syntax
func convertGtExpression(expr searchx.GtExpr) string {
	return fmt.Sprintf("%s > %s", escapeField(expr.Field), escapeNumericValue(expr.Value))
}

// convertGteExpression converts a greater-than-or-equal expression to Algolia filter syntax
func convertGteExpression(expr searchx.GteExpr) string {
	return fmt.Sprintf("%s >= %s", escapeField(expr.Field), escapeNumericValue(expr.Value))
}

// convertLtExpression converts a less-than expression to Algolia filter syntax
func convertLtExpression(expr searchx.LtExpr) string {
	return fmt.Sprintf("%s < %s", escapeField(expr.Field), escapeNumericValue(expr.Value))
}

// convertLteExpression converts a less-than-or-equal expression to Algolia filter syntax
func convertLteExpression(expr searchx.LteExpr) string {
	return fmt.Sprintf("%s <= %s", escapeField(expr.Field), escapeNumericValue(expr.Value))
}

// convertRangeExpression converts a range expression to Algolia filter syntax
func convertRangeExpression(expr searchx.RangeExpr) string {
	var filters []string

	if expr.Min != nil {
		filters = append(filters, fmt.Sprintf("%s >= %s", escapeField(expr.Field), escapeNumericValue(expr.Min)))
	}

	if expr.Max != nil {
		filters = append(filters, fmt.Sprintf("%s <= %s", escapeField(expr.Field), escapeNumericValue(expr.Max)))
	}

	if len(filters) == 0 {
		return ""
	}

	return strings.Join(filters, " AND ")
}

// convertExistsExpression converts an exists expression to Algolia filter syntax
func convertExistsExpression(expr searchx.ExistsExpr) string {
	return fmt.Sprintf("%s:*", escapeField(expr.Field))
}

// escapeField escapes field names for Algolia filters
func escapeField(field string) string {
	// Algolia field names with special characters should be quoted
	if strings.ContainsAny(field, " :-()") {
		return fmt.Sprintf(`"%s"`, field)
	}
	return field
}

// escapeValue escapes string values for Algolia filters
func escapeValue(value interface{}) string {
	if value == nil {
		return "null"
	}

	switch v := value.(type) {
	case string:
		// Quote string values and escape internal quotes
		escaped := strings.ReplaceAll(v, `"`, `\"`)
		return fmt.Sprintf(`"%s"`, escaped)
	case bool:
		return fmt.Sprintf(`"%s"`, strconv.FormatBool(v))
	default:
		return fmt.Sprintf(`"%v"`, value)
	}
}

// escapeNumericValue escapes numeric values for Algolia filters
func escapeNumericValue(value interface{}) string {
	if value == nil {
		return "0"
	}

	switch v := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%v", v)
	case float32, float64:
		return fmt.Sprintf("%v", v)
	default:
		// Try to parse as number, otherwise treat as string
		if str := fmt.Sprintf("%v", value); str != "" {
			if _, err := strconv.ParseFloat(str, 64); err == nil {
				return str
			}
		}
		return escapeValue(value)
	}
}
