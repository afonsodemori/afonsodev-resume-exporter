# afonsodev-resume-exporter

A Go-based utility designed to automate the process of exporting resumes from Google Docs, converting them into various formats, and synchronizing the results with Cloudflare R2 storage. It is specifically tailored for managing multi-language resumes in different formats (e.g., PDF, Markdown, HTML).

## Main Technologies

- **Language:** Go (1.25+)
- **Storage:** Cloudflare R2 (via AWS SDK for Go v2 S3 interface)
- **Document Processing:** Google Docs Export API
- **Markdown Conversion:** `github.com/gomarkdown/markdown`

## Architecture

The application follows a simple, procedural flow:

1.  **Configuration:** Loads document IDs, formats, and R2 credentials from environment variables.
2.  **Download:** Fetches documents from Google Docs in specified formats.
3.  **Processing:** If a document is in Markdown format, it automatically generates a corresponding HTML version.
4.  **Comparison:** Compares newly downloaded files with existing ones in the `.data/` directory using byte-by-byte comparison.
5.  **Synchronization:** Uploads only modified or new files to Cloudflare R2.

---

# Building and Running

## Prerequisites

- Go 1.25 or higher.
- A `.env` file with the required environment variables (see `.env.example`).

## Key Commands

- **Run the application:**
  ```bash
  make run
  ```
- **Build binaries for multiple platforms (Linux arm64/amd64, macOS amd64):**
  ```bash
  make build
  ```
- **Clean local data:**
  ```bash
  make clear
  ```

## Required Environment Variables

The application expects the following variables to be set (either in the environment or via a `.env` file):

- `DOCUMENT_IDS`: A JSON map (e.g., `{"en": "DOC_ID_1", "pt": "DOC_ID_2"}`).
- `DOCUMENT_FORMATS`: A JSON array (e.g., `["pdf", "md"]`).
- `CLOUDFLARE_ACCOUNT_ID`: Your Cloudflare account ID.
- `CLOUDFLARE_R2_ACCESS_KEY_ID`: R2 API Access Key.
- `CLOUDFLARE_R2_SECRET_ACCESS_KEY`: R2 API Secret Key.
- `CLOUDFLARE_R2_PUBLIC_API`: The public R2 bucket URL/endpoint.

---

# Development Conventions

- **File Structure:**
  - `main.go`: Entry point and orchestration logic.
  - `downloader.go`: Handles interaction with Google Docs API and Markdown-to-HTML conversion.
  - `uploader.go`: Manages Cloudflare R2 uploads.
  - `fileutils.go`: Helper functions for file comparison and deletion.
- **Local Data:** The `.data/` directory is used as a local cache for version comparison. It is ignored by git (as per common practice, though check `.gitignore` to be sure).
- **Style:** Follows standard Go idioms. Code is organized within the `main` package for simplicity.
- **Versioning:** Comparison is done byte-by-byte; if a file hasn't changed, it is not re-uploaded to R2.
