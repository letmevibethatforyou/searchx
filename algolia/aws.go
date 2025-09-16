package algolia

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// SecretsManagerClient defines the interface for AWS Secrets Manager operations.
type SecretsManagerClient interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

// AWSSecrets returns a FetchSecrets function that retrieves Algolia credentials
// from AWS Secrets Manager. The secret is expected to be stored at the path
// "{environment}/algolia" and contain JSON with app_id and write_api_key fields.
func AWSSecrets(ctx context.Context, client SecretsManagerClient, env string) FetchSecrets {
	return func() (Secrets, error) {
		secretPath := fmt.Sprintf("%s/algolia", env)

		input := &secretsmanager.GetSecretValueInput{
			SecretId: aws.String(secretPath),
		}

		result, err := client.GetSecretValue(ctx, input)
		if err != nil {
			return Secrets{}, fmt.Errorf("failed to get secret from AWS Secrets Manager at path %s: %w", secretPath, err)
		}

		if result.SecretString == nil {
			return Secrets{}, fmt.Errorf("secret at path %s has no string value", secretPath)
		}

		var secrets Secrets
		if err := json.Unmarshal([]byte(aws.ToString(result.SecretString)), &secrets); err != nil {
			return Secrets{}, fmt.Errorf("failed to unmarshal secret JSON from path %s: %w", secretPath, err)
		}

		return secrets, nil
	}
}

// AWSSecretsFromARN returns a FetchSecrets function that retrieves Algolia credentials
// from AWS Secrets Manager using the provided secret ARN.
// The secret is expected to contain JSON with app_id and write_api_key fields.
func AWSSecretsFromARN(ctx context.Context, client SecretsManagerClient, secretArn string) FetchSecrets {
	return func() (Secrets, error) {
		input := &secretsmanager.GetSecretValueInput{
			SecretId: aws.String(secretArn),
		}

		result, err := client.GetSecretValue(ctx, input)
		if err != nil {
			return Secrets{}, fmt.Errorf("failed to get secret from AWS Secrets Manager with ARN %s: %w", secretArn, err)
		}

		if result.SecretString == nil {
			return Secrets{}, fmt.Errorf("secret with ARN %s has no string value", secretArn)
		}

		var secrets Secrets
		if err := json.Unmarshal([]byte(aws.ToString(result.SecretString)), &secrets); err != nil {
			return Secrets{}, fmt.Errorf("failed to unmarshal secret JSON from ARN %s: %w", secretArn, err)
		}

		return secrets, nil
	}
}
