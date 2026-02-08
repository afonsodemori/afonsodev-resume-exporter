package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func main() {
	now := time.Now()
	formattedTime := now.Format("2006-01-02 15:04:05")
	fmt.Println(formattedTime)

	outputDir := ".data"

	documentsIds := map[string]string{
		"en": "1aYKfrRKX0YHVZukZvMGe3cXTOIY648ZXwF3iXTGQF34",
		"es": "1TT9BpFpy6QBs1sygecTuPAHD8iPMPII1y3Rw7rNuj74",
		"pt": "1hWho1MfmHPZIXEARbHaZJydXULzVoTqSnMi0Z64dOq8",
	}

	formats := []string{"pdf", "docx", "txt", "odt", "md"}

	for lang, documentId := range documentsIds {
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
	for lang := range documentsIds {
		for _, format := range formats {
			newFilename := filepath.Join(outputDir, fmt.Sprintf("%s-new.%s", lang, format))
			oldFilename := filepath.Join(outputDir, fmt.Sprintf("%s.%s", lang, format))

			if _, err := os.Stat(newFilename); err == nil {
				ctx := context.TODO()
				r2Key := fmt.Sprintf("afonso-de-mori-cv-%s.%s", lang, format)
				if err := UploadToR2(ctx, newFilename, r2Key); err != nil {
					fmt.Printf("Error uploading %s to R2: %v\n", newFilename, err)
					continue
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
