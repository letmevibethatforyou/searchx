package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/segmentio/ksuid"
	"github.com/urfave/cli/v2"
)

type Vehicle struct {
	Make  string `json:"make"`
	Model string `json:"model"`
	Year  int    `json:"year"`
	Color string `json:"color"`
}

type DynamoDBRecord struct {
	PK     string  `dynamodbav:"pk"`
	SK     string  `dynamodbav:"sk"`
	Object Vehicle `dynamodbav:"object"`
}

var (
	makes = map[string][]string{
		"Toyota":    {"Camry", "Corolla", "Prius", "RAV4", "Highlander", "Tacoma", "4Runner"},
		"Honda":     {"Civic", "Accord", "CR-V", "Pilot", "Fit", "HR-V", "Ridgeline"},
		"Ford":      {"F-150", "Mustang", "Explorer", "Escape", "Focus", "Fusion", "Bronco"},
		"BMW":       {"3 Series", "5 Series", "X3", "X5", "i3", "i8", "Z4"},
		"Mercedes":  {"C-Class", "E-Class", "S-Class", "GLC", "GLE", "A-Class", "CLA"},
		"Audi":      {"A3", "A4", "A6", "Q3", "Q5", "Q7", "TT"},
		"Chevrolet": {"Silverado", "Equinox", "Malibu", "Tahoe", "Suburban", "Camaro", "Corvette"},
		"Nissan":    {"Altima", "Sentra", "Rogue", "Pathfinder", "Frontier", "Titan", "370Z"},
	}

	colors = []string{
		"Red", "Blue", "Black", "White", "Silver", "Gray", "Green", "Yellow", "Orange", "Purple",
	}
)

func generateRandomVehicle() Vehicle {
	makeKeys := make([]string, 0, len(makes))
	for v := range makes {
		makeKeys = append(makeKeys, v)
	}

	selectedMake := makeKeys[rand.IntN(len(makeKeys))]
	models := makes[selectedMake]
	selectedModel := models[rand.IntN(len(models))]
	selectedYear := rand.IntN(10) + 2015 // 2015-2024
	selectedColor := colors[rand.IntN(len(colors))]

	return Vehicle{
		Make:  selectedMake,
		Model: selectedModel,
		Year:  selectedYear,
		Color: selectedColor,
	}
}

func insertVehicle(ctx context.Context, client *dynamodb.Client, tableName string, vehicle Vehicle) error {
	id := ksuid.New().String()

	record := DynamoDBRecord{
		PK:     id,
		SK:     "cars",
		Object: vehicle,
	}

	item, err := attributevalue.MarshalMap(record)
	if err != nil {
		return fmt.Errorf("failed to marshal vehicle record: %w", err)
	}

	_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to put item in DynamoDB: %w", err)
	}

	slog.InfoContext(ctx, "Successfully inserted vehicle",
		"id", id,
		"make", vehicle.Make,
		"model", vehicle.Model,
		"year", vehicle.Year,
		"color", vehicle.Color,
	)

	return nil
}

func runAction(c *cli.Context) error {
	ctx := c.Context
	env := c.String("env")
	tableName := c.String("table-name")
	count := c.Int("count")

	slog.InfoContext(ctx, "Starting vehicle generator",
		"environment", env,
		"table", tableName,
		"count", count,
	)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := dynamodb.NewFromConfig(cfg)

	for i := 0; i < count; i++ {
		vehicle := generateRandomVehicle()
		if err := insertVehicle(ctx, client, tableName, vehicle); err != nil {
			return fmt.Errorf("failed to insert vehicle %d: %w", i+1, err)
		}
	}

	slog.InfoContext(ctx, "Successfully generated and inserted all vehicles", "count", count)
	return nil
}

func main() {
	// Configure JSON logging for AWS environments
	if os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" || os.Getenv("AWS_REGION") != "" {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	}

	app := &cli.App{
		Name:  "generator",
		Usage: "Generate random vehicle data and insert into DynamoDB",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "env",
				Aliases:  []string{"e"},
				Usage:    "Environment name",
				EnvVars:  []string{"ENVIRONMENT"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     "table-name",
				Aliases:  []string{"t"},
				Usage:    "DynamoDB table name",
				EnvVars:  []string{"TABLE_NAME"},
				Required: true,
			},
			&cli.IntFlag{
				Name:    "count",
				Aliases: []string{"c"},
				Usage:   "Number of vehicles to generate",
				Value:   1,
			},
		},
		Action: runAction,
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("Application failed", "error", err)
		os.Exit(1)
	}
}
