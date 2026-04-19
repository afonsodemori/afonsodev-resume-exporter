# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this project does

A Go CLI tool that exports resume documents from Google Docs in multiple formats, detects version changes by binary comparison, and uploads new versions to Cloudflare R2 (S3-compatible). It's designed to run on a schedule (cron) on a remote server.

## Commands

```bash
make run      # Run the app (loads .env automatically)
make build    # Cross-compile for linux/darwin × amd64/arm64
make clear    # Delete all files in .data/
go run .      # Run without Makefile
```

`make run` and `make build` both require a `.env` file in the project root (the Makefile does `include .env; export`).

## Required environment variables

| Variable | Description |
|---|---|
| `DOCUMENT_IDS` | JSON object mapping language code → Google Doc ID, e.g. `{"en":"abc123","pt":"xyz456"}` |
| `DOCUMENT_FORMATS` | JSON array of export formats, e.g. `["pdf","docx","md","txt","odt"]` |
| `CLOUDFLARE_ACCOUNT_ID` | Cloudflare account ID |
| `CLOUDFLARE_R2_ACCESS_KEY_ID` | R2 access key |
| `CLOUDFLARE_R2_SECRET_ACCESS_KEY` | R2 secret key |
| `CLOUDFLARE_R2_PUBLIC_API` | Full URL including bucket path, e.g. `https://pub-xxx.r2.dev/my-bucket` |

## Architecture

All code is in a single `main` package with four files:

- **`main.go`** — Orchestration. Reads config, calls downloader per (lang, format), compares new vs existing file using the first format only to detect changes, then uploads changed files to R2 under two key names (legacy + current), archives the old version.
- **`downloader.go`** — Downloads from the Google Docs export API (`/export?format=<fmt>`). Saves as `{lang}-new.{format}`. If format is `md`, also converts to HTML and saves as `{lang}-new.html`.
- **`uploader.go`** — Uploads to Cloudflare R2 using the AWS SDK v2 with a custom endpoint resolver. Derives MIME type from file extension.
- **`fileutils.go`** — Binary file comparison (chunk-by-chunk) and file deletion.

### File naming convention in `.data/`

- `{lang}.{format}` — current/committed version
- `{lang}-new.{format}` — freshly downloaded, pending comparison/upload
- `{lang}-YYMMDD-HHMM.{format}` — archived previous version after an update

### R2 upload keys

Each changed file is uploaded under two keys:
- Legacy: `resume-{lang}-afonso_de_mori.{format}`
- Current: `afonso-de-mori-cv-{lang}.{format}`

### Change detection logic

Only the **first format** in `DOCUMENT_FORMATS` is used to compare new vs existing. If unchanged, all formats for that language are skipped. If changed (or no previous file exists), all formats are downloaded and uploaded.

## CI/CD

Triggered on `v*.*.*` tags. Builds binaries for linux/darwin/windows × amd64/arm64, creates a GitHub release with all artifacts, and deploys the linux-amd64 binary to a remote server via SSH (symlinking it as `afonsodev-resume-updater`).