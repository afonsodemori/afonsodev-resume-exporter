package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func main() {
	cfg, err := LoadConfig("config.json")
	if err != nil {
		log.Fatal(err)
	}
	const outputDir = ".data"
	now := time.Now()

	log.Printf("starting...")

	uploader, err := newR2Uploader(cfg.Cloudflare)
	if err != nil {
		log.Fatalf("failed to initialize R2 uploader: %v", err)
	}

	for _, document := range cfg.Documents.List {
		log.Printf("=> %s", document.Name)
		for loop, format := range cfg.Documents.Formats {
			newFile := filepath.Join(outputDir, fmt.Sprintf("%s-new.%s", document.Name, format))
			oldFile := filepath.Join(outputDir, fmt.Sprintf("%s.%s", document.Name, format))

			if err := downloadDocument(document.ID, format, outputDir, document.Name); err != nil {
				log.Printf("error: %v", err)
				if loop == 0 {
					break
				}
			}

			if loop == 0 {
				if _, err := os.Stat(oldFile); err != nil {
					log.Printf("first version of %s created as %s.", oldFile, newFile)
				} else {
					equal, err := filesEqual(oldFile, newFile)
					if err != nil {
						log.Printf("error comparing files: %v", err)
						break
					}
					if equal {
						log.Printf("%s has no changes.", oldFile)
						_ = os.Remove(newFile)
						break
					}
					log.Printf("%s has a new version!", oldFile)
				}
			}

			ctx := context.Background()
			key := fmt.Sprintf("afonso-de-mori-cv-%s.%s", document.Name, format)
			if err := uploader.upload(ctx, newFile, key); err != nil {
				log.Fatalf("error uploading %s: %v", key, err)
			}

			if _, err := os.Stat(oldFile); err == nil {
				archiveFile := filepath.Join(outputDir, fmt.Sprintf("%s-%s.%s", document.Name, now.Format("060102-1504"), format))
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
