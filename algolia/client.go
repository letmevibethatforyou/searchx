// Package algoliaassearch provides a lazy-loading Algolia search client with configurable secret management.
package algolia

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Secrets holds the Algolia application credentials.
type Secrets struct {
	// AppID is the Algolia application ID.
	AppID string
	// WriteApiKey is the Algolia write API key.
	WriteApiKey string
}

// FetchSecrets is a function type that retrieves Algolia credentials.
// It allows for different secret retrieval strategies (static, environment variables, etc.).
type FetchSecrets func() (Secrets, error)

// StaticSecrets returns a FetchSecrets function that provides static credentials.
// This is useful for testing or when credentials are known at compile time.
func StaticSecrets(appID, writeApiKey string) FetchSecrets {
	return func() (Secrets, error) {
		return Secrets{
			AppID:       appID,
			WriteApiKey: writeApiKey,
		}, nil
	}
}

func EnvSecrets() FetchSecrets {
	return func() (Secrets, error) {
		appID := os.Getenv("ALGOLIA_APP_ID")
		if appID == "" {
			return Secrets{}, fmt.Errorf("ALGOLIA_APP_ID environment variable is not set")
		}

		apiKey := os.Getenv("ALGOLIA_API_KEY")
		if apiKey == "" {
			return Secrets{}, fmt.Errorf("ALGOLIA_API_KEY environment variable is not set")
		}

		return Secrets{
			AppID:       appID,
			WriteApiKey: apiKey,
		}, nil
	}
}

type Client struct {
	getClient func() (*search.Client, error)
	tracer    trace.Tracer
}

func NewClient(fetchSecrets FetchSecrets) *Client {
	getClient := sync.OnceValues(func() (*search.Client, error) {
		secrets, err := fetchSecrets()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch secrets: %w", err)
		}

		if secrets.AppID == "" {
			return nil, fmt.Errorf("AppID is empty")
		}

		if secrets.WriteApiKey == "" {
			return nil, fmt.Errorf("WriteApiKey is empty")
		}

		client := search.NewClient(secrets.AppID, secrets.WriteApiKey)
		return client, nil
	})

	tracer := otel.Tracer("searchx-algolia")

	return &Client{
		getClient: getClient,
		tracer:    tracer,
	}
}

func (c *Client) SaveObject(ctx context.Context, indexName string, object map[string]interface{}) error {
	ctx, span := c.tracer.Start(ctx, "algolia.save_object",
		trace.WithAttributes(
			attribute.String("algolia.index_name", indexName),
		),
	)
	defer span.End()

	objectID := ""
	if id, ok := object["objectID"].(string); ok {
		objectID = id
		span.SetAttributes(attribute.String("algolia.object_id", objectID))
	}

	client, err := c.getClient()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get Algolia client")
		return err
	}

	index := client.InitIndex(indexName)

	_, err = index.SaveObject(object)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, fmt.Sprintf("failed to save object to index %s", indexName))
		return fmt.Errorf("failed to save object to Algolia index %s: %w", indexName, err)
	}

	span.SetStatus(codes.Ok, "object saved successfully")
	return nil
}

func (c *Client) DeleteObject(ctx context.Context, indexName string, objectID string) error {
	ctx, span := c.tracer.Start(ctx, "algolia.delete_object",
		trace.WithAttributes(
			attribute.String("algolia.index_name", indexName),
			attribute.String("algolia.object_id", objectID),
		),
	)
	defer span.End()

	client, err := c.getClient()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get Algolia client")
		return err
	}

	index := client.InitIndex(indexName)

	_, err = index.DeleteObject(objectID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, fmt.Sprintf("failed to delete object from index %s", indexName))
		return fmt.Errorf("failed to delete object from Algolia index %s: %w", indexName, err)
	}

	span.SetStatus(codes.Ok, "object deleted successfully")
	return nil
}

func (c *Client) BatchSaveObjects(ctx context.Context, indexName string, objects []map[string]interface{}) error {
	if len(objects) == 0 {
		return nil
	}

	ctx, span := c.tracer.Start(ctx, "algolia.batch_save_objects",
		trace.WithAttributes(
			attribute.String("algolia.index_name", indexName),
			attribute.Int("algolia.object_count", len(objects)),
		),
	)
	defer span.End()

	client, err := c.getClient()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get Algolia client")
		return err
	}

	index := client.InitIndex(indexName)

	_, err = index.SaveObjects(objects)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, fmt.Sprintf("failed to batch save %d objects to index %s", len(objects), indexName))
		return fmt.Errorf("failed to batch save objects to Algolia index %s: %w", indexName, err)
	}

	span.SetStatus(codes.Ok, fmt.Sprintf("batch saved %d objects successfully", len(objects)))
	return nil
}

func (c *Client) BatchDeleteObjects(ctx context.Context, indexName string, objectIDs []string) error {
	if len(objectIDs) == 0 {
		return nil
	}

	ctx, span := c.tracer.Start(ctx, "algolia.batch_delete_objects",
		trace.WithAttributes(
			attribute.String("algolia.index_name", indexName),
			attribute.Int("algolia.object_count", len(objectIDs)),
		),
	)
	defer span.End()

	client, err := c.getClient()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get Algolia client")
		return err
	}

	index := client.InitIndex(indexName)

	_, err = index.DeleteObjects(objectIDs)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, fmt.Sprintf("failed to batch delete %d objects from index %s", len(objectIDs), indexName))
		return fmt.Errorf("failed to batch delete objects from Algolia index %s: %w", indexName, err)
	}

	span.SetStatus(codes.Ok, fmt.Sprintf("batch deleted %d objects successfully", len(objectIDs)))
	return nil
}
