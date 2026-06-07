package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

type config struct {
	documentIDs map[string]string
	formats     []string
}

func loadConfig() config {
	idsJSON := os.Getenv("DOCUMENT_IDS")
	if idsJSON == "" {
		log.Fatal("DOCUMENT_IDS environment variable is not set")
	}

	var documentIDs map[string]string
	if err := json.Unmarshal([]byte(idsJSON), &documentIDs); err != nil {
		log.Fatalf("failed to parse DOCUMENT_IDS: %v", err)
	}
	if len(documentIDs) == 0 {
		log.Fatal("DOCUMENT_IDS must contain at least one document ID")
	}

	formatsJSON := os.Getenv("DOCUMENT_FORMATS")
	if formatsJSON == "" {
		log.Fatal("DOCUMENT_FORMATS environment variable is not set")
	}

	var formats []string
	if err := json.Unmarshal([]byte(formatsJSON), &formats); err != nil {
		log.Fatalf("failed to parse DOCUMENT_FORMATS: %v", err)
	}
	if len(formats) == 0 {
		log.Fatal("DOCUMENT_FORMATS must contain at least one format")
	}

	return config{documentIDs: documentIDs, formats: formats}
}

func main() {
	const outputDir = ".data"
	now := time.Now()
	cfg := loadConfig()

	fmt.Printf("Starting at %s\n", now.Format("2006-01-02 15:04:05"))

	for lang, documentID := range cfg.documentIDs {
		fmt.Printf("\n=> %s\n", lang)

		firstFormat := cfg.formats[0]
		newFile := filepath.Join(outputDir, fmt.Sprintf("%s-new.%s", lang, firstFormat))
		oldFile := filepath.Join(outputDir, fmt.Sprintf("%s.%s", lang, firstFormat))

		if err := downloadDocument(documentID, firstFormat, outputDir, lang); err != nil {
			fmt.Printf("error: %v\n", err)
			continue
		}

		if _, err := os.Stat(oldFile); err == nil {
			equal, err := filesEqual(newFile, oldFile)
			if err != nil {
				fmt.Printf("error comparing files: %v\n", err)
				continue
			}
			if equal {
				fmt.Printf("%s has no changes.\n", oldFile)
				_ = os.Remove(newFile)
				continue
			}
			fmt.Printf("%s has a new version!\n", oldFile)
		} else {
			fmt.Printf("First version of %s created as %s.\n", oldFile, newFile)
		}

		for _, format := range cfg.formats[1:] {
			if err := downloadDocument(documentID, format, outputDir, lang); err != nil {
				fmt.Printf("error: %v\n", err)
			}
		}
	}

	fmt.Println("\n=> Uploading new versions to Cloudflare R2 (if any)...")
	ctx := context.Background()

	uploader, err := newR2Uploader()
	if err != nil {
		log.Fatalf("failed to initialize R2 uploader: %v", err)
	}

	for lang := range cfg.documentIDs {
		for _, format := range cfg.formats {
			newFile := filepath.Join(outputDir, fmt.Sprintf("%s-new.%s", lang, format))
			oldFile := filepath.Join(outputDir, fmt.Sprintf("%s.%s", lang, format))

			if _, err := os.Stat(newFile); err != nil {
				continue
			}

			key := fmt.Sprintf("afonso-de-mori-cv-%s.%s", lang, format)
			if err := uploader.upload(ctx, newFile, key); err != nil {
				log.Fatalf("error uploading %s: %v", key, err)
			}

			if _, err := os.Stat(oldFile); err == nil {
				archiveFile := filepath.Join(outputDir, fmt.Sprintf("%s-%s.%s", lang, now.Format("060102-1504"), format))
				fmt.Printf("Archiving %s\n", archiveFile)
				if err := os.Rename(oldFile, archiveFile); err != nil {
					fmt.Printf("error archiving %s: %v\n", oldFile, err)
					continue
				}
			}

			if err := os.Rename(newFile, oldFile); err != nil {
				fmt.Printf("error renaming %s to %s: %v\n", newFile, oldFile, err)
			}
		}
	}

	fmt.Println("Done!")
}
