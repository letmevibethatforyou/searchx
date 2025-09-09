package inmemory

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/letmevibethatforyou/searchx"
)

func TestScoreDocument(t *testing.T) {
	searcher := New()

	doc := Document{
		ID: "1",
		Fields: map[string]interface{}{
			"title":       "Go Programming Language",
			"description": "Learn Go programming with examples",
			"tags":        []interface{}{"golang", "programming", "tutorial"},
			"nested": map[string]interface{}{
				"author": "John Doe",
				"year":   2023,
			},
			"rating": 4.5,
		},
	}

	tests := map[string]struct {
		query    string
		expected float64
	}{
		"empty_query": {
			query:    "",
			expected: 1.0,
		},
		"whitespace_query": {
			query:    "   ",
			expected: 1.0,
		},
		"single_term_match": {
			query:    "go",
			expected: 4.5, // Found in multiple fields, boosted
		},
		"single_term_case_insensitive": {
			query:    "GO",
			expected: 4.5,
		},
		"multiple_terms_all_match": {
			query:    "go programming",
			expected: 9.0, // Both terms match, with boost
		},
		"multiple_terms_partial_match": {
			query:    "go python",
			expected: 3.0, // Only "go" matches
		},
		"no_match": {
			query:    "javascript react",
			expected: 0,
		},
		"match_in_array": {
			query:    "golang",
			expected: 1.5, // Single term match with boost
		},
		"match_in_nested": {
			query:    "john",
			expected: 1.5,
		},
		"numeric_match": {
			query:    "2023",
			expected: 1.5,
		},
		"float_match": {
			query:    "4.5",
			expected: 1.5,
		},
		"partial_word_match": {
			query:    "program",
			expected: 4.5, // Matches "programming" in multiple places
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			score := searcher.scoreDocument(doc, tc.query)
			if score != tc.expected {
				t.Errorf("Expected score %f, got %f for query %q", tc.expected, score, tc.query)
			}
		})
	}
}

func TestValueContainsTerm(t *testing.T) {
	searcher := New()

	tests := map[string]struct {
		value    interface{}
		term     string
		expected bool
	}{
		"string_exact": {
			value:    "hello world",
			term:     "hello",
			expected: true,
		},
		"string_case_insensitive": {
			value:    "Hello World",
			term:     "hello",
			expected: true,
		},
		"string_no_match": {
			value:    "hello world",
			term:     "foo",
			expected: false,
		},
		"int_match": {
			value:    42,
			term:     "42",
			expected: true,
		},
		"float_match": {
			value:    3.14,
			term:     "3.14",
			expected: true,
		},
		"bool_true": {
			value:    true,
			term:     "true",
			expected: true,
		},
		"bool_false": {
			value:    false,
			term:     "false",
			expected: true,
		},
		"nil_value": {
			value:    nil,
			term:     "nil",
			expected: true,
		},
		"array_contains": {
			value:    []interface{}{"apple", "banana", "orange"},
			term:     "banana",
			expected: true,
		},
		"array_not_contains": {
			value:    []interface{}{"apple", "banana", "orange"},
			term:     "grape",
			expected: false,
		},
		"nested_array": {
			value:    []interface{}{[]interface{}{"nested", "value"}},
			term:     "nested",
			expected: true,
		},
		"map_contains_value": {
			value:    map[string]interface{}{"key": "value", "foo": "bar"},
			term:     "bar",
			expected: true,
		},
		"map_not_contains": {
			value:    map[string]interface{}{"key": "value"},
			term:     "notfound",
			expected: false,
		},
		"nested_map": {
			value: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": "target",
				},
			},
			term:     "target",
			expected: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := searcher.valueContainsTerm(tc.value, tc.term)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v for term %q in value %v", tc.expected, result, tc.term, tc.value)
			}
		})
	}
}

func TestSortMatches(t *testing.T) {
	searcher := New()

	docs := []scoredDocument{
		{
			document: Document{
				ID: "1",
				Fields: map[string]interface{}{
					"name":  "Alice",
					"age":   30,
					"score": 85.5,
				},
			},
			score: 0.8,
		},
		{
			document: Document{
				ID: "2",
				Fields: map[string]interface{}{
					"name":  "Bob",
					"age":   25,
					"score": 92.0,
				},
			},
			score: 0.9,
		},
		{
			document: Document{
				ID: "3",
				Fields: map[string]interface{}{
					"name":  "Charlie",
					"age":   35,
					"score": 78.0,
				},
			},
			score: 0.7,
		},
		{
			document: Document{
				ID: "4",
				Fields: map[string]interface{}{
					"name": "David",
					"age":  25,
					// score field missing
				},
			},
			score: 0.6,
		},
	}

	tests := map[string]struct {
		sortFields []searchx.SortField
		validate   func(t *testing.T, sorted []scoredDocument)
	}{
		"default_sort_by_score": {
			sortFields: nil,
			validate: func(t *testing.T, sorted []scoredDocument) {
				// Should be sorted by score descending
				expectedOrder := []string{"2", "1", "3", "4"}
				for i, id := range expectedOrder {
					if sorted[i].document.ID != id {
						t.Errorf("At index %d: expected ID %s, got %s", i, id, sorted[i].document.ID)
					}
				}
			},
		},
		"sort_by_age_ascending": {
			sortFields: []searchx.SortField{{Field: "age", Desc: false}},
			validate: func(t *testing.T, sorted []scoredDocument) {
				expectedOrder := []string{"2", "4", "1", "3"}
				for i, id := range expectedOrder {
					if sorted[i].document.ID != id {
						t.Errorf("At index %d: expected ID %s, got %s", i, id, sorted[i].document.ID)
					}
				}
			},
		},
		"sort_by_age_descending": {
			sortFields: []searchx.SortField{{Field: "age", Desc: true}},
			validate: func(t *testing.T, sorted []scoredDocument) {
				expectedOrder := []string{"3", "1", "2", "4"}
				for i, id := range expectedOrder {
					if sorted[i].document.ID != id {
						t.Errorf("At index %d: expected ID %s, got %s", i, id, sorted[i].document.ID)
					}
				}
			},
		},
		"sort_by_name_ascending": {
			sortFields: []searchx.SortField{{Field: "name", Desc: false}},
			validate: func(t *testing.T, sorted []scoredDocument) {
				expectedOrder := []string{"1", "2", "3", "4"}
				for i, id := range expectedOrder {
					if sorted[i].document.ID != id {
						t.Errorf("At index %d: expected ID %s, got %s", i, id, sorted[i].document.ID)
					}
				}
			},
		},
		"sort_by_score_field_with_nil": {
			sortFields: []searchx.SortField{{Field: "score", Desc: true}},
			validate: func(t *testing.T, sorted []scoredDocument) {
				// nil should sort last
				expectedOrder := []string{"2", "1", "3", "4"}
				for i, id := range expectedOrder {
					if sorted[i].document.ID != id {
						t.Errorf("At index %d: expected ID %s, got %s", i, id, sorted[i].document.ID)
					}
				}
			},
		},
		"multi_field_sort": {
			sortFields: []searchx.SortField{
				{Field: "age", Desc: false},
				{Field: "name", Desc: false},
			},
			validate: func(t *testing.T, sorted []scoredDocument) {
				// Age 25: Bob then David (by name)
				// Age 30: Alice
				// Age 35: Charlie
				expectedOrder := []string{"2", "4", "1", "3"}
				for i, id := range expectedOrder {
					if sorted[i].document.ID != id {
						t.Errorf("At index %d: expected ID %s, got %s", i, id, sorted[i].document.ID)
					}
				}
			},
		},
		"sort_by_relevance_score": {
			sortFields: []searchx.SortField{{Field: "_score", Desc: true}},
			validate: func(t *testing.T, sorted []scoredDocument) {
				expectedOrder := []string{"2", "1", "3", "4"}
				for i, id := range expectedOrder {
					if sorted[i].document.ID != id {
						t.Errorf("At index %d: expected ID %s, got %s", i, id, sorted[i].document.ID)
					}
				}
			},
		},
		"sort_by_relevance_score_ascending": {
			sortFields: []searchx.SortField{{Field: "_score", Desc: false}},
			validate: func(t *testing.T, sorted []scoredDocument) {
				expectedOrder := []string{"4", "3", "1", "2"}
				for i, id := range expectedOrder {
					if sorted[i].document.ID != id {
						t.Errorf("At index %d: expected ID %s, got %s", i, id, sorted[i].document.ID)
					}
				}
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Make a copy of docs to avoid modifying the original
			sorted := make([]scoredDocument, len(docs))
			copy(sorted, docs)

			searcher.sortMatches(sorted, tc.sortFields)
			tc.validate(t, sorted)
		})
	}
}

func TestSearchContextCancellation(t *testing.T) {
	searcher := New()

	// Add many documents to make search slower
	for i := 0; i < 1000; i++ {
		searcher.AddDocument(Document{
			ID: fmt.Sprintf("%d", i),
			Fields: map[string]interface{}{
				"content": fmt.Sprintf("Document content %d with some text", i),
			},
		})
	}

	tests := map[string]struct {
		setupContext func() (context.Context, context.CancelFunc)
		expectError  error
	}{
		"immediate_cancellation": {
			setupContext: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately
				return ctx, cancel
			},
			expectError: searchx.ErrCanceled,
		},
		"timeout_context": {
			setupContext: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
				time.Sleep(10 * time.Millisecond) // Ensure timeout
				return ctx, cancel
			},
			expectError: searchx.ErrCanceled,
		},
		"normal_context": {
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 1*time.Second)
			},
			expectError: nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := tc.setupContext()
			defer cancel()

			_, err := searcher.Search(ctx, "content")

			if tc.expectError != nil {
				if err != tc.expectError {
					t.Errorf("Expected error %v, got %v", tc.expectError, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestSearchPagination(t *testing.T) {
	searcher := New()

	// Add 15 documents
	for i := 1; i <= 15; i++ {
		searcher.AddDocument(Document{
			ID: fmt.Sprintf("%d", i),
			Fields: map[string]interface{}{
				"index": i,
				"text":  "matching content",
			},
		})
	}

	ctx := context.Background()

	tests := map[string]struct {
		offset      int
		limit       int
		expectTotal int64
		expectItems int
		expectNext  *int
	}{
		"first_page": {
			offset:      0,
			limit:       5,
			expectTotal: 15,
			expectItems: 5,
			expectNext:  intPtr(5),
		},
		"second_page": {
			offset:      5,
			limit:       5,
			expectTotal: 15,
			expectItems: 5,
			expectNext:  intPtr(10),
		},
		"last_page_full": {
			offset:      10,
			limit:       5,
			expectTotal: 15,
			expectItems: 5,
			expectNext:  nil,
		},
		"last_page_partial": {
			offset:      12,
			limit:       5,
			expectTotal: 15,
			expectItems: 3,
			expectNext:  nil,
		},
		"offset_beyond_results": {
			offset:      20,
			limit:       5,
			expectTotal: 15,
			expectItems: 0,
			expectNext:  nil,
		},
		"zero_limit_uses_default": {
			offset:      0,
			limit:       0,
			expectTotal: 15,
			expectItems: 10, // Default limit
			expectNext:  intPtr(10),
		},
		"large_limit": {
			offset:      0,
			limit:       100,
			expectTotal: 15,
			expectItems: 15,
			expectNext:  nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			opts := []searchx.SearchOption{
				searchx.WithOffset(tc.offset),
			}
			if tc.limit > 0 {
				opts = append(opts, searchx.WithLimit(tc.limit))
			}

			results, err := searcher.Search(ctx, "content", opts...)
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}

			if results.Total != tc.expectTotal {
				t.Errorf("Expected total %d, got %d", tc.expectTotal, results.Total)
			}

			if len(results.Items) != tc.expectItems {
				t.Errorf("Expected %d items, got %d", tc.expectItems, len(results.Items))
			}

			if tc.expectNext == nil {
				if results.NextOffset != nil {
					t.Errorf("Expected no NextOffset, got %d", *results.NextOffset)
				}
			} else {
				if results.NextOffset == nil {
					t.Errorf("Expected NextOffset %d, got nil", *tc.expectNext)
				} else if *results.NextOffset != *tc.expectNext {
					t.Errorf("Expected NextOffset %d, got %d", *tc.expectNext, *results.NextOffset)
				}
			}
		})
	}
}

func TestMatchesFilters(t *testing.T) {
	searcher := New()

	doc := Document{
		ID: "1",
		Fields: map[string]interface{}{
			"name":   "Test",
			"value":  42,
			"active": true,
		},
	}

	tests := map[string]struct {
		filters  []searchx.Expression
		expected bool
	}{
		"no_filters": {
			filters:  []searchx.Expression{},
			expected: true,
		},
		"nil_filters": {
			filters:  nil,
			expected: true,
		},
		"single_filter_match": {
			filters:  []searchx.Expression{searchx.Eq("name", "Test")},
			expected: true,
		},
		"single_filter_no_match": {
			filters:  []searchx.Expression{searchx.Eq("name", "Other")},
			expected: false,
		},
		"multiple_filters_all_match": {
			filters: []searchx.Expression{
				searchx.Eq("name", "Test"),
				searchx.Gt("value", 40),
				searchx.Eq("active", true),
			},
			expected: true,
		},
		"multiple_filters_one_fails": {
			filters: []searchx.Expression{
				searchx.Eq("name", "Test"),
				searchx.Gt("value", 50), // Fails
				searchx.Eq("active", true),
			},
			expected: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := searcher.matchesFilters(doc, tc.filters)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestSearchMetadata(t *testing.T) {
	searcher := New()
	searcher.AddDocument(Document{
		ID:     "1",
		Fields: map[string]interface{}{"text": "test"},
	})

	ctx := context.Background()

	tests := map[string]struct {
		query    string
		validate func(t *testing.T, results *searchx.Results)
	}{
		"query_stored": {
			query: "test query",
			validate: func(t *testing.T, results *searchx.Results) {
				if results.Query != "test query" {
					t.Errorf("Expected query 'test query', got %q", results.Query)
				}
			},
		},
		"took_time_recorded": {
			query: "test",
			validate: func(t *testing.T, results *searchx.Results) {
				if results.Took < 0 {
					t.Errorf("Expected non-negative Took time, got %d", results.Took)
				}
			},
		},
		"max_score_calculated": {
			query: "test",
			validate: func(t *testing.T, results *searchx.Results) {
				if results.MaxScore <= 0 {
					t.Errorf("Expected positive MaxScore, got %f", results.MaxScore)
				}
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			results, err := searcher.Search(ctx, tc.query)
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}
			tc.validate(t, results)
		})
	}
}

func TestSearchWithComplexDocuments(t *testing.T) {
	searcher := New()

	// Add complex nested documents
	searcher.AddDocument(Document{
		ID: "1",
		Fields: map[string]interface{}{
			"deeply": map[string]interface{}{
				"nested": map[string]interface{}{
					"structure": map[string]interface{}{
						"value": "hidden treasure",
					},
				},
			},
		},
	})

	searcher.AddDocument(Document{
		ID: "2",
		Fields: map[string]interface{}{
			"array_of_objects": []interface{}{
				map[string]interface{}{"name": "first", "value": 1},
				map[string]interface{}{"name": "second", "value": 2},
				map[string]interface{}{"name": "treasure", "value": 3},
			},
		},
	})

	ctx := context.Background()

	tests := map[string]struct {
		query       string
		expectCount int
	}{
		"find_deeply_nested": {
			query:       "treasure",
			expectCount: 2,
		},
		"find_in_array_of_objects": {
			query:       "second",
			expectCount: 1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			results, err := searcher.Search(ctx, tc.query)
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}
			if len(results.Items) != tc.expectCount {
				t.Errorf("Expected %d results, got %d", tc.expectCount, len(results.Items))
			}
		})
	}
}

// Helper function to create int pointer
func intPtr(v int) *int {
	return &v
}

func TestEmptySearcher(t *testing.T) {
	searcher := New()
	ctx := context.Background()

	// Test search on empty database
	results, err := searcher.Search(ctx, "anything")
	if err != nil {
		t.Fatalf("Search on empty database failed: %v", err)
	}

	if results.Total != 0 {
		t.Errorf("Expected 0 results, got %d", results.Total)
	}

	if len(results.Items) != 0 {
		t.Errorf("Expected empty items, got %d items", len(results.Items))
	}
}

func TestSpecialCharactersInSearch(t *testing.T) {
	searcher := New()

	searcher.AddDocument(Document{
		ID: "1",
		Fields: map[string]interface{}{
			"email":   "user@example.com",
			"url":     "https://example.com/path?query=value",
			"special": "hello-world_test.txt",
		},
	})

	ctx := context.Background()

	tests := map[string]struct {
		query    string
		expected bool
	}{
		"email_with_at": {
			query:    "user@example.com",
			expected: true,
		},
		"partial_email": {
			query:    "example.com",
			expected: true,
		},
		"url_with_protocol": {
			query:    "https://",
			expected: true,
		},
		"hyphen_underscore": {
			query:    "hello-world",
			expected: true,
		},
		"dot_in_filename": {
			query:    ".txt",
			expected: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			results, err := searcher.Search(ctx, tc.query)
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}

			found := len(results.Items) > 0
			if found != tc.expected {
				t.Errorf("Expected found=%v, got %v for query %q", tc.expected, found, tc.query)
			}
		})
	}
}

func TestSearchScoring(t *testing.T) {
	searcher := New()

	// Add documents with varying relevance
	searcher.AddDocument(Document{
		ID: "1",
		Fields: map[string]interface{}{
			"title":   "Go Programming",
			"content": "Learn Go programming language",
		},
	})

	searcher.AddDocument(Document{
		ID: "2",
		Fields: map[string]interface{}{
			"title":   "Python Guide",
			"content": "Go to Python documentation",
		},
	})

	searcher.AddDocument(Document{
		ID: "3",
		Fields: map[string]interface{}{
			"title":   "Going Forward",
			"content": "Strategic planning guide",
		},
	})

	ctx := context.Background()

	// Search for "go" - document 1 should score highest
	results, err := searcher.Search(ctx, "go programming")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results.Items) == 0 {
		t.Fatal("Expected results, got none")
	}

	// First result should be document 1 (most relevant)
	if results.Items[0].ID != "1" {
		t.Errorf("Expected document 1 to be first, got %s", results.Items[0].ID)
	}

	// Check that scores are in descending order
	for i := 1; i < len(results.Items); i++ {
		if results.Items[i].Score > results.Items[i-1].Score {
			t.Errorf("Scores not in descending order: %f > %f at index %d",
				results.Items[i].Score, results.Items[i-1].Score, i)
		}
	}
}

func TestConcurrentOperations(t *testing.T) {
	searcher := New()
	ctx := context.Background()

	// Test concurrent adds and searches
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			searcher.AddDocument(Document{
				ID: fmt.Sprintf("doc%d", i),
				Fields: map[string]interface{}{
					"index": i,
					"text":  fmt.Sprintf("Document number %d", i),
				},
			})
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_, err := searcher.Search(ctx, "document")
			if err != nil {
				t.Errorf("Search failed: %v", err)
			}
		}
		done <- true
	}()

	// Updater goroutine
	go func() {
		for i := 0; i < 50; i++ {
			searcher.AddDocument(Document{
				ID: fmt.Sprintf("doc%d", i),
				Fields: map[string]interface{}{
					"index":   i,
					"text":    fmt.Sprintf("Updated document %d", i),
					"updated": true,
				},
			})
		}
		done <- true
	}()

	// Deleter goroutine
	go func() {
		for i := 0; i < 25; i++ {
			searcher.RemoveDocument(fmt.Sprintf("doc%d", i))
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 4; i++ {
		<-done
	}

	// Verify data integrity
	size := searcher.Size()
	if size < 0 || size > 100 {
		t.Errorf("Unexpected size after concurrent operations: %d", size)
	}
}

func TestSearchMultipleWords(t *testing.T) {
	searcher := New()

	searcher.AddDocument(Document{
		ID: "1",
		Fields: map[string]interface{}{
			"content": "The quick brown fox jumps over the lazy dog",
		},
	})

	searcher.AddDocument(Document{
		ID: "2",
		Fields: map[string]interface{}{
			"content": "The dog is quick but not as quick as the fox",
		},
	})

	searcher.AddDocument(Document{
		ID: "3",
		Fields: map[string]interface{}{
			"content": "Brown bears are lazy in winter",
		},
	})

	ctx := context.Background()

	tests := map[string]struct {
		query          string
		expectedOrder  []string
		expectedScores map[string]float64
	}{
		"two_words_both_match": {
			query:         "quick fox",
			expectedOrder: []string{"1", "2"}, // Both have both terms
		},
		"three_words": {
			query:         "quick brown fox",
			expectedOrder: []string{"1", "2", "3"}, // 1 has all, 2 has quick+fox, 3 has brown
		},
		"repeated_word_boosts_score": {
			query:         "quick quick",
			expectedOrder: []string{"1", "2"}, // Both match, but scoring is based on all terms
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			results, err := searcher.Search(ctx, tc.query, searchx.WithLimit(10))
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}

			// Check order of results
			for i, expectedID := range tc.expectedOrder {
				if i >= len(results.Items) {
					t.Errorf("Expected result at index %d with ID %s, but got no result", i, expectedID)
					continue
				}
				if results.Items[i].ID != expectedID {
					t.Errorf("At index %d: expected ID %s, got %s", i, expectedID, results.Items[i].ID)
				}
			}
		})
	}
}

func TestUnicodeAndInternational(t *testing.T) {
	searcher := New()

	searcher.AddDocument(Document{
		ID: "1",
		Fields: map[string]interface{}{
			"text":  "Hello 世界 مرحبا мир",
			"emoji": "🚀 🌟 😊",
		},
	})

	searcher.AddDocument(Document{
		ID: "2",
		Fields: map[string]interface{}{
			"text": "Café français naïve",
		},
	})

	ctx := context.Background()

	tests := map[string]struct {
		query    string
		expected []string
	}{
		"chinese": {
			query:    "世界",
			expected: []string{"1"},
		},
		"arabic": {
			query:    "مرحبا",
			expected: []string{"1"},
		},
		"cyrillic": {
			query:    "мир",
			expected: []string{"1"},
		},
		"emoji": {
			query:    "🚀",
			expected: []string{"1"},
		},
		"french_accents": {
			query:    "français",
			expected: []string{"2"},
		},
		"mixed_unicode": {
			query:    "Hello",
			expected: []string{"1"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			results, err := searcher.Search(ctx, tc.query)
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}

			if len(results.Items) != len(tc.expected) {
				t.Errorf("Expected %d results, got %d", len(tc.expected), len(results.Items))
			}

			for i, expectedID := range tc.expected {
				if i >= len(results.Items) {
					break
				}
				if results.Items[i].ID != expectedID {
					t.Errorf("Expected ID %s, got %s", expectedID, results.Items[i].ID)
				}
			}
		})
	}
}

func TestLongQueryStrings(t *testing.T) {
	searcher := New()

	searcher.AddDocument(Document{
		ID: "1",
		Fields: map[string]interface{}{
			"content": "This is a document with many words that we will search for using a very long query string",
		},
	})

	ctx := context.Background()

	// Create a very long query
	words := []string{}
	for i := 0; i < 100; i++ {
		words = append(words, fmt.Sprintf("word%d", i))
	}
	longQuery := strings.Join(words, " ")

	// Should not panic or error
	results, err := searcher.Search(ctx, longQuery)
	if err != nil {
		t.Fatalf("Search with long query failed: %v", err)
	}

	// The query won't match, but it should execute successfully
	if results == nil {
		t.Error("Expected results object, got nil")
	}
}
