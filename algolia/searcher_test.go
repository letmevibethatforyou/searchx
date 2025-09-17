package algolia

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/letmevibethatforyou/searchx"
)

func TestNewSearcher(t *testing.T) {
	client := NewClient(StaticSecrets("test-app", "test-key"))
	searcher := NewSearcher(client, "test-index")

	if searcher == nil {
		t.Fatal("NewSearcher returned nil")
	}

	if searcher.client != client {
		t.Error("Searcher client not set correctly")
	}

	if searcher.indexName != "test-index" {
		t.Errorf("Expected index name 'test-index', got '%s'", searcher.indexName)
	}
}

func TestBuildSearchParams(t *testing.T) {
	tests := []struct {
		name          string
		config        *searchx.SearchConfig
		expectedCount int
	}{
		{
			name: "default parameters",
			config: &searchx.SearchConfig{
				Limit: 10,
			},
			expectedCount: 1, // HitsPerPage option
		},
		{
			name: "with offset",
			config: &searchx.SearchConfig{
				Limit:  20,
				Offset: 40,
			},
			expectedCount: 2, // HitsPerPage and Page options
		},
		{
			name: "with filters",
			config: &searchx.SearchConfig{
				Limit: 10,
				Filters: []searchx.Expression{
					searchx.Eq("status", "active"),
				},
			},
			expectedCount: 2, // HitsPerPage and Filters options
		},
		{
			name: "with sort fields",
			config: &searchx.SearchConfig{
				Limit: 10,
				Sort: []searchx.SortField{
					{Field: "title", Desc: false},
					{Field: "date", Desc: true},
				},
			},
			expectedCount: 1, // HitsPerPage only (sorting needs replica indices)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := buildSearchParams(tt.config)
			if len(params) != tt.expectedCount {
				t.Errorf("Expected %d parameters, got %d", tt.expectedCount, len(params))
			}
		})
	}
}

func TestCalculateScore(t *testing.T) {
	tests := []struct {
		name         string
		totalResults int
		position     int
		expected     float64
	}{
		{
			name:         "first result",
			totalResults: 10,
			position:     0,
			expected:     1.0,
		},
		{
			name:         "middle result",
			totalResults: 10,
			position:     4,
			expected:     0.6,
		},
		{
			name:         "last result",
			totalResults: 10,
			position:     9,
			expected:     0.1,
		},
		{
			name:         "no results",
			totalResults: 0,
			position:     0,
			expected:     1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateScore(tt.totalResults, tt.position)
			if score != tt.expected {
				t.Errorf("Expected score %f, got %f", tt.expected, score)
			}
		})
	}
}

func TestConvertExpressionToFilter(t *testing.T) {
	tests := []struct {
		name     string
		expr     searchx.Expression
		expected string
	}{
		{
			name:     "equality expression",
			expr:     searchx.Eq("status", "active"),
			expected: `status:"active"`,
		},
		{
			name:     "not equal expression",
			expr:     searchx.Ne("status", "inactive"),
			expected: `NOT status:"inactive"`,
		},
		{
			name:     "greater than expression",
			expr:     searchx.Gt("price", 100),
			expected: "price > 100",
		},
		{
			name:     "greater than or equal expression",
			expr:     searchx.Gte("price", 50),
			expected: "price >= 50",
		},
		{
			name:     "less than expression",
			expr:     searchx.Lt("price", 200),
			expected: "price < 200",
		},
		{
			name:     "less than or equal expression",
			expr:     searchx.Lte("price", 150),
			expected: "price <= 150",
		},
		{
			name:     "range expression",
			expr:     searchx.Range("price", 50, 200),
			expected: "price >= 50 AND price <= 200",
		},
		{
			name:     "range expression with nil min",
			expr:     searchx.Range("price", nil, 200),
			expected: "price <= 200",
		},
		{
			name:     "range expression with nil max",
			expr:     searchx.Range("price", 50, nil),
			expected: "price >= 50",
		},
		{
			name:     "exists expression",
			expr:     searchx.Exists("description"),
			expected: "description:*",
		},
		{
			name:     "AND expression",
			expr:     searchx.And(searchx.Eq("status", "active"), searchx.Gt("price", 100)),
			expected: `(status:"active") AND (price > 100)`,
		},
		{
			name:     "OR expression",
			expr:     searchx.Or(searchx.Eq("category", "electronics"), searchx.Eq("category", "books")),
			expected: `(category:"electronics") OR (category:"books")`,
		},
		{
			name:     "NOT expression",
			expr:     searchx.Not(searchx.Eq("status", "deleted")),
			expected: `NOT (status:"deleted")`,
		},
		{
			name: "complex nested expression",
			expr: searchx.And(
				searchx.Eq("status", "active"),
				searchx.Or(
					searchx.Gt("price", 100),
					searchx.Eq("featured", true),
				),
			),
			expected: `(status:"active") AND ((price > 100) OR (featured:"true"))`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertExpressionToFilter(tt.expr)
			if result != tt.expected {
				t.Errorf("Expected filter '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestEscapeField(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		expected string
	}{
		{
			name:     "simple field",
			field:    "title",
			expected: "title",
		},
		{
			name:     "field with space",
			field:    "product name",
			expected: `"product name"`,
		},
		{
			name:     "field with colon",
			field:    "user:id",
			expected: `"user:id"`,
		},
		{
			name:     "field with dash",
			field:    "created-at",
			expected: `"created-at"`,
		},
		{
			name:     "field with parentheses",
			field:    "count(items)",
			expected: `"count(items)"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeField(tt.field)
			if result != tt.expected {
				t.Errorf("Expected escaped field '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestEscapeValue(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{
			name:     "string value",
			value:    "test",
			expected: `"test"`,
		},
		{
			name:     "string with quotes",
			value:    `hello "world"`,
			expected: `"hello \"world\""`,
		},
		{
			name:     "boolean true",
			value:    true,
			expected: `"true"`,
		},
		{
			name:     "boolean false",
			value:    false,
			expected: `"false"`,
		},
		{
			name:     "nil value",
			value:    nil,
			expected: "null",
		},
		{
			name:     "integer value",
			value:    42,
			expected: `"42"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeValue(tt.value)
			if result != tt.expected {
				t.Errorf("Expected escaped value '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestEscapeNumericValue(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{
			name:     "integer",
			value:    42,
			expected: "42",
		},
		{
			name:     "float",
			value:    3.14,
			expected: "3.14",
		},
		{
			name:     "negative number",
			value:    -100,
			expected: "-100",
		},
		{
			name:     "string number",
			value:    "123.45",
			expected: "123.45",
		},
		{
			name:     "non-numeric string",
			value:    "abc",
			expected: `"abc"`,
		},
		{
			name:     "nil",
			value:    nil,
			expected: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeNumericValue(tt.value)
			if result != tt.expected {
				t.Errorf("Expected numeric value '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestSearcherInterface verifies that Searcher implements the searchx.Searcher interface
func TestSearcherInterface(t *testing.T) {
	client := NewClient(StaticSecrets("test-app", "test-key"))
	searcher := NewSearcher(client, "test-index")

	// This should compile if Searcher implements searchx.Searcher
	var _ searchx.Searcher = searcher
}

// TestSearchWithInvalidClient tests error handling when client initialization fails
func TestSearchWithInvalidClient(t *testing.T) {
	// Create a client with invalid credentials
	fetchSecrets := func() (Secrets, error) {
		return Secrets{}, fmt.Errorf("failed to fetch secrets")
	}
	client := NewClient(fetchSecrets)
	searcher := NewSearcher(client, "test-index")

	ctx := context.Background()
	_, err := searcher.Search(ctx, "test query")

	if err == nil {
		t.Error("Expected error when client initialization fails, got nil")
	}

	// Check that we get the correct error code
	if !errors.Is(err, searchx.ErrBackendUnavailable) {
		t.Errorf("Expected ErrBackendUnavailable, got: %v", err)
	}

	// Check that the error details contain the wrapped error
	errStr := fmt.Sprintf("%+v", err)
	if !strings.Contains(errStr, "failed to get Algolia client") {
		t.Errorf("Expected error details to contain 'failed to get Algolia client', got: %v", errStr)
	}
}

// TestSearchWithCanceledContext tests context cancellation
func TestSearchWithCanceledContext(t *testing.T) {
	client := NewClient(StaticSecrets("test-app", "test-key"))
	searcher := NewSearcher(client, "test-index")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := searcher.Search(ctx, "test query")

	if err != searchx.ErrCanceled {
		t.Errorf("Expected ErrCanceled, got: %v", err)
	}
}

// TestSearchWithTimeout tests context timeout
func TestSearchWithTimeout(t *testing.T) {
	client := NewClient(StaticSecrets("test-app", "test-key"))
	searcher := NewSearcher(client, "test-index")

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Sleep to ensure timeout
	time.Sleep(1 * time.Millisecond)

	_, err := searcher.Search(ctx, "test query")

	if err != searchx.ErrCanceled {
		t.Errorf("Expected ErrCanceled for timed out context, got: %v", err)
	}
}

// TestErrorCodeUsage verifies that we're using the correct error codes
func TestErrorCodeUsage(t *testing.T) {
	tests := []struct {
		name          string
		setupClient   func() *Client
		expectedError error
		description   string
	}{
		{
			name: "invalid credentials",
			setupClient: func() *Client {
				fetchSecrets := func() (Secrets, error) {
					return Secrets{}, fmt.Errorf("secrets unavailable")
				}
				return NewClient(fetchSecrets)
			},
			expectedError: searchx.ErrBackendUnavailable,
			description:   "client initialization failure should return ErrBackendUnavailable",
		},
		{
			name: "empty credentials",
			setupClient: func() *Client {
				fetchSecrets := func() (Secrets, error) {
					return Secrets{ApplicationID: "", WriteApiKey: ""}, nil
				}
				return NewClient(fetchSecrets)
			},
			expectedError: searchx.ErrBackendUnavailable,
			description:   "empty credentials should return ErrBackendUnavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()
			searcher := NewSearcher(client, "test-index")

			ctx := context.Background()
			_, err := searcher.Search(ctx, "test query")

			if err == nil {
				t.Errorf("Expected error for %s, got nil", tt.description)
				return
			}

			if !errors.Is(err, tt.expectedError) {
				t.Errorf("Expected %v for %s, got: %v", tt.expectedError, tt.description, err)
			}
		})
	}
}
