# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Build and Test
- `go build ./...` - Build all packages in the project
- `go test ./...` - Run all tests across all packages
- `go test -v ./inmemory` - Run tests for the inmemory package with verbose output
- `go test -v ./algolia` - Run tests for the algolia package with verbose output
- `go mod tidy` - Clean up dependencies and update go.mod/go.sum

### Command Line Tools
- `go run cmd/query/main.go` - Run the Algolia query CLI tool
- `go run cmd/generator/main.go` - Run the vehicle data generator for DynamoDB

### AWS Lambda Function
- `go run functions/trigger-algolia/main.go` - Run the DynamoDB to Algolia sync Lambda function

## Architecture Overview

This is a Go-based search abstraction library (`searchx`) that provides a unified interface for different search backends. The project implements a pluggable search architecture with multiple implementations.

### Core Components

**Core Search Interface** (`searcher.go`, `types.go`):
- `Searcher` interface defines the unified search API
- `SearchOption` pattern for configurable search parameters
- Comprehensive error handling with specific error codes
- Support for filtering, sorting, pagination, and full-text search

**Search Implementations**:
1. **In-Memory Searcher** (`inmemory/`): Thread-safe in-memory search implementation with full-text scoring
2. **Algolia Searcher** (`algolia/`): Algolia cloud search integration with AWS Secrets Manager support

**Expression System** (`expression.go`, `inmemory/expressions.go`):
- Rich query DSL with operators: `Eq`, `Ne`, `Gt`, `Gte`, `Lt`, `Lte`, `Range`, `Exists`
- Boolean operators: `And`, `Or`, `Not`
- Type-safe expression evaluation for filtering

**AWS Integration**:
- **AWS Secrets Manager**: Secure credential management for Algolia API keys
- **DynamoDB Integration**: Stream-based sync from DynamoDB to Algolia via Lambda
- **Lambda Functions**: Event-driven architecture for real-time search index updates

### Key Patterns

**Options Pattern**: Search configurations use functional options (`WithLimit`, `WithSort`, `WithOffset`)

**Expression Builder**: Type-safe query construction using expression builders rather than string-based queries

**Backend Abstraction**: All search implementations satisfy the same `Searcher` interface, enabling easy backend switching

**Error Handling**: Uses `cockroachdb/errors` for enhanced error context and categorization

**Tracing**: OpenTelemetry integration for observability

## Package Structure

- **Root**: Core interfaces and types
- **`inmemory/`**: In-memory search implementation with comprehensive test coverage
- **`algolia/`**: Algolia search client with AWS integration
- **`cmd/query/`**: CLI tool for querying Algolia indices with complex filters
- **`cmd/generator/`**: DynamoDB data generator for testing
- **`functions/trigger-algolia/`**: Lambda function for DynamoDB-to-Algolia sync
- **`internal/ddb/`**: Internal DynamoDB event handling utilities

## Testing Strategy

The project has comprehensive test coverage across all packages:
- Unit tests for expression evaluation and type conversion
- Integration tests for search functionality
- Concurrent operation tests
- Unicode and internationalization tests
- Context cancellation and timeout handling

Run tests frequently during development as they validate core search behavior and expression evaluation logic.

## AWS Dependencies

When working with AWS features:
- Ensure AWS credentials are configured (via AWS CLI, IAM roles, or environment variables)
- DynamoDB table structure: `pk` (partition key), `sk` (sort key), `object` (document data)
- Secrets Manager stores Algolia credentials in JSON format: `{"algolia_app_id": "...", "algolia_api_key": "..."}`

## Development Notes

- The project uses Go 1.24+ with specific toolchain requirements
- OpenTelemetry tracing is integrated throughout the Algolia client
- Error codes are centralized in `types.go` for consistent error handling
- Thread safety is critical - the inmemory searcher uses RWMutex for concurrent access
- Expression evaluation supports type coercion (int/float) for numeric comparisons