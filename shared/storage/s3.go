package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

// TODO bucket name known at compile time

var tracer = otel.Tracer("github.com/TSM-061/Raggy/shared/storage/s3")

type SignedUrl struct {
	Url           string
	SignedHeaders map[string]string
}

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

	SignedPutUrlTTL time.Duration `env:"SIGNED_PUT_URL_TTL" envDefault:"5m"`
}

type S3Bucket struct {
	client        *s3.Client
	config        *S3Config
	presignClient *s3.PresignClient
}

func NewS3Bucket(
	ctx context.Context,
	cred *S3Credentials,
	cfg *S3Config,
) (*S3Bucket, error) {
	credentialsProvider := credentials.NewStaticCredentialsProvider(
		cred.AccessKeyID,
		cred.SecretAccessKey,
		"",
	)

	options := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
		config.WithBaseEndpoint(cfg.Endpoint),
		config.WithCredentialsProvider(credentialsProvider),
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.UsePathStyle
		o.EndpointOptions.DisableHTTPS = cfg.DisableSSL
	})

	presignClient := s3.NewPresignClient(client)

	return &S3Bucket{
		config:        cfg,
		client:        client,
		presignClient: presignClient,
	}, nil
}

func (b *S3Bucket) Download(ctx context.Context, key uuid.UUID) ([]byte, error) {
	_, span := tracer.Start(ctx, "s3.GetObject")
	span.SetAttributes(
		semconv.AWSS3Bucket(b.config.Bucket),
		semconv.ServerAddress(b.config.Endpoint),
	)
	defer span.End()

	result, err := b.client.GetObject(
		ctx,
		&s3.GetObjectInput{
			Bucket: &b.config.Bucket,
			Key:    aws.String(key.String()),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}

	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("stream data into buffer: %w", err)
	}

	return data, nil
}

func (b *S3Bucket) SignedPutURL(ctx context.Context, key string) (*SignedUrl, error) {
	input := &s3.PutObjectInput{
		Bucket:   aws.String(b.config.Bucket),
		Key:      aws.String(key),
		Metadata: map[string]string{},
	}

	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(input.Metadata))

	request, err := b.presignClient.PresignPutObject(
		ctx,
		input,
		func(opts *s3.PresignOptions) {
			opts.Expires = b.config.SignedPutUrlTTL
		},
	)
	if err != nil {
		return nil, fmt.Errorf("generate presigned put request: %w", err)
	}

	url := &SignedUrl{
		Url:           request.URL,
		SignedHeaders: make(map[string]string),
	}

	for header, values := range request.SignedHeader {
		if strings.ToLower(header) == "host" {
			continue
		}

		url.SignedHeaders[header] = strings.Join(values, ", ")
	}

	return url, err
}
