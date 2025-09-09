package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/letmevibethatforyou/searchx"
	"github.com/letmevibethatforyou/searchx/algolia"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "query",
		Usage: "Query Algolia search index with various operators and filters",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "env",
				Aliases:  []string{"e"},
				Usage:    "Environment for AWS Secrets Manager (e.g., prod, staging, dev)",
				EnvVars:  []string{"ENV"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     "index",
				Aliases:  []string{"i"},
				Usage:    "Algolia index name to search",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "filters",
				Aliases: []string{"f"},
				Usage:   "Comma-separated filters with operators (e.g., 'price>100,category=electronics,rating>=4')",
			},
			&cli.StringFlag{
				Name:  "facets",
				Usage: "Comma-separated list of facets to retrieve",
			},
			&cli.BoolFlag{
				Name:  "facet-search",
				Usage: "Enable faceted search",
			},
			&cli.IntFlag{
				Name:    "page",
				Aliases: []string{"p"},
				Usage:   "Page number (0-based)",
				Value:   0,
			},
			&cli.IntFlag{
				Name:    "hits-per-page",
				Aliases: []string{"hpp"},
				Usage:   "Number of hits per page",
				Value:   20,
			},
			&cli.StringFlag{
				Name:    "sort-by",
				Aliases: []string{"s"},
				Usage:   "Sort by attribute (can include direction with :asc or :desc)",
			},
			&cli.BoolFlag{
				Name:  "pretty",
				Usage: "Pretty-print JSON output",
				Value: true,
			},
		},
		Action: runSearch,
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("Application failed", "error", err)
		os.Exit(1)
	}
}

func runSearch(c *cli.Context) error {
	ctx := c.Context
	env := c.String("env")
	indexName := c.String("index")

	// Build query from remaining arguments
	query := strings.Join(c.Args().Slice(), " ")

	slog.InfoContext(ctx, "Starting Algolia search",
		"environment", env,
		"index", indexName,
		"query", query,
	)

	// Setup AWS config and Secrets Manager client
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	secretsClient := secretsmanager.NewFromConfig(cfg)
	fetchSecrets := algolia.AWSSecrets(ctx, secretsClient, env)

	// Create Algolia client and get searcher
	algoliaClient := algolia.NewClient(fetchSecrets)
	searcher := algolia.NewSearcher(algoliaClient, indexName)

	// Build search options from CLI flags
	searchOptions, err := buildSearchOptions(c)
	if err != nil {
		return fmt.Errorf("failed to build search options: %w", err)
	}

	// Perform search using searcher
	results, err := searcher.Search(ctx, query, searchOptions...)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	// Output results
	if c.Bool("pretty") {
		prettyJSON, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format JSON: %w", err)
		}
		fmt.Println(string(prettyJSON))
	} else {
		jsonData, err := json.Marshal(results)
		if err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	}

	return nil
}

func buildSearchOptions(c *cli.Context) ([]searchx.SearchOption, error) {
	var options []searchx.SearchOption

	// Pagination
	if limit := c.Int("hits-per-page"); limit > 0 {
		options = append(options, searchx.WithLimit(limit))
	}

	if page := c.Int("page"); page > 0 {
		hitsPerPage := c.Int("hits-per-page")
		if hitsPerPage <= 0 {
			hitsPerPage = 20 // Default
		}
		offset := page * hitsPerPage
		options = append(options, searchx.WithOffset(offset))
	}

	// Sorting
	if sortBy := c.String("sort-by"); sortBy != "" {
		field := sortBy
		desc := false

		// Check for direction suffix
		if strings.HasSuffix(sortBy, ":desc") {
			field = strings.TrimSuffix(sortBy, ":desc")
			desc = true
		} else if strings.HasSuffix(sortBy, ":asc") {
			field = strings.TrimSuffix(sortBy, ":asc")
			desc = false
		}

		options = append(options, searchx.WithSort(field, desc))
	}

	// Filters
	if filters := c.String("filters"); filters != "" {
		expressions, err := parseFiltersToExpressions(filters)
		if err != nil {
			return nil, fmt.Errorf("failed to parse filters: %w", err)
		}

		// Add all expressions as options
		for _, expr := range expressions {
			options = append(options, expr)
		}
	}

	return options, nil
}

func parseFiltersToExpressions(filtersStr string) ([]searchx.Expression, error) {
	filterParts := strings.Split(filtersStr, ",")
	var expressions []searchx.Expression

	for _, filter := range filterParts {
		filter = strings.TrimSpace(filter)
		if filter == "" {
			continue
		}

		expr, err := parseFilterToExpression(filter)
		if err != nil {
			return nil, fmt.Errorf("invalid filter '%s': %w", filter, err)
		}
		expressions = append(expressions, expr)
	}

	return expressions, nil
}

func parseFilterToExpression(filter string) (searchx.Expression, error) {
	// Support operators: >=, <=, !=, >, <, =
	operators := []string{">=", "<=", "!=", ">", "<", "="}

	for _, op := range operators {
		if idx := strings.Index(filter, op); idx > 0 {
			attribute := strings.TrimSpace(filter[:idx])
			value := strings.TrimSpace(filter[idx+len(op):])

			if attribute == "" || value == "" {
				return nil, fmt.Errorf("attribute and value cannot be empty")
			}

			// Convert value to appropriate type
			var parsedValue interface{} = value
			if isNumeric(value) {
				if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
					parsedValue = floatVal
				}
			}

			// Handle different operators
			switch op {
			case "=":
				return searchx.Eq(attribute, parsedValue), nil
			case "!=":
				return searchx.Ne(attribute, parsedValue), nil
			case ">":
				return searchx.Gt(attribute, parsedValue), nil
			case ">=":
				return searchx.Gte(attribute, parsedValue), nil
			case "<":
				return searchx.Lt(attribute, parsedValue), nil
			case "<=":
				return searchx.Lte(attribute, parsedValue), nil
			}
		}
	}

	return nil, fmt.Errorf("no valid operator found")
}

func isNumeric(value string) bool {
	// Try to parse as float
	_, err := strconv.ParseFloat(value, 64)
	return err == nil
}
