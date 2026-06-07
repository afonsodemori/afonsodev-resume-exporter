package main

import (
	"context"
	"fmt"
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

func newR2Uploader(cfCfg Cloudflare) (*r2Uploader, error) {
	accountID := cfCfg.AccountID
	accessKeyID := cfCfg.R2.AccessKeyID
	secretAccessKey := cfCfg.R2.SecretAccessKey
	publicAPI := cfCfg.R2.PublicAPI

	if accountID == "" || publicAPI == "" || accessKeyID == "" || secretAccessKey == "" {
		return nil, fmt.Errorf("missing Cloudflare config (AccountID, PublicAPI, AccessKeyID, SecretAccessKey)")
	}

	u, err := url.Parse(publicAPI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Cloudflare.R2.PublicAPI: %w", err)
	}

	cfg := aws.Config{
		Credentials: credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
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
