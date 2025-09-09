package ddb

import (
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DynamoDBEvent represents a DynamoDB stream event
type DynamoDBEvent struct {
	Records []DynamoDBEventRecord `json:"Records"`
}

// DynamoDBEventRecord represents a single DynamoDB stream record
type DynamoDBEventRecord struct {
	AWSRegion      string               `json:"awsRegion"`
	Change         DynamoDBStreamRecord `json:"dynamodb"`
	EventID        string               `json:"eventID"`
	EventName      string               `json:"eventName"`
	EventSource    string               `json:"eventSource"`
	EventVersion   string               `json:"eventVersion"`
	EventSourceArn string               `json:"eventSourceARN"`
}

// DynamoDBStreamRecord represents the DynamoDB stream data
type DynamoDBStreamRecord struct {
	ApproximateCreationDateTime int64                           `json:"ApproximateCreationDateTime,omitempty"`
	Keys                        map[string]types.AttributeValue `json:"Keys,omitempty"`
	NewImage                    map[string]types.AttributeValue `json:"NewImage,omitempty"`
	OldImage                    map[string]types.AttributeValue `json:"OldImage,omitempty"`
	SequenceNumber              string                          `json:"SequenceNumber"`
	SizeBytes                   int64                           `json:"SizeBytes"`
	StreamViewType              string                          `json:"StreamViewType"`
}

// DynamoDBOperationType represents the type of DynamoDB operation
type DynamoDBOperationType string

const (
	DynamoDBOperationTypeInsert DynamoDBOperationType = "INSERT"
	DynamoDBOperationTypeModify DynamoDBOperationType = "MODIFY"
	DynamoDBOperationTypeRemove DynamoDBOperationType = "REMOVE"
)

// Record represents a processed DynamoDB record with extracted fields
type Record struct {
	ID        string         `dynamodbav:"pk"`     // PK field
	IndexName string         `dynamodbav:"sk"`     // SK field
	Object    map[string]any `dynamodbav:"object"` // object field
}

// UnmarshalRecord converts a DynamoDB NewImage into a Record struct
func UnmarshalRecord(newImage map[string]types.AttributeValue) (Record, error) {
	var record Record
	err := attributevalue.UnmarshalMap(newImage, &record)
	if err != nil {
		return Record{}, err
	}
	return record, nil
}
