package ddb

import (
	"encoding/json"
	"fmt"

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
	Keys           map[string]types.AttributeValue `json:"Keys,omitempty"`
	NewImage       map[string]types.AttributeValue `json:"NewImage,omitempty"`
	OldImage       map[string]types.AttributeValue `json:"OldImage,omitempty"`
	SequenceNumber string                          `json:"SequenceNumber"`
	SizeBytes      int64                           `json:"SizeBytes"`
	StreamViewType string                          `json:"StreamViewType"`
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

// UnmarshalJSON implements custom JSON unmarshaling for DynamoDBStreamRecord
func (r *DynamoDBStreamRecord) UnmarshalJSON(data []byte) error {
	// First unmarshal into a raw structure to handle the basic fields
	type rawRecord struct {
		Keys           json.RawMessage `json:"Keys,omitempty"`
		NewImage       json.RawMessage `json:"NewImage,omitempty"`
		OldImage       json.RawMessage `json:"OldImage,omitempty"`
		SequenceNumber string          `json:"SequenceNumber"`
		SizeBytes      int64           `json:"SizeBytes"`
		StreamViewType string          `json:"StreamViewType"`
	}

	var raw rawRecord
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Set the simple fields
	r.SequenceNumber = raw.SequenceNumber
	r.SizeBytes = raw.SizeBytes
	r.StreamViewType = raw.StreamViewType

	// Convert Keys if present
	if len(raw.Keys) > 0 {
		keys, err := UnmarshalAttributeValueMap(raw.Keys)
		if err != nil {
			return fmt.Errorf("failed to unmarshal Keys: %w", err)
		}
		r.Keys = keys
	}

	// Convert NewImage if present
	if len(raw.NewImage) > 0 {
		newImage, err := UnmarshalAttributeValueMap(raw.NewImage)
		if err != nil {
			return fmt.Errorf("failed to unmarshal NewImage: %w", err)
		}
		r.NewImage = newImage
	}

	// Convert OldImage if present
	if len(raw.OldImage) > 0 {
		oldImage, err := UnmarshalAttributeValueMap(raw.OldImage)
		if err != nil {
			return fmt.Errorf("failed to unmarshal OldImage: %w", err)
		}
		r.OldImage = oldImage
	}

	return nil
}

// UnmarshalAttributeValueMap converts DynamoDB JSON format to map[string]types.AttributeValue
func UnmarshalAttributeValueMap(data []byte) (map[string]types.AttributeValue, error) {
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return nil, err
	}

	result := make(map[string]types.AttributeValue)
	for key, value := range rawMap {
		av, err := unmarshalAttributeValue(value)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal attribute %s: %w", key, err)
		}
		result[key] = av
	}

	return result, nil
}

// unmarshalAttributeValue converts a single DynamoDB JSON value to types.AttributeValue
func unmarshalAttributeValue(data []byte) (types.AttributeValue, error) {
	var rawValue map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawValue); err != nil {
		return nil, err
	}

	// Check for each AttributeValue type
	if sValue, exists := rawValue["S"]; exists {
		var str string
		if err := json.Unmarshal(sValue, &str); err != nil {
			return nil, err
		}
		return &types.AttributeValueMemberS{Value: str}, nil
	}

	if nValue, exists := rawValue["N"]; exists {
		var num string
		if err := json.Unmarshal(nValue, &num); err != nil {
			return nil, err
		}
		return &types.AttributeValueMemberN{Value: num}, nil
	}

	if bValue, exists := rawValue["BOOL"]; exists {
		var b bool
		if err := json.Unmarshal(bValue, &b); err != nil {
			return nil, err
		}
		return &types.AttributeValueMemberBOOL{Value: b}, nil
	}

	if mValue, exists := rawValue["M"]; exists {
		subMap, err := UnmarshalAttributeValueMap(mValue)
		if err != nil {
			return nil, err
		}
		return &types.AttributeValueMemberM{Value: subMap}, nil
	}

	if lValue, exists := rawValue["L"]; exists {
		var rawList []json.RawMessage
		if err := json.Unmarshal(lValue, &rawList); err != nil {
			return nil, err
		}

		var attributeList []types.AttributeValue
		for _, item := range rawList {
			av, err := unmarshalAttributeValue(item)
			if err != nil {
				return nil, err
			}
			attributeList = append(attributeList, av)
		}
		return &types.AttributeValueMemberL{Value: attributeList}, nil
	}

	if bsValue, exists := rawValue["BS"]; exists {
		var binarySet []string
		if err := json.Unmarshal(bsValue, &binarySet); err != nil {
			return nil, err
		}
		var binaryValues [][]byte
		for _, b := range binarySet {
			binaryValues = append(binaryValues, []byte(b))
		}
		return &types.AttributeValueMemberBS{Value: binaryValues}, nil
	}

	if nsValue, exists := rawValue["NS"]; exists {
		var numSet []string
		if err := json.Unmarshal(nsValue, &numSet); err != nil {
			return nil, err
		}
		return &types.AttributeValueMemberNS{Value: numSet}, nil
	}

	if ssValue, exists := rawValue["SS"]; exists {
		var strSet []string
		if err := json.Unmarshal(ssValue, &strSet); err != nil {
			return nil, err
		}
		return &types.AttributeValueMemberSS{Value: strSet}, nil
	}

	if nullValue, exists := rawValue["NULL"]; exists {
		var isNull bool
		if err := json.Unmarshal(nullValue, &isNull); err != nil {
			return nil, err
		}
		if isNull {
			return &types.AttributeValueMemberNULL{Value: true}, nil
		}
	}

	return nil, fmt.Errorf("unknown AttributeValue type in JSON: %s", string(data))
}
