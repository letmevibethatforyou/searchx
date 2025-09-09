package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/letmevibethatforyou/searchx/internal/algoliaassearch"
	"github.com/letmevibethatforyou/searchx/internal/ddb"
	"github.com/urfave/cli/v2"
)

type Handler struct {
	tableName     string
	algoliaClient *algoliaassearch.Client
}

func NewHandler(tableName string, fetchSecrets algoliaassearch.FetchSecrets) *Handler {
	algoliaClient := algoliaassearch.NewClient(fetchSecrets)

	return &Handler{
		tableName:     tableName,
		algoliaClient: algoliaClient,
	}
}

func (h *Handler) HandleDynamoDBEvent(ctx context.Context, e ddb.DynamoDBEvent) error {
	slog.InfoContext(ctx, "Processing DynamoDB stream records", "record_count", len(e.Records))

	for _, record := range e.Records {
		if err := h.processRecord(ctx, record); err != nil {
			slog.ErrorContext(ctx, "Error processing record", "error", err)
			return err
		}
	}

	return nil
}

func (h *Handler) processRecord(ctx context.Context, record ddb.DynamoDBEventRecord) error {
	switch ddb.DynamoDBOperationType(record.EventName) {
	case ddb.DynamoDBOperationTypeInsert, ddb.DynamoDBOperationTypeModify:
		if record.Change.NewImage == nil {
			slog.WarnContext(ctx, "No new image for insert/modify operation, skipping record")
			return nil
		}

		// Unmarshal the NewImage into our custom Record type
		parsedRecord, err := ddb.UnmarshalRecord(record.Change.NewImage)
		if err != nil {
			slog.WarnContext(ctx, "Failed to unmarshal record, skipping", "error", err)
			return nil
		}

		// Validate required fields
		if parsedRecord.ID == "" {
			slog.WarnContext(ctx, "Missing ID (pk) in record, skipping record")
			return nil
		}
		if parsedRecord.IndexName == "" {
			slog.WarnContext(ctx, "Missing IndexName (sk) in record, skipping record")
			return nil
		}
		if parsedRecord.Object == nil {
			slog.WarnContext(ctx, "Missing Object in record, skipping record", "id", parsedRecord.ID, "index", parsedRecord.IndexName)
			return nil
		}

		return h.handleUpsert(ctx, &parsedRecord)

	case ddb.DynamoDBOperationTypeRemove:
		// For delete operations, we only need the keys
		parsedRecord, err := ddb.UnmarshalRecord(record.Change.Keys)
		if err != nil {
			slog.WarnContext(ctx, "Failed to unmarshal keys for delete operation, skipping", "error", err)
			return nil
		}

		if parsedRecord.ID == "" || parsedRecord.IndexName == "" {
			slog.WarnContext(ctx, "Missing ID or IndexName in delete record, skipping record")
			return nil
		}

		return h.handleDelete(ctx, parsedRecord.IndexName, parsedRecord.ID)

	default:
		slog.InfoContext(ctx, "Ignoring event type", "event_type", record.EventName)
		return nil
	}
}

func (h *Handler) handleUpsert(ctx context.Context, record *ddb.Record) error {
	// Convert the object map to the format expected by Algolia
	algoliaObject := make(map[string]interface{})
	for k, v := range record.Object {
		algoliaObject[k] = v
	}

	// Set the Algolia objectID
	algoliaObject["objectID"] = record.ID

	slog.InfoContext(ctx, "Saving object to Algolia", "object_id", record.ID, "index", record.IndexName)
	return h.algoliaClient.SaveObject(ctx, record.IndexName, algoliaObject)
}

func (h *Handler) handleDelete(ctx context.Context, indexName, objectID string) error {
	slog.InfoContext(ctx, "Deleting object from Algolia", "object_id", objectID, "index", indexName)
	return h.algoliaClient.DeleteObject(ctx, indexName, objectID)
}

func main() {
	app := &cli.App{
		Name:  "dynamodb-algolia-sync",
		Usage: "Sync DynamoDB stream events to Algolia",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "table-name",
				Usage:    "DynamoDB table name to sync from",
				EnvVars:  []string{"TABLE_NAME"},
				Required: true,
			},
			&cli.StringFlag{
				Name:    "algolia-app-id",
				Usage:   "Algolia application ID",
				EnvVars: []string{"ALGOLIA_APP_ID"},
			},
			&cli.StringFlag{
				Name:    "algolia-api-key",
				Usage:   "Algolia API key",
				EnvVars: []string{"ALGOLIA_API_KEY"},
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
	tableName := c.String("table-name")
	algoliaAppID := c.String("algolia-app-id")
	algoliaAPIKey := c.String("algolia-api-key")

	slog.InfoContext(ctx, "Starting DynamoDB to Algolia sync", "table", tableName)

	var fetchSecrets algoliaassearch.FetchSecrets
	if algoliaAppID != "" && algoliaAPIKey != "" {
		fetchSecrets = algoliaassearch.StaticSecrets(algoliaAppID, algoliaAPIKey)
	} else {
		fetchSecrets = algoliaassearch.EnvSecrets()
	}

	handler := NewHandler(tableName, fetchSecrets)

	if os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" {
		slog.InfoContext(ctx, "Running in Lambda environment")
		lambda.Start(handler.HandleDynamoDBEvent)
	} else {
		slog.InfoContext(ctx, "Running in CLI mode")
		lambda.Start(handler.HandleDynamoDBEvent)
	}

	return nil
}
