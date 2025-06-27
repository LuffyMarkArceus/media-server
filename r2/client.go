package r2

import (
	"context"
	"fmt"
	"media-server/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func NewR2Client() (*s3.Client, error) {
	accountID := config.CloudflareR2AccountID
	accessKeyID := config.CloudflareR2AccessKeyID
	accessKeySecret := config.CloudflareR2SecretAccessKey

	if accountID == "" || accessKeyID == "" || accessKeySecret == "" {
		return nil, fmt.Errorf("R2 credentials are not set in the environment variables")
	}

	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID),
		}, nil
	})

	cfg, err := awsConfig.LoadDefaultConfig(context.TODO(),
		awsConfig.WithEndpointResolverWithOptions(r2Resolver),
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, accessKeySecret, "")),
		awsConfig.WithRegion("auto"), // R2 region is typically 'auto'
	)

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for R2: %w", err)
	}

	client := s3.NewFromConfig(cfg)
	return client, nil
}