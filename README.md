# afonsodev-resume-updater

A Go CLI tool that automatically exports my resume from Google Docs, detects when it changes, and publishes the new version to cloud storage — so any link pointing to my resume is always up to date without manual intervention.

## Why this exists

My resume lives in Google Docs — easy to keep up to date, available in three languages (EN, ES, PT). I share it through [my website](https://afonso.dev/resume) in multiple formats (PDF, DOCX, ODT, Markdown, plain text), so anyone can download whichever works best for them.

This tool runs on a schedule and keeps all those files in sync automatically. Edit the doc, and every format and language updates on its own.

## How it works

1. Downloads the resume from the Google Docs export API in each configured format
2. Compares the new file against the previous version (byte-by-byte)
3. If there are changes, uploads all formats to Cloudflare R2 and archives the old version
4. If nothing changed, it exits cleanly without touching anything

## Stack

- **Go** — single binary, no runtime dependencies
- **Google Docs Export API** — no OAuth needed for publicly shared documents
- **Cloudflare R2** — S3-compatible object storage via AWS SDK v2
- **GoReleaser** — cross-compilation and GitHub releases for linux/darwin/windows × amd64/arm64
- **GitLab CI** — deploys new releases to the server automatically

## Configuration

The tool is configured entirely via environment variables — document IDs, export formats, and R2 credentials. A `.env` file is used locally; secrets are injected by the CI/CD pipeline on the server.

## License

[MIT](LICENSE)
