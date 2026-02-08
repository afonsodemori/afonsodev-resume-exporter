package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func DownloadDocument(documentId, format, outputDir, lang string) error {
	url := fmt.Sprintf(
		"https://docs.google.com/document/d/%s/export?format=%s",
		documentId,
		format,
	)

	fmt.Printf("Getting %s... ", url)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("Request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Unexpected status: %s", resp.Status)
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
