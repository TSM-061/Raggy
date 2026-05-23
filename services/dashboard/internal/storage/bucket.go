package storage

import (
	"context"
	"fmt"

	"github.com/TSM-061/Raggy/dashboard/internal/config"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gocloud.dev/blob"
	"gocloud.dev/blob/s3blob"
)

func OpenBucket(ctx context.Context, bucketURL string) (*blob.Bucket, error) {
	return blob.OpenBucket(ctx, bucketURL)
}

func OpenS3Bucket(ctx context.Context, cfg *config.Config) (*blob.Bucket, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	credentialsProvider := credentials.NewStaticCredentialsProvider(
		cfg.S3AccessKeyID,
		cfg.S3SecretAccessKey,
		"",
	)

	loadOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.S3Region),
		awsconfig.WithCredentialsProvider(credentialsProvider),
	}

	if cfg.S3Endpoint != "" {
		endpoint := cfg.S3Endpoint
		loadOptions = append(loadOptions,
			awsconfig.WithBaseEndpoint(endpoint),
		)
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.S3UsePathStyle
		o.EndpointOptions.DisableHTTPS = cfg.S3DisableSSL
	})

	bucket, err := s3blob.OpenBucket(ctx, client, config.ServiceUploadsBucketName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open s3 bucket: %w", err)
	}

	return bucket, nil
}