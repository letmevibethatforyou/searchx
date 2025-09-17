package ddb

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func TestDynamoDBStreamRecord_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name               string
		jsonData           string
		expectedSeqNum     string
		expectedSizeBytes  int64
		expectedStreamType string
		hasKeys            bool
		hasNewImage        bool
		hasOldImage        bool
		wantErr            bool
	}{
		{
			name: "complete stream record with all fields",
			jsonData: `{
				"Keys": {
					"pk": {"S": "user#123"},
					"sk": {"S": "profile"}
				},
				"NewImage": {
					"pk": {"S": "user#123"},
					"sk": {"S": "profile"},
					"object": {
						"M": {
							"name": {"S": "John Doe"},
							"email": {"S": "john@example.com"},
							"age": {"N": "30"}
						}
					}
				},
				"OldImage": {
					"pk": {"S": "user#123"},
					"sk": {"S": "profile"},
					"object": {
						"M": {
							"name": {"S": "John Smith"},
							"email": {"S": "john.smith@example.com"},
							"age": {"N": "29"}
						}
					}
				},
				"SequenceNumber": "123456789",
				"SizeBytes": 1024,
				"StreamViewType": "NEW_AND_OLD_IMAGES"
			}`,
			expectedSeqNum:     "123456789",
			expectedSizeBytes:  1024,
			expectedStreamType: "NEW_AND_OLD_IMAGES",
			hasKeys:            true,
			hasNewImage:        true,
			hasOldImage:        true,
			wantErr:            false,
		},
		{
			name: "insert operation with only NewImage",
			jsonData: `{
				"Keys": {
					"pk": {"S": "item#456"},
					"sk": {"S": "metadata"}
				},
				"NewImage": {
					"pk": {"S": "item#456"},
					"sk": {"S": "metadata"},
					"object": {
						"M": {
							"title": {"S": "New Item"},
							"price": {"N": "99.99"},
							"available": {"BOOL": true}
						}
					}
				},
				"SequenceNumber": "987654321",
				"SizeBytes": 512,
				"StreamViewType": "NEW_AND_OLD_IMAGES"
			}`,
			expectedSeqNum:     "987654321",
			expectedSizeBytes:  512,
			expectedStreamType: "NEW_AND_OLD_IMAGES",
			hasKeys:            true,
			hasNewImage:        true,
			hasOldImage:        false,
			wantErr:            false,
		},
		{
			name: "remove operation with only OldImage",
			jsonData: `{
				"Keys": {
					"pk": {"S": "deleted#789"}
				},
				"OldImage": {
					"pk": {"S": "deleted#789"},
					"object": {
						"M": {
							"status": {"S": "deleted"}
						}
					}
				},
				"SequenceNumber": "555666777",
				"SizeBytes": 256,
				"StreamViewType": "OLD_IMAGE"
			}`,
			expectedSeqNum:     "555666777",
			expectedSizeBytes:  256,
			expectedStreamType: "OLD_IMAGE",
			hasKeys:            true,
			hasNewImage:        false,
			hasOldImage:        true,
			wantErr:            false,
		},
		{
			name: "minimal record with only required fields",
			jsonData: `{
				"SequenceNumber": "000111222",
				"SizeBytes": 100,
				"StreamViewType": "KEYS_ONLY"
			}`,
			expectedSeqNum:     "000111222",
			expectedSizeBytes:  100,
			expectedStreamType: "KEYS_ONLY",
			hasKeys:            false,
			hasNewImage:        false,
			hasOldImage:        false,
			wantErr:            false,
		},
		{
			name: "record with complex nested object",
			jsonData: `{
				"Keys": {
					"pk": {"S": "complex#001"}
				},
				"NewImage": {
					"pk": {"S": "complex#001"},
					"object": {
						"M": {
							"metadata": {
								"M": {
									"tags": {
										"L": [
											{"S": "tag1"},
											{"S": "tag2"}
										]
									},
									"scores": {
										"L": [
											{"N": "95.5"},
											{"N": "87.2"}
										]
									}
								}
							},
							"config": {
								"M": {
									"enabled": {"BOOL": true},
									"timeout": {"N": "30"}
								}
							}
						}
					}
				},
				"SequenceNumber": "111222333",
				"SizeBytes": 2048,
				"StreamViewType": "NEW_IMAGE"
			}`,
			expectedSeqNum:     "111222333",
			expectedSizeBytes:  2048,
			expectedStreamType: "NEW_IMAGE",
			hasKeys:            true,
			hasNewImage:        true,
			hasOldImage:        false,
			wantErr:            false,
		},
		{
			name:     "invalid JSON should fail",
			jsonData: `{"invalid": json}`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var record DynamoDBStreamRecord
			err := json.Unmarshal([]byte(tt.jsonData), &record)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify basic fields
			if record.SequenceNumber != tt.expectedSeqNum {
				t.Errorf("SequenceNumber mismatch: got %s, want %s", record.SequenceNumber, tt.expectedSeqNum)
			}

			if record.SizeBytes != tt.expectedSizeBytes {
				t.Errorf("SizeBytes mismatch: got %d, want %d", record.SizeBytes, tt.expectedSizeBytes)
			}

			if record.StreamViewType != tt.expectedStreamType {
				t.Errorf("StreamViewType mismatch: got %s, want %s", record.StreamViewType, tt.expectedStreamType)
			}

			// Verify presence/absence of Keys, NewImage, OldImage
			if tt.hasKeys && record.Keys == nil {
				t.Error("Expected Keys to be present but got nil")
			}
			if !tt.hasKeys && record.Keys != nil {
				t.Error("Expected Keys to be nil but got data")
			}

			if tt.hasNewImage && record.NewImage == nil {
				t.Error("Expected NewImage to be present but got nil")
			}
			if !tt.hasNewImage && record.NewImage != nil {
				t.Error("Expected NewImage to be nil but got data")
			}

			if tt.hasOldImage && record.OldImage == nil {
				t.Error("Expected OldImage to be present but got nil")
			}
			if !tt.hasOldImage && record.OldImage != nil {
				t.Error("Expected OldImage to be nil but got data")
			}

			// Verify AttributeValue types are properly unmarshaled
			if tt.hasKeys {
				verifyAttributeValueMap(t, record.Keys, "Keys")
			}
			if tt.hasNewImage {
				verifyAttributeValueMap(t, record.NewImage, "NewImage")
			}
			if tt.hasOldImage {
				verifyAttributeValueMap(t, record.OldImage, "OldImage")
			}
		})
	}
}

func TestDynamoDBEventRecord_UnmarshalJSON(t *testing.T) {
	jsonData := `{
		"awsRegion": "us-east-1",
		"eventID": "test-event-123",
		"eventName": "INSERT",
		"eventSource": "aws:dynamodb",
		"eventVersion": "1.1",
		"eventSourceARN": "arn:aws:dynamodb:us-east-1:123456789:table/TestTable/stream/2023-01-01T00:00:00.000",
		"dynamodb": {
			"Keys": {
				"pk": {"S": "test#123"}
			},
			"NewImage": {
				"pk": {"S": "test#123"},
				"object": {
					"M": {
						"name": {"S": "Test Item"}
					}
				}
			},
			"SequenceNumber": "123456789",
			"SizeBytes": 512,
			"StreamViewType": "NEW_AND_OLD_IMAGES"
		}
	}`

	var eventRecord DynamoDBEventRecord
	err := json.Unmarshal([]byte(jsonData), &eventRecord)
	if err != nil {
		t.Fatalf("Failed to unmarshal DynamoDBEventRecord: %v", err)
	}

	if eventRecord.AWSRegion != "us-east-1" {
		t.Errorf("AWSRegion mismatch: got %s, want us-east-1", eventRecord.AWSRegion)
	}

	if eventRecord.EventID != "test-event-123" {
		t.Errorf("EventID mismatch: got %s, want test-event-123", eventRecord.EventID)
	}

	if eventRecord.EventName != "INSERT" {
		t.Errorf("EventName mismatch: got %s, want INSERT", eventRecord.EventName)
	}

	if eventRecord.Change.SequenceNumber != "123456789" {
		t.Errorf("DynamoDB SequenceNumber mismatch: got %s, want 123456789", eventRecord.Change.SequenceNumber)
	}

	// Verify that the nested DynamoDB record was properly unmarshaled
	if eventRecord.Change.Keys == nil {
		t.Error("Expected Keys to be present")
	}

	if eventRecord.Change.NewImage == nil {
		t.Error("Expected NewImage to be present")
	}

	verifyAttributeValueMap(t, eventRecord.Change.Keys, "Keys")
	verifyAttributeValueMap(t, eventRecord.Change.NewImage, "NewImage")
}

func TestDynamoDBEvent_UnmarshalJSON(t *testing.T) {
	jsonData := `{
		"Records": [
			{
				"awsRegion": "us-east-1",
				"eventID": "event-1",
				"eventName": "INSERT",
				"eventSource": "aws:dynamodb",
				"eventVersion": "1.1",
				"eventSourceARN": "arn:aws:dynamodb:us-east-1:123456789:table/TestTable/stream/2023-01-01T00:00:00.000",
				"dynamodb": {
					"SequenceNumber": "111",
					"SizeBytes": 100,
					"StreamViewType": "NEW_IMAGE"
				}
			},
			{
				"awsRegion": "us-east-1",
				"eventID": "event-2",
				"eventName": "MODIFY",
				"eventSource": "aws:dynamodb",
				"eventVersion": "1.1",
				"eventSourceARN": "arn:aws:dynamodb:us-east-1:123456789:table/TestTable/stream/2023-01-01T00:00:00.000",
				"dynamodb": {
					"SequenceNumber": "222",
					"SizeBytes": 200,
					"StreamViewType": "NEW_AND_OLD_IMAGES"
				}
			}
		]
	}`

	var event DynamoDBEvent
	err := json.Unmarshal([]byte(jsonData), &event)
	if err != nil {
		t.Fatalf("Failed to unmarshal DynamoDBEvent: %v", err)
	}

	if len(event.Records) != 2 {
		t.Errorf("Expected 2 records, got %d", len(event.Records))
	}

	if event.Records[0].EventName != "INSERT" {
		t.Errorf("First record EventName mismatch: got %s, want INSERT", event.Records[0].EventName)
	}

	if event.Records[1].EventName != "MODIFY" {
		t.Errorf("Second record EventName mismatch: got %s, want MODIFY", event.Records[1].EventName)
	}

	if event.Records[0].Change.SequenceNumber != "111" {
		t.Errorf("First record SequenceNumber mismatch: got %s, want 111", event.Records[0].Change.SequenceNumber)
	}

	if event.Records[1].Change.SequenceNumber != "222" {
		t.Errorf("Second record SequenceNumber mismatch: got %s, want 222", event.Records[1].Change.SequenceNumber)
	}
}

func TestUnmarshalRecord_WithDynamoDBStreamRecord(t *testing.T) {
	// Test the existing UnmarshalRecord function with properly unmarshaled stream record
	jsonData := `{
		"pk": {"S": "user#789"},
		"sk": {"S": "profile"},
		"object": {
			"M": {
				"name": {"S": "Jane Smith"},
				"email": {"S": "jane@example.com"},
				"age": {"N": "25"},
				"active": {"BOOL": true}
			}
		}
	}`

	// Use our custom unmarshaling function to convert DynamoDB JSON to AttributeValue map
	newImageMap, err := UnmarshalAttributeValueMap([]byte(jsonData))
	if err != nil {
		t.Fatalf("Failed to unmarshal NewImage: %v", err)
	}

	// Test UnmarshalRecord function
	record, err := UnmarshalRecord(newImageMap)
	if err != nil {
		t.Fatalf("UnmarshalRecord failed: %v", err)
	}

	if record.ID != "user#789" {
		t.Errorf("ID mismatch: got %s, want user#789", record.ID)
	}

	if record.IndexName != "profile" {
		t.Errorf("IndexName mismatch: got %s, want profile", record.IndexName)
	}

	if record.Object == nil {
		t.Error("Object should not be nil")
	}

	if name, ok := record.Object["name"]; !ok || name != "Jane Smith" {
		t.Errorf("Object.name mismatch: got %v, want Jane Smith", name)
	}

	if email, ok := record.Object["email"]; !ok || email != "jane@example.com" {
		t.Errorf("Object.email mismatch: got %v, want jane@example.com", email)
	}

	if age, ok := record.Object["age"]; !ok || age != float64(25) {
		t.Errorf("Object.age mismatch: got %v, want 25", age)
	}

	if active, ok := record.Object["active"]; !ok || active != true {
		t.Errorf("Object.active mismatch: got %v, want true", active)
	}
}

func TestUnmarshalRecord_WithRawMessage(t *testing.T) {
	// Test with json.RawMessage to demonstrate the complete workflow
	rawData := json.RawMessage(`{
		"pk": {"S": "product#123"},
		"sk": {"S": "details"},
		"object": {
			"M": {
				"title": {"S": "Amazing Product"},
				"price": {"N": "29.99"},
				"inStock": {"BOOL": false},
				"tags": {
					"L": [
						{"S": "electronics"},
						{"S": "gadget"}
					]
				}
			}
		}
	}`)

	// Convert RawMessage to AttributeValue map using our custom function
	newImageMap, err := UnmarshalAttributeValueMap(rawData)
	if err != nil {
		t.Fatalf("Failed to unmarshal RawMessage: %v", err)
	}

	// Test UnmarshalRecord function
	record, err := UnmarshalRecord(newImageMap)
	if err != nil {
		t.Fatalf("UnmarshalRecord failed: %v", err)
	}

	if record.ID != "product#123" {
		t.Errorf("ID mismatch: got %s, want product#123", record.ID)
	}

	if record.IndexName != "details" {
		t.Errorf("IndexName mismatch: got %s, want details", record.IndexName)
	}

	if record.Object == nil {
		t.Error("Object should not be nil")
	}

	if title, ok := record.Object["title"]; !ok || title != "Amazing Product" {
		t.Errorf("Object.title mismatch: got %v, want Amazing Product", title)
	}

	if price, ok := record.Object["price"]; !ok || price != float64(29.99) {
		t.Errorf("Object.price mismatch: got %v, want 29.99", price)
	}

	if inStock, ok := record.Object["inStock"]; !ok || inStock != false {
		t.Errorf("Object.inStock mismatch: got %v, want false", inStock)
	}

	// Test list/array handling
	if tags, ok := record.Object["tags"]; !ok {
		t.Error("Object.tags should be present")
	} else if tagList, ok := tags.([]interface{}); !ok {
		t.Errorf("Object.tags should be a slice, got %T", tags)
	} else if len(tagList) != 2 {
		t.Errorf("Object.tags should have 2 elements, got %d", len(tagList))
	} else {
		if tagList[0] != "electronics" {
			t.Errorf("First tag mismatch: got %v, want electronics", tagList[0])
		}
		if tagList[1] != "gadget" {
			t.Errorf("Second tag mismatch: got %v, want gadget", tagList[1])
		}
	}
}

// verifyAttributeValueMap checks that an AttributeValue map contains proper types
func verifyAttributeValueMap(t *testing.T, m map[string]types.AttributeValue, fieldName string) {
	if m == nil {
		t.Errorf("%s should not be nil", fieldName)
		return
	}

	for key, value := range m {
		if value == nil {
			t.Errorf("%s[%s] should not be nil", fieldName, key)
			continue
		}

		// Verify that we have proper AttributeValue types
		switch v := value.(type) {
		case *types.AttributeValueMemberS:
			if v.Value == "" {
				t.Errorf("%s[%s] string value should not be empty", fieldName, key)
			}
		case *types.AttributeValueMemberN:
			if v.Value == "" {
				t.Errorf("%s[%s] number value should not be empty", fieldName, key)
			}
		case *types.AttributeValueMemberBOOL:
			// Boolean values are fine as-is
		case *types.AttributeValueMemberM:
			// Recursively verify nested maps
			verifyAttributeValueMap(t, v.Value, fieldName+"."+key)
		case *types.AttributeValueMemberL:
			if len(v.Value) == 0 {
				t.Errorf("%s[%s] list should not be empty", fieldName, key)
			}
		default:
			t.Errorf("%s[%s] has unexpected AttributeValue type: %T", fieldName, key, value)
		}
	}
}
