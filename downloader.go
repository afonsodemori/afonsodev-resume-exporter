package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

func downloadDocument(documentID, format, outputDir, lang string) error {
	url := fmt.Sprintf(
		"https://docs.google.com/document/d/%s/export?format=%s",
		documentID,
		format,
	)

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

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if _, err := out.Write(content); err != nil {
		return err
	}

	if strings.ToLower(format) == "md" {
		slog.Debug("converting to HTML...")
		if err := convertMarkdownToHTML(content, outputDir, lang); err != nil {
			return fmt.Errorf("HTML conversion failed: %w", err)
		}
	}

	return nil
}

// TODO: Downloader should not manage conversion
func convertMarkdownToHTML(mdContent []byte, outputDir, lang string) error {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	p := parser.NewWithExtensions(extensions)

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	htmlContent := markdown.ToHTML(mdContent, p, renderer)

	htmlFilename := filepath.Join(outputDir, fmt.Sprintf("%s-new.html", lang))
	return os.WriteFile(htmlFilename, htmlContent, 0o644)
}
