help:

.PHONY: clear
clear:
	rm -rf .data/*

.PHONY: build
build:
	@echo "Building for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o afonso-dev-resume-exporter-linux-arm64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o afonso-dev-resume-exporter-linux-amd64 .
	@echo "Building for macOS..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o afonso-dev-resume-exporter-darwin-amd64 .

.PHONY: run
run:
	@set -a; \
	[ -f .env ] && . ./.env; \
	set +a; \
	go run .
