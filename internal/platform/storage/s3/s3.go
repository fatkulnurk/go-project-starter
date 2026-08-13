// Package s3 implements the application/storage contract with AWS S3 or any
// S3-compatible service (MinIO, Cloudflare R2, Ceph) via the AWS SDK v2.
package s3

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/fatkulnurk/go-project-starter/internal/application/storage"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
)

// S3 is an S3/S3-compatible object store.
type S3 struct {
	client *s3.Client
	bucket string
}

// NewS3 builds a client from config. Empty endpoint means real AWS.
func NewS3(cfg config.S3Config) (*S3, error) {
	if cfg.Bucket == "" {
		return nil, errors.New("STORAGE_S3_BUCKET is required")
	}
	var creds aws.CredentialsProvider
	if cfg.AccessKey != "" {
		creds = credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")
	}
	opts := []func(*awsconfig.LoadOptions) error{}
	if creds != nil {
		opts = append(opts, awsconfig.WithCredentialsProvider(creds))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = cfg.UsePathStyle
	})
	return &S3{client: client, bucket: cfg.Bucket}, nil
}

// Put implements storage.Storage.
func (s *S3) Put(ctx context.Context, key string, r io.Reader) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   r,
	})
	return err
}

// Get implements storage.Storage.
func (s *S3) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, s.mapErr(key, err)
	}
	return out.Body, nil
}

// Delete implements storage.Storage.
func (s *S3) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

// Attributes implements storage.Storage.
func (s *S3) Attributes(ctx context.Context, key string) (storage.ObjectAttrs, error) {
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return storage.ObjectAttrs{}, s.mapErr(key, err)
	}
	size := int64(0)
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	return storage.ObjectAttrs{Size: size}, nil
}

// Presign implements storage.Presigner.
func (s *S3) Presign(ctx context.Context, key string) (string, error) {
	presigner := s3.NewPresignClient(s.client)
	out, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", err
	}
	return out.URL, nil
}

func (s *S3) mapErr(key string, err error) error {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) && (apiErr.ErrorCode() == "NoSuchKey" || apiErr.ErrorCode() == "NotFound") {
		return fmt.Errorf("object %q: %w", key, storage.ErrNotFound)
	}
	return fmt.Errorf("s3 %q: %w", key, err)
}
