package main

import (
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

	filename := filepath.Join(outputDir, fmt.Sprintf("resume-%s-afonso_de_mori.%s", lang, format))
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
		for _, format := range formats {
			if err := downloadDoc(docID, format, outputDir, lang); err != nil {
				fmt.Println("Error: ", err)
			}
		}
	}
}
