# SearchX

SearchX is a Go library that provides a unified search interface with pluggable backend implementations. It's designed to abstract search operations and allow easy switching between different search providers.

## Features

- **Unified Interface**: Single `Searcher` interface for all search operations
- **Multiple Backends**: Currently supports Algolia with extensible architecture for additional providers
- **Advanced Options**: Support for filters, facets, pagination, and custom expressions
- **AWS Integration**: Built-in support for AWS Lambda functions and DynamoDB
- **Error Handling**: Comprehensive error codes and structured error handling
- **Cloud Deployment**: CloudFormation template included for AWS deployment

## Installation

```bash
go get github.com/letmevibethatforyou/searchx
```

## Quick Start

### Basic Search

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/letmevibethatforyou/searchx"
    "github.com/letmevibethatforyou/searchx/algolia"
)

func main() {
    // Create an Algolia client
    client := algolia.NewClient("your-app-id", "your-api-key")

    // Create a searcher for your index
    searcher := algolia.NewSearcher(client, "your-index-name")

    // Perform a search
    results, err := searcher.Search(context.Background(), "search query")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Found %d results\n", results.Total)
}
```

### Advanced Search with Options

```go
results, err := searcher.Search(
    context.Background(),
    "search query",
    searchx.WithLimit(20),
    searchx.WithOffset(10),
    searchx.WithFacets("category", "brand"),
    searchx.WithFilters(searchx.Filter{
        Field: "price",
        Op:    searchx.OpGte,
        Value: 100,
    }),
)
```

## Search Options

SearchX supports various search options:

- **Pagination**: `WithLimit()`, `WithOffset()`
- **Filtering**: `WithFilters()` with operators (`OpEq`, `OpNe`, `OpGt`, `OpGte`, `OpLt`, `OpLte`, `OpExists`)
- **Faceting**: `WithFacets()`
- **Sorting**: `WithSort()`
- **Timeouts**: `WithTimeout()`
- **Custom Expressions**: `WithExpression()`

## Backends

### Algolia

The Algolia backend provides full-featured search capabilities:

```go
import "github.com/letmevibethatforyou/searchx/algolia"

// Create client with credentials from AWS Secrets Manager
client, err := algolia.NewClientFromAWS(ctx, "your-secret-name")
if err != nil {
    log.Fatal(err)
}

searcher := algolia.NewSearcher(client, "your-index")
```

## AWS Integration

### Lambda Functions

The repository includes AWS Lambda function templates in the `functions/` directory for triggering Algolia indexing operations.

### CloudFormation Deployment

Deploy SearchX infrastructure using the included CloudFormation template:

```bash
./scripts/deploy.sh
```

The template creates:
- DynamoDB table for data storage
- Lambda functions for search operations
- API Gateway endpoints
- Required IAM roles and policies

## Data Generation

Use the included data generator to populate test data:

```bash
cd cmd/generator
go build
./generator --env production --table-name vehicles --count 100
```

This generates random vehicle data and inserts it into DynamoDB.

## Error Handling

SearchX provides structured error handling with specific error codes:

```go
if err != nil {
    if errors.Is(err, searchx.ErrEmptyQuery) {
        // Handle empty query
    } else if errors.Is(err, searchx.ErrTimeout) {
        // Handle timeout
    }
    // Handle other errors
}
```

Available error codes:
- `ErrEmptyQuery`: Empty search query provided
- `ErrInvalidOption`: Invalid search option
- `ErrInvalidExpression`: Invalid filter expression
- `ErrTimeout`: Search operation timed out
- `ErrCanceled`: Search operation was canceled
- `ErrNotImplemented`: Feature not implemented
- `ErrBackendUnavailable`: Search backend unavailable

## Development

### Prerequisites

- Go 1.24+
- AWS CLI configured (for AWS integrations)
- Algolia account (for Algolia backend)

### Running Tests

```bash
go test ./...
```

### Project Structure

```
.
├── algolia/           # Algolia backend implementation
├── cmd/generator/     # Data generation utility
├── functions/         # AWS Lambda functions
├── inmemory/          # In-memory backend (for testing)
├── internal/          # Internal packages
├── scripts/           # Deployment scripts
├── expression.go      # Filter expression parsing
├── options.go         # Search options
├── results.go         # Search result types
├── searcher.go        # Core searcher interface
└── types.go           # Core types and errors
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License.