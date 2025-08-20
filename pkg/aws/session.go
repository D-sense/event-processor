package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/d-sense/event-processor/internal/config"
)

// NewSession creates a new AWS session based on configuration
func NewSession(cfg *config.Config) (*session.Session, error) {
	awsConfig := &aws.Config{
		Region: aws.String(cfg.AWSRegion),
	}

	// For LocalStack or custom endpoint
	if cfg.AWSEndpointURL != "" {
		awsConfig.Endpoint = aws.String(cfg.AWSEndpointURL)
		awsConfig.S3ForcePathStyle = aws.Bool(true) // Required for LocalStack
		awsConfig.DisableSSL = aws.Bool(true)       // For local development

		// Use static credentials for LocalStack
		awsConfig.Credentials = credentials.NewStaticCredentials(
			cfg.AWSAccessKeyID,
			cfg.AWSSecretAccessKey,
			"",
		)
	}

	return session.NewSession(awsConfig)
}

// GetAWSConfig returns AWS config for the session
func GetAWSConfig(cfg *config.Config) *aws.Config {
	awsConfig := &aws.Config{
		Region: aws.String(cfg.AWSRegion),
	}

	if cfg.AWSEndpointURL != "" {
		awsConfig.Endpoint = aws.String(cfg.AWSEndpointURL)
		awsConfig.S3ForcePathStyle = aws.Bool(true)
		awsConfig.DisableSSL = aws.Bool(true)

		awsConfig.Credentials = credentials.NewStaticCredentials(
			cfg.AWSAccessKeyID,
			cfg.AWSSecretAccessKey,
			"",
		)
	}

	return awsConfig
}
