package main

import (
	"context"
	"fmt"
	"log"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type r2Uploader struct {
	client     *s3.Client
	bucketName string
}

func newR2Uploader() (*r2Uploader, error) {
	accountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	accessKeyID := os.Getenv("CLOUDFLARE_R2_ACCESS_KEY_ID")
	secretKey := os.Getenv("CLOUDFLARE_R2_SECRET_ACCESS_KEY")
	publicAPI := os.Getenv("CLOUDFLARE_R2_PUBLIC_API")

	if accountID == "" || accessKeyID == "" || secretKey == "" {
		return nil, fmt.Errorf("missing Cloudflare R2 credentials (CLOUDFLARE_ACCOUNT_ID, CLOUDFLARE_R2_ACCESS_KEY_ID, CLOUDFLARE_R2_SECRET_ACCESS_KEY)")
	}
	if publicAPI == "" {
		return nil, fmt.Errorf("CLOUDFLARE_R2_PUBLIC_API is not set")
	}

	u, err := url.Parse(publicAPI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CLOUDFLARE_R2_PUBLIC_API: %w", err)
	}

	cfg := aws.Config{
		Credentials: credentials.NewStaticCredentialsProvider(accessKeyID, secretKey, ""),
		Region:      "auto",
	}

	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	return &r2Uploader{
		client:     client,
		bucketName: strings.TrimPrefix(u.Path, "/"),
	}, nil
}

func (u *r2Uploader) upload(ctx context.Context, filePath, key string) error {
	log.Printf("uploading %s with key %s...", filePath, key)

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	contentType := mime.TypeByExtension(filepath.Ext(filePath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err = u.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &u.bucketName,
		Key:         &key,
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("failed to upload to R2: %w", err)
	}

	return nil
}
