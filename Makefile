run:
	go run . -v

clear:
	rm -rf .data/* dist/*

build-snapshot:
	@echo "Building SNAPSHOT with GoReleaser..."
	@goreleaser build --clean --auto-snapshot

release-test:
	@goreleaser release --clean --auto-snapshot --skip=publish

run-builded:
	@echo "Running the built binary..."
	@dist/afonsodev-resume-sync_linux_arm64_v8.0/afonsodev-resume-sync
