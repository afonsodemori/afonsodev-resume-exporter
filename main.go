package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func LoadDocumentsConfig() (map[string]string, []string) {
	idsJSON := os.Getenv("DOCUMENT_IDS")
	if idsJSON == "" {
		panic("DOCUMENT_IDS environment variable is not set")
	}

	var documentIds map[string]string
	if err := json.Unmarshal([]byte(idsJSON), &documentIds); err != nil {
		panic(fmt.Sprintf("Failed to parse DOCUMENT_IDS: %v", err))
	}
	if len(documentIds) == 0 {
		panic("DOCUMENT_IDS must contain at least one document ID")
	}

	formatsJSON := os.Getenv("DOCUMENT_FORMATS")
	if formatsJSON == "" {
		panic("DOCUMENT_FORMATS environment variable is not set")
	}

	var formats []string
	if err := json.Unmarshal([]byte(formatsJSON), &formats); err != nil {
		panic(fmt.Sprintf("Failed to parse DOCUMENT_FORMATS: %v", err))
	}
	if len(formats) == 0 {
		panic("DOCUMENT_FORMATS must contain at least one format")
	}

	return documentIds, formats
}

func main() {
	now := time.Now()
	formattedTime := now.Format("2006-01-02 15:04:05")
	fmt.Println(formattedTime)

	outputDir := ".data"
	documentIds, formats := LoadDocumentsConfig()

	for lang, documentId := range documentIds {
		fmt.Println("\n=>", lang)
		firstFormatProcessed := false
		skipCurrentDocumentId := false

		for _, format := range formats {
			if skipCurrentDocumentId {
				break
			}

			if err := DownloadDocument(documentId, format, outputDir, lang); err != nil {
				fmt.Println("Error: ", err)
				continue
			}

			if !firstFormatProcessed {
				firstFormatProcessed = true

				newFilename := filepath.Join(outputDir, fmt.Sprintf("%s-new.%s", lang, format))
				oldFilename := filepath.Join(outputDir, fmt.Sprintf("%s.%s", lang, format))

				_, err := os.Stat(oldFilename)
				oldFileExists := !os.IsNotExist(err)

				if oldFileExists {
					areEqual, err := AreFilesEqual(newFilename, oldFilename)
					if err != nil {
						fmt.Printf("Error comparing files %s and %s: %v\n", newFilename, oldFilename, err)
						continue
					}

					if areEqual {
						fmt.Printf("%s has no changes.\n", oldFilename)
						if err := DeleteFile(newFilename); err != nil {
							fmt.Printf("Error deleting file %s: %v\n", newFilename, err)
						}
						skipCurrentDocumentId = true
					} else {
						fmt.Printf("%s has a new version!\n", oldFilename)
					}
				} else {
					fmt.Printf("First version of %s just created as %s.\n", oldFilename, newFilename)
				}
			}
		}
		if skipCurrentDocumentId {
			continue
		}
	}

	fmt.Println("\n=> Uploading new versions to Cloudflare R2 (if any)...")
	for lang := range documentIds {
		for _, format := range formats {
			newFilename := filepath.Join(outputDir, fmt.Sprintf("%s-new.%s", lang, format))
			oldFilename := filepath.Join(outputDir, fmt.Sprintf("%s.%s", lang, format))

			if _, err := os.Stat(newFilename); err == nil {
				ctx := context.TODO()

				// TODO: Temporary hack to avoid 404 for now.
				r2KeyLegacy := fmt.Sprintf("resume-%s-afonso_de_mori.%s", lang, format)
				if err := UploadToR2(ctx, newFilename, r2KeyLegacy); err != nil {
					panic(fmt.Sprintf("Error uploading %s to R2: %v", r2KeyLegacy, err))
				}

				r2Key := fmt.Sprintf("afonso-de-mori-cv-%s.%s", lang, format)
				if err := UploadToR2(ctx, newFilename, r2Key); err != nil {
					panic(fmt.Sprintf("Error uploading %s to R2: %v", r2Key, err))
				}

				if _, err := os.Stat(oldFilename); err == nil {
					archiveFilename := filepath.Join(outputDir, fmt.Sprintf("%s-%s.%s", lang, now.Format("060102-1504"), format))
					fmt.Printf("Archiving %s\n", archiveFilename)
					if err := os.Rename(oldFilename, archiveFilename); err != nil {
						fmt.Printf("Error archiving file %s: %v\n", oldFilename, err)
						continue
					}
				}

				if err := os.Rename(newFilename, oldFilename); err != nil {
					fmt.Printf("Error renaming file %s: %v\n", newFilename, err)
					continue
				}
			}
		}
	}

	fmt.Println("Done!")
}
