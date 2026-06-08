package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"
)

func main() {
	start := time.Now()
	slog.Info("starting")

	verbose := flag.Bool("v", false, "enable debug logging")
	flag.Parse()

	logLevel := slog.LevelInfo
	if *verbose {
		logLevel = slog.LevelDebug
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})))

	cfg, err := LoadConfig("config.json")
	if err != nil {
		slog.Error("load config failed", "err", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(cfg.Documents.Dirs.Archive, 0o755); err != nil {
		slog.Error("failed to create archive dir", "err", err)
		os.Exit(1)
	}

	uploader, err := newR2Uploader(cfg.Cloudflare)
	if err != nil {
		slog.Error("failed to initialize R2 uploader", "err", err)
		os.Exit(1)
	}

	slog.Info("downloads...")
	for _, document := range cfg.Documents.List {
		for loop, format := range cfg.Documents.Formats {
			downloadPath := document.DownloadPath(format)
			currentPath := document.CurrentPath(format)

			if format == "html" {
				// TODO: Temporary hack. Generated from MD. Do not download HTML.
				continue
			}

			slog.Debug("downloads", "document", document.Name, "format", format)
			if err := downloadDocument(document.ID, format, document.dirs.Output, document.Name); err != nil {
				slog.Error("download error", "err", err)
				os.Exit(1)
			}

			if loop == 0 {
				if _, err := os.Stat(currentPath); err != nil {
					slog.Info("first version created", "document", document.Name)
				} else {
					equal, err := filesEqual(currentPath, downloadPath)
					if err != nil {
						slog.Error("error comparing files", "err", err)
						break
					}
					if equal {
						slog.Debug("unchanged")
						_ = os.Remove(downloadPath)
						break
					}
					slog.Info("new version found", "document", document.Name)
				}
			}
		}
	}

	slog.Info("uploads...")
	for _, document := range cfg.Documents.List {
		for _, format := range cfg.Documents.Formats {
			downloadPath := document.DownloadPath(format)
			currentPath := document.CurrentPath(format)

			if _, err := os.Stat(downloadPath); err == nil {
				ctx := context.Background()
				key := fmt.Sprintf("afonso-de-mori-cv-%s.%s", document.Name, format)
				slog.Debug("uploading", "document", document.Name, "format", format)
				if err := uploader.upload(ctx, downloadPath, key); err != nil {
					slog.Error("upload error", "file", downloadPath, "err", err)
					os.Exit(1)
				}

				archivePath := document.ArchivePath(format)
				slog.Debug("archiving", "from", currentPath, "to", archivePath)
				_ = os.Rename(currentPath, archivePath)
				_ = os.Rename(downloadPath, currentPath)
			}
		}
	}

	slog.Info("done", "duration_ms", time.Since(start).Milliseconds())
}
