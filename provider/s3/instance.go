package s3provider

import (
	"context"
	"fmt"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	configprovider "github.com/yunerou/niarb/provider/config-provider"
)

type client struct {
	s3Client      *s3.Client
	presignClient *s3.PresignClient
	bucket        string
	region        string
	accessKey     string
	secretKey     string
}

func New(cfg *configprovider.S3T) ObjectStorage {
	if cfg == nil {
		panic("s3provider: config must not be nil")
	}
	if cfg.Region == "" {
		panic("s3provider: region must not be empty")
	}
	if cfg.Bucket == "" {
		panic("s3provider: bucket must not be empty")
	}

	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
	}

	if cfg.StaticAccessKey != nil && cfg.StaticSecretKey != nil {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(*cfg.StaticAccessKey, *cfg.StaticSecretKey, ""),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		panic(fmt.Sprintf("s3provider: failed to load aws config: %v", err))
	}

	s3Opts := []func(*s3.Options){
		func(o *s3.Options) {
			o.UsePathStyle = cfg.ForcePathStyle
		},
	}
	if cfg.Endpoint != nil && *cfg.Endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = cfg.Endpoint
		})
	}

	s3Client := s3.NewFromConfig(awsCfg, s3Opts...)
	presignClient := s3.NewPresignClient(s3Client)

	var accessKey, secretKey string
	if cfg.StaticAccessKey != nil {
		accessKey = *cfg.StaticAccessKey
	}
	if cfg.StaticSecretKey != nil {
		secretKey = *cfg.StaticSecretKey
	}

	return &client{
		s3Client:      s3Client,
		presignClient: presignClient,
		bucket:        cfg.Bucket,
		region:        cfg.Region,
		accessKey:     accessKey,
		secretKey:     secretKey,
	}
}
