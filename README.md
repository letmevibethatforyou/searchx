# SearchX

A Go library providing a unified search abstraction layer with support for multiple backends, including in-memory search and Algolia. SearchX offers a consistent API regardless of the underlying search implementation, making it easy to switch between different search providers.

## Features

- **Unified Search Interface**: Single API for multiple search backends
- **Type-Safe Expressions**: Rich query DSL with compile-time safety  
- **Multiple Backends**: In-memory and Algolia implementations included
- **AWS Integration**: Built-in support for AWS Secrets Manager and DynamoDB
- **Comprehensive Filtering**: Support for equality, comparison, range, and existence checks
- **Boolean Logic**: Combine filters with AND, OR, and NOT operations
- **Pagination**: Built-in offset/limit pagination support
- **Sorting**: Multi-field sorting with ascending/descending options
- **Full-Text Search**: Relevance-based text search with scoring
- **Thread-Safe**: Concurrent operations supported across all implementations

## Installation

```bash
go get github.com/letmevibethatforyou/searchx
```

## Quick Start

### Basic In-Memory Search

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/letmevibethatforyou/searchx"
    "github.com/letmevibethatforyou/searchx/inmemory"
)

func main() {
    // Create an in-memory searcher
    searcher := inmemory.New()
    
    // Add some documents
    searcher.AddDocument(inmemory.Document{
        ID: "1",
        Fields: map[string]interface{}{
            "title": "The Go Programming Language",
            "author": "Alan Donovan",
            "year": 2015,
            "price": 45.99,
        },
    })
    
    searcher.AddDocument(inmemory.Document{
        ID: "2", 
        Fields: map[string]interface{}{
            "title": "Effective Go",
            "author": "Rob Pike",
            "year": 2009,
            "price": 0.0,
        },
    })
    
    // Perform a search
    results, err := searcher.Search(context.Background(), "Go programming")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d results\n", results.Total)
    for _, item := range results.Items {
        fmt.Printf("- %s (Score: %.2f)\n", item.Fields["title"], item.Score)
    }
}
```

### Advanced Filtering with Expressions

```go
// Search for books published after 2010 with price less than $50
results, err := searcher.Search(
    context.Background(),
    "", // Empty query to match all documents
    searchx.And(
        searchx.Gt("year", 2010),
        searchx.Lt("price", 50.0),
    ),
    searchx.WithLimit(10),
    searchx.WithSort("year", true), // Sort by year descending
)
```

### Complex Query with Multiple Conditions

```go
// Find books by specific authors OR books published recently
complexFilter := searchx.Or(
    searchx.And(
        searchx.Eq("author", "Rob Pike"),
        searchx.Exists("title"),
    ),
    searchx.And(
        searchx.Gte("year", 2020),
        searchx.Lt("price", 100.0),
    ),
)

results, err := searcher.Search(
    context.Background(),
    "programming",
    complexFilter,
    searchx.WithLimit(5),
    searchx.WithOffset(0),
)
```

### Range and Existence Queries

```go
// Books priced between $20-$60 that have an ISBN
results, err := searcher.Search(
    context.Background(),
    "",
    searchx.And(
        searchx.Range("price", 20.0, 60.0),
        searchx.Exists("isbn"),
        searchx.Ne("status", "discontinued"),
    ),
)
```

## Algolia Integration

### Basic Algolia Setup with AWS Secrets Manager

```go
package main

import (
    "context"
    "log"

    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
    "github.com/letmevibethatforyou/searchx"
    "github.com/letmevibethatforyou/searchx/algolia"
)

func main() {
    ctx := context.Background()
    
    // Load AWS config
    cfg, err := config.LoadDefaultConfig(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    // Create Secrets Manager client
    secretsClient := secretsmanager.NewFromConfig(cfg)
    
    // Create Algolia client with AWS secrets
    fetchSecrets := algolia.AWSSecrets(ctx, secretsClient, "production")
    algoliaClient := algolia.NewClient(fetchSecrets)
    
    // Create searcher for specific index
    searcher := algolia.NewSearcher(algoliaClient, "products")
    
    // Search with filters
    results, err := searcher.Search(
        ctx,
        "smartphone",
        searchx.And(
            searchx.Eq("category", "electronics"),
            searchx.Gte("price", 100),
            searchx.Lt("price", 1000),
        ),
        searchx.WithLimit(20),
        searchx.WithSort("price", false), // Sort by price ascending
    )
    
    if err != nil {
        log.Fatal(err)
    }
    
    for _, item := range results.Items {
        fmt.Printf("Product: %s - $%.2f\n", 
            item.Fields["name"], 
            item.Fields["price"])
    }
}
```

### Alternative Credential Methods

```go
// Static credentials
fetchSecrets := algolia.StaticSecrets("YOUR_APP_ID", "YOUR_API_KEY")

// Environment variables (ALGOLIA_APP_ID, ALGOLIA_API_KEY)
fetchSecrets := algolia.EnvSecrets()
```

## Pagination Example

```go
func paginatedSearch(searcher searchx.Searcher, query string, pageSize int) {
    offset := 0
    
    for {
        results, err := searcher.Search(
            context.Background(),
            query,
            searchx.WithLimit(pageSize),
            searchx.WithOffset(offset),
        )
        if err != nil {
            log.Fatal(err)
        }
        
        // Process current page
        fmt.Printf("Page results (offset %d):\n", offset)
        for _, item := range results.Items {
            fmt.Printf("- %s\n", item.ID)
        }
        
        // Check if there are more results
        if results.NextOffset == nil {
            break
        }
        offset = *results.NextOffset
    }
}
```

## Multi-Field Sorting

```go
// Sort by relevance score (descending), then by publication year (descending), then by title (ascending)
results, err := searcher.Search(
    ctx,
    "programming",
    searchx.WithSort("_score", true),    // Primary: relevance descending
    searchx.WithSort("year", true),      // Secondary: year descending  
    searchx.WithSort("title", false),    // Tertiary: title ascending
)
```

## Error Handling

SearchX provides structured error handling with specific error codes:

```go
results, err := searcher.Search(ctx, "query")
if err != nil {
    switch {
    case errors.Is(err, searchx.ErrEmptyQuery):
        fmt.Println("Query cannot be empty")
    case errors.Is(err, searchx.ErrTimeout):
        fmt.Println("Search timed out")
    case errors.Is(err, searchx.ErrCanceled):
        fmt.Println("Search was canceled")
    case errors.Is(err, searchx.ErrBackendUnavailable):
        fmt.Println("Search backend is unavailable")
    default:
        log.Printf("Search failed: %v", err)
    }
}
```

## Command Line Tools

### Query Tool

Search Algolia indices from the command line:

```bash
# Basic search
go run cmd/query/main.go -env=prod -index=products "smartphone"

# Search with filters and sorting
go run cmd/query/main.go \
    -env=prod \
    -index=products \
    -filters="category=electronics,price>100,price<1000" \
    -sort-by="price:asc" \
    -page=0 \
    -hits-per-page=20 \
    "high-end smartphone"
```

### Data Generator

Generate test data for DynamoDB:

```bash
go run cmd/generator/main.go \
    -env=development \
    -table-name=products \
    -count=1000
```

## AWS Integration

### DynamoDB to Algolia Sync

The library includes a Lambda function for real-time synchronization from DynamoDB to Algolia:

```go
// Lambda function example
func main() {
    tableName := os.Getenv("TABLE_NAME")
    env := os.Getenv("ENV")
    
    cfg, _ := config.LoadDefaultConfig(context.Background())
    secretsClient := secretsmanager.NewFromConfig(cfg)
    fetchSecrets := algolia.AWSSecrets(context.Background(), secretsClient, env)
    
    handler := NewHandler(tableName, fetchSecrets)
    lambda.Start(handler.HandleDynamoDBEvent)
}
```

### Expected DynamoDB Structure

```json
{
    "pk": "unique-item-id",
    "sk": "index-name", 
    "object": {
        "title": "Product Name",
        "category": "electronics",
        "price": 299.99
    }
}
```

## Testing

Run the comprehensive test suite:

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test -v ./inmemory
go test -v ./algolia
```

## Performance Considerations

- **In-Memory**: Best for small datasets (< 1M documents), development, and testing
- **Algolia**: Recommended for production workloads with large datasets and advanced search requirements
- **Concurrent Access**: All implementations are thread-safe and support concurrent operations
- **Batch Operations**: For bulk indexing, use the appropriate backend's batch APIs

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass: `go test ./...`
5. Submit a pull request

## License

MIT License - see LICENSE file for details.