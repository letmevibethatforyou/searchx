package inmemory

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/letmevibethatforyou/searchx"
)

func TestInMemorySearcher(t *testing.T) {
	searcher := New()

	// Add test documents
	docs := []struct {
		id   string
		data string
	}{
		{"1", `{"title": "Go Programming", "author": "John Doe", "year": 2020, "category": "programming", "price": 29.99}`},
		{"2", `{"title": "Python Basics", "author": "Jane Smith", "year": 2021, "category": "programming", "price": 24.99}`},
		{"3", `{"title": "Data Science", "author": "Bob Johnson", "year": 2020, "category": "data", "price": 39.99}`},
		{"4", `{"title": "Machine Learning", "author": "Alice Brown", "year": 2022, "category": "data", "price": 44.99}`},
		{"5", `{"title": "Web Development", "author": "John Doe", "year": 2021, "category": "programming", "price": 34.99}`},
	}

	for _, doc := range docs {
		if err := searcher.AddJSON(doc.id, []byte(doc.data)); err != nil {
			t.Fatalf("Failed to add document %s: %v", doc.id, err)
		}
	}

	ctx := context.Background()

	t.Run("BasicSearch", func(t *testing.T) {
		results, err := searcher.Search(ctx, "programming")
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if results.Total != 3 {
			t.Errorf("Expected 3 results, got %d", results.Total)
		}
	})

	t.Run("EmptyQuery", func(t *testing.T) {
		results, err := searcher.Search(ctx, "", searchx.WithLimit(5))
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if results.Total != 5 {
			t.Errorf("Expected 5 results, got %d", results.Total)
		}
	})

	t.Run("WithFilters", func(t *testing.T) {
		results, err := searcher.Search(ctx, "",
			searchx.Eq("category", "programming"),
			searchx.WithLimit(10),
		)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if results.Total != 3 {
			t.Errorf("Expected 3 programming results, got %d", results.Total)
		}
	})

	t.Run("RangeFilter", func(t *testing.T) {
		results, err := searcher.Search(ctx, "",
			searchx.Range("price", 25.0, 35.0),
			searchx.WithLimit(10),
		)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if results.Total != 2 {
			t.Errorf("Expected 2 results in price range, got %d", results.Total)
		}
	})

	t.Run("ComparisonOperators", func(t *testing.T) {
		results, err := searcher.Search(ctx, "",
			searchx.Gt("year", 2020),
			searchx.WithLimit(10),
		)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if results.Total != 3 {
			t.Errorf("Expected 3 results after 2020, got %d", results.Total)
		}
	})

	t.Run("AndExpression", func(t *testing.T) {
		results, err := searcher.Search(ctx, "",
			searchx.And(
				searchx.Eq("category", "programming"),
				searchx.Gte("year", 2021),
			),
			searchx.WithLimit(10),
		)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if results.Total != 2 {
			t.Errorf("Expected 2 results, got %d", results.Total)
		}
	})

	t.Run("OrExpression", func(t *testing.T) {
		results, err := searcher.Search(ctx, "",
			searchx.Or(
				searchx.Eq("category", "data"),
				searchx.Eq("author", "John Doe"),
			),
			searchx.WithLimit(10),
		)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if results.Total != 4 {
			t.Errorf("Expected 4 results, got %d", results.Total)
		}
	})

	t.Run("NotExpression", func(t *testing.T) {
		results, err := searcher.Search(ctx, "",
			searchx.Not(searchx.Eq("category", "programming")),
			searchx.WithLimit(10),
		)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if results.Total != 2 {
			t.Errorf("Expected 2 non-programming results, got %d", results.Total)
		}
	})

	t.Run("ComplexExpression", func(t *testing.T) {
		results, err := searcher.Search(ctx, "",
			searchx.And(
				searchx.Or(
					searchx.Eq("category", "programming"),
					searchx.Eq("category", "data"),
				),
				searchx.Gte("price", 30.0),
			),
			searchx.WithLimit(10),
		)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if results.Total != 3 {
			t.Errorf("Expected 3 results, got %d", results.Total)
		}
	})

	t.Run("Pagination", func(t *testing.T) {
		// First page
		results1, err := searcher.Search(ctx, "",
			searchx.WithLimit(2),
			searchx.WithOffset(0),
		)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(results1.Items) != 2 {
			t.Errorf("Expected 2 items in first page, got %d", len(results1.Items))
		}

		if results1.NextOffset == nil || *results1.NextOffset != 2 {
			t.Error("Expected NextOffset to be 2")
		}

		// Second page
		results2, err := searcher.Search(ctx, "",
			searchx.WithLimit(2),
			searchx.WithOffset(2),
		)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(results2.Items) != 2 {
			t.Errorf("Expected 2 items in second page, got %d", len(results2.Items))
		}

		// Ensure different items
		if results1.Items[0].ID == results2.Items[0].ID {
			t.Error("Pages contain the same items")
		}
	})

	t.Run("Sorting", func(t *testing.T) {
		results, err := searcher.Search(ctx, "",
			searchx.WithSort("year", false), // ascending
			searchx.WithLimit(5),
		)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Check that years are in ascending order
		prevYear := 0
		for _, item := range results.Items {
			year, ok := item.Fields["year"].(float64)
			if !ok {
				t.Error("Year field not found or not a number")
				continue
			}
			if int(year) < prevYear {
				t.Error("Results not sorted correctly by year")
			}
			prevYear = int(year)
		}
	})

	t.Run("ExistsFilter", func(t *testing.T) {
		// Add a document without a price field
		err := searcher.AddJSON("6", []byte(`{"title": "Free Book", "author": "Someone", "year": 2023, "category": "other"}`))
		if err != nil {
			t.Fatalf("Failed to add document 6: %v", err)
		}

		results, err := searcher.Search(ctx, "",
			searchx.Not(searchx.Exists("price")),
			searchx.WithLimit(10),
		)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if results.Total != 1 {
			t.Errorf("Expected 1 result without price, got %d", results.Total)
		}
	})
}

func TestDocumentOperations(t *testing.T) {
	searcher := New()

	// Test adding documents
	doc1 := Document{
		ID: "test1",
		Fields: map[string]interface{}{
			"name":  "Test Document",
			"value": 42,
		},
	}
	searcher.AddDocument(doc1)

	if searcher.Size() != 1 {
		t.Errorf("Expected size 1, got %d", searcher.Size())
	}

	// Test updating document
	doc1Updated := Document{
		ID: "test1",
		Fields: map[string]interface{}{
			"name":  "Updated Document",
			"value": 100,
		},
	}
	searcher.AddDocument(doc1Updated)

	if searcher.Size() != 1 {
		t.Errorf("Expected size 1 after update, got %d", searcher.Size())
	}

	// Verify update
	ctx := context.Background()
	results, err := searcher.Search(ctx, "Updated")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if results.Total != 1 {
		t.Error("Updated document not found")
	}

	// Test removing document
	removed := searcher.RemoveDocument("test1")
	if !removed {
		t.Error("Failed to remove document")
	}

	if searcher.Size() != 0 {
		t.Errorf("Expected size 0 after removal, got %d", searcher.Size())
	}

	// Test removing non-existent document
	removed = searcher.RemoveDocument("nonexistent")
	if removed {
		t.Error("Should not remove non-existent document")
	}

	// Test Clear
	searcher.AddDocument(doc1)
	err = searcher.AddJSON("test2", []byte(`{"name": "Another"}`))
	if err != nil {
		t.Fatalf("Failed to add JSON document: %v", err)
	}
	searcher.Clear()

	if searcher.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", searcher.Size())
	}
}

func TestSearcherFunc(t *testing.T) {
	// Create a custom searcher using SearcherFunc
	var customSearcher searchx.SearcherFunc = func(ctx context.Context, query string, opts ...searchx.SearchOption) (*searchx.Results, error) {
		return &searchx.Results{
			Items: []searchx.Result{
				{ID: "custom1", Score: 1.0, Fields: map[string]interface{}{"test": true}},
			},
			Total: 1,
		}, nil
	}

	ctx := context.Background()
	results, err := customSearcher.Search(ctx, "test")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if results.Total != 1 {
		t.Errorf("Expected 1 result, got %d", results.Total)
	}
}

func BenchmarkSearch(b *testing.B) {
	searcher := New()

	// Add 1000 documents
	for i := 0; i < 1000; i++ {
		data := map[string]interface{}{
			"id":       i,
			"title":    "Document " + string(rune(i)),
			"category": []string{"cat1", "cat2", "cat3"}[i%3],
			"score":    float64(i % 100),
		}
		jsonData, _ := json.Marshal(data)
		_ = searcher.AddJSON(string(rune(i)), jsonData)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = searcher.Search(ctx, "Document",
			searchx.Range("score", 25.0, 75.0),
			searchx.WithLimit(10),
		)
	}
}
