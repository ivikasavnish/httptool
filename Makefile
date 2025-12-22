.PHONY: build install test clean fmt lint examples help

# Default target
.DEFAULT_GOAL := help

# Binary name
BINARY_NAME=httptool
VERSION?=0.1.0

# Build directory
BUILD_DIR=bin

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

help: ## Show this help message
	@echo "httptool - HTTP Execution & Evaluation Engine"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -v ./cmd/httptool
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

build-all: ## Build for multiple platforms
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/httptool
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/httptool
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/httptool
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/httptool
	@echo "Multi-platform builds complete"

install: build ## Install to /usr/local/bin
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "Installed successfully"

test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	@echo "Tests complete"

test-coverage: test ## Run tests with coverage report
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report: coverage.html"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.txt coverage.html
	@echo "Clean complete"

fmt: ## Format code
	@echo "Formatting code..."
	$(GOFMT) ./...
	@echo "Format complete"

lint: ## Run linter
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...
	@echo "Lint complete"

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "Dependencies updated"

examples: build ## Run example scripts
	@echo "Running examples..."
	@chmod +x examples/basic-curl.sh
	@echo "Example: Basic curl commands"
	./$(BUILD_DIR)/$(BINARY_NAME) exec 'curl https://httpbin.org/get'
	@echo ""
	@echo "Example: Convert to IR"
	./$(BUILD_DIR)/$(BINARY_NAME) convert 'curl https://httpbin.org/get'

evaluators: ## Set up evaluator environments
	@echo "Setting up evaluators..."
	@which bun > /dev/null || (echo "Installing Bun..." && curl -fsSL https://bun.sh/install | bash)
	@which python3 > /dev/null || (echo "Python3 not found. Please install Python 3.8+")
	@chmod +x cmd/evaluators/bun/evaluator.js
	@chmod +x cmd/evaluators/python/evaluator.py
	@echo "Evaluators ready"

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) .
	docker tag $(BINARY_NAME):$(VERSION) $(BINARY_NAME):latest
	@echo "Docker image built"

docker-run: ## Run in Docker
	docker run --rm -it $(BINARY_NAME):latest

validate-schemas: ## Validate JSON schemas
	@echo "Validating JSON schemas..."
	@which ajv > /dev/null || npm install -g ajv-cli
	ajv compile -s schemas/ir-v1.json
	ajv compile -s schemas/evaluation-context-v1.json
	ajv compile -s schemas/evaluator-decision-v1.json
	@echo "Schemas valid"

dev: build ## Development mode - build and run examples
	@echo "Development mode..."
	@make examples

release: clean test build-all ## Build release binaries
	@echo "Creating release $(VERSION)..."
	@mkdir -p release
	cd $(BUILD_DIR) && tar czf ../release/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	cd $(BUILD_DIR) && tar czf ../release/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64
	cd $(BUILD_DIR) && tar czf ../release/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	cd $(BUILD_DIR) && zip ../release/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe
	@echo "Release $(VERSION) created in release/"

.PHONY: all
all: clean deps fmt lint test build ## Run all checks and build
