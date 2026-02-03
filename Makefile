.PHONY: build
build:
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 go build -o bin/main-linux main.go

	@echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 go build -o bin/main-darwin main.go

.PHONY: run
run:
	go run main.go
