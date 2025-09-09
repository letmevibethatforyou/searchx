package algolia

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// mockSecretsManagerClient implements SecretsManagerClient for testing
type mockSecretsManagerClient struct {
	secretValue *string
	err         error
}

func (m *mockSecretsManagerClient) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &secretsmanager.GetSecretValueOutput{
		SecretString: m.secretValue,
	}, nil
}

func TestAWSSecrets_Success(t *testing.T) {
	ctx := context.Background()
	env := "production"
	secretJSON := `{"app_id":"test-app-id","write_api_key":"test-api-key"}`

	client := &mockSecretsManagerClient{
		secretValue: aws.String(secretJSON),
	}

	fetchSecrets := AWSSecrets(ctx, client, env)
	secrets, err := fetchSecrets()

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if secrets.AppID != "test-app-id" {
		t.Errorf("Expected AppID to be 'test-app-id', got '%s'", secrets.AppID)
	}

	if secrets.WriteApiKey != "test-api-key" {
		t.Errorf("Expected WriteApiKey to be 'test-api-key', got '%s'", secrets.WriteApiKey)
	}
}

func TestAWSSecrets_GetSecretError(t *testing.T) {
	ctx := context.Background()
	env := "production"

	client := &mockSecretsManagerClient{
		err: errors.New("secrets manager error"),
	}

	fetchSecrets := AWSSecrets(ctx, client, env)
	_, err := fetchSecrets()

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	expectedMsg := "failed to get secret from AWS Secrets Manager at path production/algolia"
	if !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestAWSSecrets_NilSecretString(t *testing.T) {
	ctx := context.Background()
	env := "production"

	client := &mockSecretsManagerClient{
		secretValue: nil,
	}

	fetchSecrets := AWSSecrets(ctx, client, env)
	_, err := fetchSecrets()

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	expectedMsg := "secret at path production/algolia has no string value"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestAWSSecrets_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	env := "production"
	invalidJSON := `{"app_id":"test-app-id","write_api_key":}`

	client := &mockSecretsManagerClient{
		secretValue: aws.String(invalidJSON),
	}

	fetchSecrets := AWSSecrets(ctx, client, env)
	_, err := fetchSecrets()

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	expectedMsg := "failed to unmarshal secret JSON from path production/algolia"
	if !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestAWSSecrets_EnvironmentPath(t *testing.T) {
	ctx := context.Background()
	env := "staging"
	secretJSON := `{"app_id":"staging-app-id","write_api_key":"staging-api-key"}`

	client := &mockSecretsManagerClient{
		secretValue: aws.String(secretJSON),
	}

	fetchSecrets := AWSSecrets(ctx, client, env)
	secrets, err := fetchSecrets()

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if secrets.AppID != "staging-app-id" {
		t.Errorf("Expected AppID to be 'staging-app-id', got '%s'", secrets.AppID)
	}

	if secrets.WriteApiKey != "staging-api-key" {
		t.Errorf("Expected WriteApiKey to be 'staging-api-key', got '%s'", secrets.WriteApiKey)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
