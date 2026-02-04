package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
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

// areFilesEqual compares the content of two files.
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

// deleteFile deletes a file at the given path.
func deleteFile(path string) error {
	err := os.Remove(path)
	if err != nil {
		return fmt.Errorf("Failed to delete file %s: %w", path, err)
	}
	return nil
}

func main() {
	now := time.Now()
	formattedTime := now.Format("2006-01-02 15:04:05")
	fmt.Println(formattedTime)

	outputDir := ".data"

	docIDs := map[string]string{
		"en": "1aYKfrRKX0YHVZukZvMGe3cXTOIY648ZXwF3iXTGQF34",
		"pt": "1TT9BpFpy6QBs1sygecTuPAHD8iPMPII1y3Rw7rNuj74",
		"es": "1hWho1MfmHPZIXEARbHaZJydXULzVoTqSnMi0Z64dOq8",
	}

	formats := []string{"pdf", "docx", "txt", "odt", "md"}

	for lang, docID := range docIDs {
		fmt.Println("=>", lang)
		firstFormatProcessed := false // Flag to track if the first format has been processed for the current docID
		skipCurrentDocID := false     // Flag to control skipping to the next docID

		for _, format := range formats {
			if skipCurrentDocID {
				// If we decided to skip this docID (because the first format was identical),
				// break out of the inner loop to move to the next docID.
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
						// If current and -new files are equal, delete -new and skip other formats, and continue to next file ID.
						fmt.Printf("Files %s and %s are identical. Deleting %s and skipping to next DocID.\n", newFilename, oldFilename, newFilename)
						if err := deleteFile(newFilename); err != nil {
							fmt.Printf("Error deleting file %s: %v\n", newFilename, err)
						}
						skipCurrentDocID = true // Set flag to skip this docID, which will cause the 'break' and then 'continue' outer loop.
					} else {
						// If -new and current files are different, KEEP BOTH FILES and continue downloading the other formats.
						fmt.Printf("Files %s and %s are different. Keeping both and continuing with other formats.\n", newFilename, oldFilename)
					}
				} else {
					// If the current version does not exist, keep the -new file.
					fmt.Printf("No existing file %s. Keeping %s and continuing with other formats.\n", oldFilename, newFilename)
				}
			}
		}
		if skipCurrentDocID {
			continue // Move to the next docID
		}
	}
}
