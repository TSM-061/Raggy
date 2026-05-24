package storage

import (
	"context"
	"fmt"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	s3Credentials "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gocloud.dev/blob"
	"gocloud.dev/blob/s3blob"
)

type S3Config struct {
	Bucket       string
	Region       string
	Endpoint     string
	UsePathStyle bool
	DisableSSL   bool
}

type S3Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
}

func OpenS3Bucket(ctx context.Context, cred *S3Credentials, cfg *S3Config) (*blob.Bucket, error) {
	if cred.SecretAccessKey == "" || cred.AccessKeyID == "" {
		return nil, fmt.Errorf("credentials are required")
	}

	if cfg.Region == "" {
		cfg.Region = "ap-southeast-2"
	}

	credentialsProvider := s3Credentials.NewStaticCredentialsProvider(
		cred.AccessKeyID,
		cred.SecretAccessKey,
		"",
	)

	loadOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithBaseEndpoint(cfg.Endpoint),
		awsconfig.WithCredentialsProvider(credentialsProvider),
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.UsePathStyle
		o.EndpointOptions.DisableHTTPS = cfg.DisableSSL
	})

	bucket, err := s3blob.OpenBucket(ctx, client, cfg.Bucket, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open s3 bucket: %w", err)
	}

	return bucket, nil
}
