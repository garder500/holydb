# Makefile for HolyDB

# Binary name
BINARY_NAME=holydb

# Build the application
build:
	go build -o $(BINARY_NAME) .

# Run the application
run:
	go run .

# Test the application
test:
	go test -v ./...

# Clean build artifacts
clean:
	go clean
	rm -f $(BINARY_NAME)

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Run linter (if golangci-lint is available)
lint:
	@which golangci-lint > /dev/null && golangci-lint run || echo "golangci-lint not found, skipping linting"

# Install dependencies
deps:
	go mod download
	go mod tidy

# Show help
help:
	@echo "Available targets:"
	@echo "  build    - Build the application"
	@echo "  run      - Run the application"
	@echo "  test     - Run tests"
	@echo "  clean    - Clean build artifacts"
	@echo "  fmt      - Format code"
	@echo "  vet      - Vet code"
	@echo "  lint     - Run linter"
	@echo "  deps     - Install and tidy dependencies"
	@echo "  help     - Show this help"

.PHONY: build run test clean fmt vet lint deps help