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

var (
	r2BucketName string
	r2Endpoint   string
)

func UploadToR2(ctx context.Context, filePath, r2Key string) error {
	r2AccountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	r2AccessKeyID := os.Getenv("CLOUDFLARE_R2_ACCESS_KEY_ID")
	r2AccessKeySecret := os.Getenv("CLOUDFLARE_R2_SECRET_ACCESS_KEY")
	r2PublicAPI := os.Getenv("CLOUDFLARE_R2_PUBLIC_API")

	if r2PublicAPI == "" {
		return fmt.Errorf("CLOUDFLARE_R2_PUBLIC_API environment variable not set")
	}

	u, err := url.Parse(r2PublicAPI)
	if err != nil {
		return fmt.Errorf("failed to parse CLOUDFLARE_R2_PUBLIC_API: %w", err)
	}

	if r2AccountID == "" || r2AccessKeyID == "" || r2AccessKeySecret == "" {
		return fmt.Errorf("Cloudflare R2 credentials (CLOUDFLARE_ACCOUNT_ID, CLOUDFLARE_R2_ACCESS_KEY_ID, CLOUDFLARE_R2_SECRET_ACCESS_KEY) not set")
	}

	r2Endpoint = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	r2BucketName = strings.TrimPrefix(u.Path, "/")

	fmt.Printf("Uploading %s with key %s... ", filePath, r2Key)

	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", r2AccountID),
		}, nil
	})

	cfg := aws.Config{
		Credentials:                 credentials.NewStaticCredentialsProvider(r2AccessKeyID, r2AccessKeySecret, ""),
		Region:                      "auto",
		EndpointResolverWithOptions: r2Resolver,
	}

	client := s3.NewFromConfig(cfg)

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("Failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	contentType := mime.TypeByExtension(filepath.Ext(filePath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &r2BucketName,
		Key:         &r2Key,
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("Failed to upload file to R2: %w", err)
	}

	fmt.Println("OK")
	return nil
}
