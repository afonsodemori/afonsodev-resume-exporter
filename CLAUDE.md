# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this project does

A Go CLI tool that exports resume documents from Google Docs in multiple formats, detects version changes by binary comparison, and uploads new versions to Cloudflare R2 (S3-compatible). It's designed to run on a schedule (cron) on a remote server.

## Commands

```bash
make run              # Run the app (loads .env automatically)
make build-snapshot   # Build binaries via GoReleaser (snapshot, no publish)
make release-test     # Full release dry-run (builds all targets, skips publish)
make run-builded      # Run the linux/arm64 binary from dist/
make clear            # Delete .data/* and dist/*
go run .              # Run without Makefile (requires env vars set manually)
```

`make run` and `make build-snapshot` both require a `.env` file in the project root (the Makefile does `include .env; export`).

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

All code is in a single `main` package with four files. There are no tests.

- **`main.go`** — Orchestration. Reads config, calls downloader per (lang, format), compares new vs existing file using the first format only to detect changes, then uploads changed files to R2 under two key names (legacy + current), archives the old version. The R2 uploader is initialized after all downloads complete, so missing R2 credentials only cause a fatal error at upload time.
- **`downloader.go`** — Downloads from the Google Docs export API (`/export?format=<fmt>`). Saves as `{lang}-new.{format}`. If format is `md`, also auto-generates `{lang}-new.html` via `gomarkdown` — no explicit `html` format entry is needed.
- **`uploader.go`** — Uploads to Cloudflare R2 using AWS SDK v2. The bucket name is extracted from the path component of `CLOUDFLARE_R2_PUBLIC_API`. MIME type is derived from file extension.
- **`fileutils.go`** — Binary file comparison (chunk-by-chunk).

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

Triggered on `v*.*.*` tags via GoReleaser. Builds binaries for linux/darwin/windows × amd64/arm64 and creates a GitHub release. After release, triggers a GitLab pipeline (`afonsodemori/packages`) via webhook to deploy the new version — the pipeline receives `APP` and `VERSION` variables and handles the server deployment.