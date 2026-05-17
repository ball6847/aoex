.DEFAULT_GOAL := help

# Colors for pretty output
BLUE := \033[36m
GREEN := \033[32m
YELLOW := \033[33m
RED := \033[31m
RESET := \033[0m

.PHONY: help
help: ## Show this help message
	@echo "$(BLUE)Available targets:$(RESET)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-15s$(RESET) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: fmt
fmt: ## Format Go code (gofmt + goimports)
	@echo "$(BLUE)Formatting code...$(RESET)"
	@go fmt ./...

.PHONY: vet
vet: ## Run go vet
	@echo "$(BLUE)Running go vet...$(RESET)"
	@go vet ./...

.PHONY: lint
lint: ## Run golangci-lint (v2)
	@echo "$(BLUE)Running golangci-lint...$(RESET)"
	@golangci-lint run ./...

.PHONY: lint-fix
lint-fix: ## Run golangci-lint with auto-fix enabled
	@echo "$(BLUE)Running golangci-lint with auto-fix...$(RESET)"
	@golangci-lint run --fix ./...

.PHONY: test
test: ## Run all tests (./...)
	@echo "$(BLUE)Running tests...$(RESET)"
	@go test -v ./...

.PHONY: test-race
test-race: ## Run tests with race detector
	@echo "$(BLUE)Running tests with race detector...$(RESET)"
	@go test -race ./...

.PHONY: coverage
coverage: ## Generate test coverage report
	@echo "$(BLUE)Generating coverage report...$(RESET)"
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(RESET)"

.PHONY: build
build: ## Build the aoex binary (output: aoex)
	@echo "$(BLUE)Building aoex...$(RESET)"
	@go build -o aoex .
	@echo "$(GREEN)Build complete: ./aoex$(RESET)"

.PHONY: build-release
build-release: ## Build optimized release binary
	@echo "$(BLUE)Building release binary...$(RESET)"
	@go build -ldflags="-s -w" -trimpath -o aoex .
	@echo "$(GREEN)Release build complete: ./aoex$(RESET)"

.PHONY: install
install: ## Install aoex to $GOPATH/bin
	@echo "$(BLUE)Installing to $$(go env GOPATH)/bin...$(RESET)"
	@go install .
	@echo "$(GREEN)Installed to $$(go env GOPATH)/bin/aoex$(RESET)"

.PHONY: clean
clean: ## Remove build artifacts
	@echo "$(BLUE)Cleaning build artifacts...$(RESET)"
	@rm -f aoex coverage.out coverage.html
	@echo "$(GREEN)Clean complete$(RESET)"

.PHONY: ci
ci: fmt vet lint test ## Run full CI pipeline (fmt + vet + lint + test)
	@echo "$(GREEN)All CI checks passed!$(RESET)"

.PHONY: deps
deps: ## Download and verify Go module dependencies
	@echo "$(BLUE)Downloading dependencies...$(RESET)"
	@go mod download
	@go mod tidy
	@go mod verify
