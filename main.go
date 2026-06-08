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

	slog.Info("starting")

	uploader, err := newR2Uploader(cfg.Cloudflare)
	if err != nil {
		slog.Error("failed to initialize R2 uploader", "err", err)
		os.Exit(1)
	}

	for _, document := range cfg.Documents.List {
		slog.Info("processing", "document", document.Name)
		for loop, format := range cfg.Documents.Formats {
			newFile := document.NewPath(format)
			oldFile := document.CurrentPath(format)

			slog.Debug("downloading", "document", document.Name, "format", format)
			if err := downloadDocument(document.ID, format, document.dirs.Output, document.Name); err != nil {
				slog.Error("download error", "err", err)
				if loop == 0 {
					break
				}
			}

			if loop == 0 {
				if _, err := os.Stat(oldFile); err != nil {
					slog.Info("first version created", "new_file", newFile)
				} else {
					equal, err := filesEqual(oldFile, newFile)
					if err != nil {
						slog.Error("error comparing files", "err", err)
						break
					}
					if equal {
						slog.Info("unchanged")
						_ = os.Remove(newFile)
						break
					}
					slog.Info("new version found")
				}
			}

			ctx := context.Background()
			key := fmt.Sprintf("afonso-de-mori-cv-%s.%s", document.Name, format)
			slog.Debug("uploading", "document", document.Name, "format", format)
			if err := uploader.upload(ctx, newFile, key); err != nil {
				slog.Error("upload error", "file", newFile, "err", err)
				os.Exit(1)
			}

			if _, err := os.Stat(oldFile); err == nil {
				archiveFile := document.ArchivePath(format)
				slog.Debug("archiving", "archive_file", archiveFile)
				if err := os.Rename(oldFile, archiveFile); err != nil {
					slog.Error("error archiving", "err", err)
					continue
				}
			}

			if err := os.Rename(newFile, oldFile); err != nil {
				slog.Error("renaming error", "err", err)
			}
		}
	}

	slog.Info("done", "duration_ms", time.Since(start).Milliseconds())
}
