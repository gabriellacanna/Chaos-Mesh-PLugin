.PHONY: build build-debug clean test lint fmt vet deps tidy

# Build variables
BINARY_NAME=chaos-mesh-plugin
VERSION?=v0.1.0
BUILD_DIR=dist
LDFLAGS=-ldflags "-X main.version=${VERSION}"

# Go variables
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

# Default target
all: clean deps test build

# Build the plugin
build:
	@echo "Building ${BINARY_NAME} for ${GOOS}/${GOARCH}..."
	@mkdir -p ${BUILD_DIR}
	CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-${GOOS}-${GOARCH} .

# Build debug version with debug symbols
build-debug:
	@echo "Building debug version of ${BINARY_NAME}..."
	@mkdir -p ${BUILD_DIR}
	CGO_ENABLED=0 go build -gcflags="all=-N -l" -o ${BUILD_DIR}/${BINARY_NAME}-debug .

# Build for multiple platforms
build-all: clean deps
	@echo "Building for multiple platforms..."
	@mkdir -p ${BUILD_DIR}
	GOOS=linux GOARCH=amd64 $(MAKE) build
	GOOS=linux GOARCH=arm64 $(MAKE) build
	GOOS=darwin GOARCH=amd64 $(MAKE) build
	GOOS=darwin GOARCH=arm64 $(MAKE) build
	GOOS=windows GOARCH=amd64 $(MAKE) build

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf ${BUILD_DIR}

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Lint code
lint:
	@echo "Running linter..."
	golangci-lint run

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Vet code
vet:
	@echo "Vetting code..."
	go vet ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	go mod tidy

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run the plugin locally (for testing)
run:
	@echo "Running plugin locally..."
	go run . 

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t chaos-mesh-plugin:${VERSION} .

# Docker build multi-arch
docker-buildx:
	@echo "Building multi-arch Docker image..."
	docker buildx build --platform linux/amd64,linux/arm64 -t chaos-mesh-plugin:${VERSION} .

# Release (build all platforms and create checksums)
release: clean build-all
	@echo "Creating release artifacts..."
	@cd ${BUILD_DIR} && \
	for file in *; do \
		if [ -f "$$file" ]; then \
			sha256sum "$$file" > "$$file.sha256"; \
		fi \
	done
	@echo "Release artifacts created in ${BUILD_DIR}/"

# Help
help:
	@echo "Available targets:"
	@echo "  build         - Build the plugin for current platform"
	@echo "  build-debug   - Build debug version with symbols"
	@echo "  build-all     - Build for all supported platforms"
	@echo "  clean         - Clean build artifacts"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  lint          - Run linter"
	@echo "  fmt           - Format code"
	@echo "  vet           - Vet code"
	@echo "  deps          - Download dependencies"
	@echo "  tidy          - Tidy dependencies"
	@echo "  install-tools - Install development tools"
	@echo "  run           - Run plugin locally"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-buildx - Build multi-arch Docker image"
	@echo "  release       - Create release artifacts"
	@echo "  help          - Show this help"