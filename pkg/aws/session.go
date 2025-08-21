package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"

	"github.com/d-sense/event-processor/internal/config"
)

// NewSession creates a new AWS config
func NewSession(cfg *config.Config) (aws.Config, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion(cfg.AWSRegion),
		awsconfig.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID:     cfg.AWSAccessKeyID,
				SecretAccessKey: cfg.AWSSecretAccessKey,
			},
		}),
	)
	if err != nil {
		return aws.Config{}, err
	}

	return awsCfg, nil
}
