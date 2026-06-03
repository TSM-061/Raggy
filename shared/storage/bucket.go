package storage

import (
	"context"
	"fmt"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awscredentials "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gocloud.dev/blob"
	"gocloud.dev/blob/s3blob"
)

type S3Credentials struct {
	AccessKeyID     string `env:"ACCESS_KEY_ID,required,notEmpty"`
	SecretAccessKey string `env:"SECRET_ACCESS_KEY,required,notEmpty"`
}

type S3Config struct {
	Bucket       string `env:"BUCKET" envDefault:"uploads"`
	Region       string `env:"REGION" envDefault:"ap-southeast-2"`
	Endpoint     string `env:"ENDPOINT,required"`
	UsePathStyle bool   `env:"USE_PATH_STYLE" envDefault:"true"`
	DisableSSL   bool   `env:"DISABLE_SSL" envDefault:"false"`
}

func OpenS3Bucket(ctx context.Context, cred *S3Credentials, cfg *S3Config) (*blob.Bucket, error) {
	credentialsProvider := awscredentials.NewStaticCredentialsProvider(
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
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.UsePathStyle
		o.EndpointOptions.DisableHTTPS = cfg.DisableSSL
	})

	bucket, err := s3blob.OpenBucket(ctx, client, cfg.Bucket, nil)
	if err != nil {
		return nil, fmt.Errorf("open s3 bucket: %w", err)
	}

	return bucket, nil
}
