package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/letmevibethatforyou/searchx"
	"github.com/letmevibethatforyou/searchx/algolia"
	"github.com/urfave/cli/v2"
)

const (
	defaultLimit   = 10
	defaultTimeout = 5 * time.Second
)

func main() {
	if os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" || os.Getenv("AWS_REGION") != "" {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	}

	app := &cli.App{
		Name:  "query",
		Usage: "Execute search queries against an Algolia index",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "index",
				Aliases:  []string{"i"},
				Usage:    "Algolia index name",
				EnvVars:  []string{"ALGOLIA_INDEX"},
				Required: true,
			},
			&cli.StringFlag{
				Name:    "algolia-secret-arn",
				Usage:   "ARN of AWS Secrets Manager secret containing Algolia credentials",
				EnvVars: []string{"ALGOLIA_SECRET_ARN"},
			},
			&cli.StringFlag{
				Name:    "query",
				Aliases: []string{"q"},
				Usage:   "Query string to search for; positional arg is a fallback",
			},
			&cli.IntFlag{
				Name:    "limit",
				Aliases: []string{"l"},
				Usage:   "Maximum number of results to return",
				Value:   defaultLimit,
			},
			&cli.IntFlag{
				Name:    "offset",
				Aliases: []string{"o"},
				Usage:   "Number of results to skip before returning hits",
				Value:   0,
			},
			&cli.DurationFlag{
				Name:  "timeout",
				Usage: "Timeout for the search request",
				Value: defaultTimeout,
			},
			&cli.StringSliceFlag{
				Name:  "filter",
				Usage: "Filter in field=value format; repeatable",
			},
		},
		Action: runAction,
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("Application failed", "error", err)
		os.Exit(1)
	}
}

func runAction(c *cli.Context) error {
	ctx := c.Context

	query := strings.TrimSpace(c.String("query"))
	if query == "" && c.NArg() > 0 {
		query = strings.TrimSpace(c.Args().First())
	}

	indexName := strings.TrimSpace(c.String("index"))

	limit := c.Int("limit")
	if limit <= 0 {
		slog.WarnContext(ctx, "limit must be positive; falling back to default", "limit", limit, "default", defaultLimit)
		limit = defaultLimit
	}

	offset := c.Int("offset")
	if offset < 0 {
		slog.WarnContext(ctx, "offset cannot be negative; resetting to 0", "offset", offset)
		offset = 0
	}

	timeout := c.Duration("timeout")
	if timeout <= 0 {
		slog.WarnContext(ctx, "timeout must be positive; using default", "timeout", timeout, "default", defaultTimeout)
		timeout = defaultTimeout
	}

	filterOptions, err := buildFilterOptions(c.StringSlice("filter"))
	if err != nil {
		return fmt.Errorf("invalid filter: %w", err)
	}

	secretArn := strings.TrimSpace(c.String("algolia-secret-arn"))

	var fetchSecrets algolia.FetchSecrets
	if secretArn != "" {
		slog.InfoContext(ctx, "using AWS Secrets Manager for Algolia credentials", "secret_arn", secretArn)
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return fmt.Errorf("failed to load AWS config: %w", err)
		}
		secretsClient := secretsmanager.NewFromConfig(cfg)
		fetchSecrets = algolia.AWSSecretsFromARN(ctx, secretsClient, secretArn)
	} else {
		fetchSecrets = algolia.EnvSecrets()
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := algolia.NewClient(fetchSecrets)
	searcher := algolia.NewSearcher(client, indexName)

	opts := []searchx.SearchOption{
		searchx.WithLimit(limit),
		searchx.WithOffset(offset),
	}
	opts = append(opts, filterOptions...)

	slog.InfoContext(ctx, "executing query",
		"index", indexName,
		"query", query,
		"limit", limit,
		"offset", offset,
		"filter_count", len(filterOptions),
		"timeout", timeout,
	)

	results, err := searcher.Search(ctx, query, opts...)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if err := printResults(results); err != nil {
		return fmt.Errorf("failed to serialize results: %w", err)
	}

	return nil
}

func buildFilterOptions(raw []string) ([]searchx.SearchOption, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	options := make([]searchx.SearchOption, 0, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item == "" {
			return nil, fmt.Errorf("filter cannot be empty")
		}

		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("filter must be in field=value format: %q", item)
		}

		field := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if field == "" || value == "" {
			return nil, fmt.Errorf("filter field and value must be non-empty: %q", item)
		}

		options = append(options, searchx.Eq(field, value))
	}

	return options, nil
}

func printResults(res *searchx.Results) error {
	if res == nil {
		fmt.Println("{}")
		return nil
	}

	payload := struct {
		Total      int64            `json:"total"`
		Took       int64            `json:"took_ms"`
		Query      string           `json:"query"`
		MaxScore   float64          `json:"max_score"`
		NextOffset *int             `json:"next_offset,omitempty"`
		Items      []searchx.Result `json:"items"`
	}{
		Total:      res.Total,
		Took:       res.Took,
		Query:      res.Query,
		MaxScore:   res.MaxScore,
		NextOffset: res.NextOffset,
		Items:      res.Items,
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	fmt.Println(string(data))
	return nil
}
