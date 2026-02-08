package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var (
	r2BucketName string
	r2Endpoint   string
)

func downloadDoc(docID, format, outputDir, lang string) error {
	url := fmt.Sprintf(
		"https://docs.google.com/document/d/%s/export?format=%s",
		docID,
		format,
	)

	fmt.Print("Getting ", url, "... ")

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}

	filename := filepath.Join(outputDir, fmt.Sprintf("%s-new.%s", lang, format))
	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	fmt.Println("OK")
	return nil
}

func areFilesEqual(path1, path2 string) (bool, error) {
	file1, err := os.Open(path1)
	if err != nil {
		return false, fmt.Errorf("Failed to open file %s: %w", path1, err)
	}
	defer file1.Close()

	file2, err := os.Open(path2)
	if err != nil {
		return false, fmt.Errorf("Failed to open file %s: %w", path2, err)
	}
	defer file2.Close()

	const bufferSize = 4096
	buf1 := make([]byte, bufferSize)
	buf2 := make([]byte, bufferSize)

	for {
		n1, err1 := file1.Read(buf1)
		n2, err2 := file2.Read(buf2)

		if err1 != nil && err1 != io.EOF {
			return false, fmt.Errorf("error reading file %s: %w", path1, err1)
		}
		if err2 != nil && err2 != io.EOF {
			return false, fmt.Errorf("error reading file %s: %w", path2, err2)
		}

		if n1 != n2 || !bytes.Equal(buf1[:n1], buf2[:n2]) {
			return false, nil
		}

		if err1 == io.EOF && err2 == io.EOF {
			return true, nil
		}
	}
}

func deleteFile(path string) error {
	err := os.Remove(path)
	if err != nil {
		return fmt.Errorf("Failed to delete file %s: %w", path, err)
	}
	return nil
}

func uploadToR2(ctx context.Context, filePath, r2Key string) error {
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

	r2Endpoint = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	r2BucketName = strings.TrimPrefix(u.Path, "/")

	fmt.Printf("Uploading %s to R2 bucket %s at endpoint %s with key %s\n", filePath, r2BucketName, r2Endpoint, r2Key)

	if r2AccountID == "" || r2AccessKeyID == "" || r2AccessKeySecret == "" {
		return fmt.Errorf("Cloudflare R2 credentials (CLOUDFLARE_R2_ACCOUNT_ID, CLOUDFLARE_R2_ACCESS_KEY_ID, CLOUDFLARE_R2_SECRET_ACCESS_KEY) not set")
	}

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
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
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
		return fmt.Errorf("failed to upload file to R2: %w", err)
	}

	fmt.Printf("Successfully uploaded %s to R2 as %s\n", filePath, r2Key)
	return nil
}

func main() {
	now := time.Now()
	formattedTime := now.Format("2006-01-02 15:04:05")
	fmt.Println(formattedTime)

	outputDir := ".data"

	docIDs := map[string]string{
		"en": "1aYKfrRKX0YHVZukZvMGe3cXTOIY648ZXwF3iXTGQF34",
		"es": "1TT9BpFpy6QBs1sygecTuPAHD8iPMPII1y3Rw7rNuj74",
		"pt": "1hWho1MfmHPZIXEARbHaZJydXULzVoTqSnMi0Z64dOq8",
	}

	formats := []string{"pdf", "docx", "txt", "odt", "md"}

	for lang, docID := range docIDs {
		fmt.Println("=>", lang)
		firstFormatProcessed := false
		skipCurrentDocID := false

		for _, format := range formats {
			if skipCurrentDocID {
				break
			}

			newFilename := filepath.Join(outputDir, fmt.Sprintf("%s-new.%s", lang, format))
			oldFilename := filepath.Join(outputDir, fmt.Sprintf("%s.%s", lang, format))

			if err := downloadDoc(docID, format, outputDir, lang); err != nil {
				fmt.Println("Error: ", err)
				continue
			}

			// Only perform comparison and conditional skipping for the first format encountered for this docID
			if !firstFormatProcessed {
				firstFormatProcessed = true

				_, err := os.Stat(oldFilename)
				oldFileExists := !os.IsNotExist(err)

				if oldFileExists {
					areEqual, err := areFilesEqual(newFilename, oldFilename)
					if err != nil {
						fmt.Printf("Error comparing files %s and %s: %v\n", newFilename, oldFilename, err)
						continue
					}

					if areEqual {
						fmt.Printf("Files %s and %s are identical. Deleting %s and skipping to next DocID.\n", newFilename, oldFilename, newFilename)
						if err := deleteFile(newFilename); err != nil {
							fmt.Printf("Error deleting file %s: %v\n", newFilename, err)
						}
						skipCurrentDocID = true
					} else {
						fmt.Printf("Files %s and %s are different. Keeping both and continuing with other formats.\n", newFilename, oldFilename)
					}
				} else {
					fmt.Printf("No existing file %s. Keeping %s and continuing with other formats.\n", oldFilename, newFilename)
				}
			}
		}
		if skipCurrentDocID {
			continue
		}
	}

	fmt.Println("\n=> Uploading new files to Cloudflare R2 and managing local versions")
	for lang := range docIDs {
		for _, format := range formats {
			newFilename := filepath.Join(outputDir, fmt.Sprintf("%s-new.%s", lang, format))
			oldFilename := filepath.Join(outputDir, fmt.Sprintf("%s.%s", lang, format))

			if _, err := os.Stat(newFilename); err == nil {
				r2Key := fmt.Sprintf("resume-%s-afonso_de_mori.%s", lang, format)

				ctx := context.TODO()
				if err := uploadToR2(ctx, newFilename, r2Key); err != nil {
					fmt.Printf("Error uploading %s to R2: %v\n", newFilename, err)
					continue
				}

				if _, err := os.Stat(oldFilename); err == nil {
					archiveFilename := filepath.Join(outputDir, fmt.Sprintf("%s-%s.%s", lang, now.Format("060102-1504"), format))
					fmt.Printf("Archiving old file %s to %s\n", oldFilename, archiveFilename)
					if err := os.Rename(oldFilename, archiveFilename); err != nil {
						fmt.Printf("Error archiving file %s: %v\n", oldFilename, err)
						continue
					}
				}

				fmt.Printf("Promoting new file %s to %s\n", newFilename, oldFilename)
				if err := os.Rename(newFilename, oldFilename); err != nil {
					fmt.Printf("Error promoting file %s: %v\n", newFilename, err)
					continue
				}
				fmt.Printf("Successfully processed %s\n", newFilename)
			}
		}
	}
}
