include .env
export

run:
	go run .

clear:
	rm -rf .data/*

build:
	@echo "Building for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o afonsodev-resume-exporter-linux-arm64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o afonsodev-resume-exporter-linux-amd64 .
	@echo "Building for macOS..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o afonsodev-resume-exporter-darwin-amd64 .
