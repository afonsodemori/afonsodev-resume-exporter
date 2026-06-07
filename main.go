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

	log.Printf("starting...")

	for lang, documentID := range cfg.documentIDs {
		log.Printf("=> %s", lang)

		firstFormat := cfg.formats[0]
		newFile := filepath.Join(outputDir, fmt.Sprintf("%s-new.%s", lang, firstFormat))
		oldFile := filepath.Join(outputDir, fmt.Sprintf("%s.%s", lang, firstFormat))

		if err := downloadDocument(documentID, firstFormat, outputDir, lang); err != nil {
			log.Printf("error: %v", err)
			continue
		}

		if _, err := os.Stat(oldFile); err == nil {
			equal, err := filesEqual(newFile, oldFile)
			if err != nil {
				log.Printf("error comparing files: %v", err)
				continue
			}
			if equal {
				log.Printf("%s has no changes.", oldFile)
				_ = os.Remove(newFile)
				continue
			}
			log.Printf("%s has a new version!", oldFile)
		} else {
			log.Printf("first version of %s created as %s.", oldFile, newFile)
		}

		for _, format := range cfg.formats[1:] {
			if err := downloadDocument(documentID, format, outputDir, lang); err != nil {
				log.Printf("error: %v", err)
			}
		}
	}

	log.Println("uploading new versions to Cloudflare R2 (if any)...")
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
				log.Printf("archiving %s", archiveFile)
				if err := os.Rename(oldFile, archiveFile); err != nil {
					log.Printf("error archiving %s: %v", oldFile, err)
					continue
				}
			}

			if err := os.Rename(newFile, oldFile); err != nil {
				log.Print(err)
			}
		}
	}

	log.Println("done!")
}
