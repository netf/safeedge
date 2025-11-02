.PHONY: help proto sqlc build test clean docker-up docker-down

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

proto: ## Generate protobuf code
	@echo "Generating protobuf code..."
	@mkdir -p api/proto/gen
	@protoc --go_out=api/proto/gen --go_opt=paths=source_relative \
		--go-grpc_out=api/proto/gen --go-grpc_opt=paths=source_relative \
		-I api/proto \
		api/proto/*.proto
	@echo "✓ Protobuf code generated"

sqlc: ## Generate database code
	@echo "Generating database code..."
	@sqlc generate -f internal/controlplane/database/sqlc.yaml
	@echo "✓ Database code generated"

build: ## Build all binaries
	@echo "Building binaries..."
	@go build -o bin/control-plane ./cmd/control-plane
	@go build -o bin/agent ./cmd/agent
	@go build -o bin/cli ./cmd/cli
	@echo "✓ Binaries built"

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -timeout=5m ./...

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@echo "✓ Code formatted"

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...
	@echo "✓ Vet complete"

lint: fmt vet ## Run all linters

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@echo "✓ Clean complete"

docker-up: ## Start Docker Compose infrastructure
	@echo "Starting Docker Compose..."
	@docker compose up -d
	@echo "✓ Infrastructure started"

docker-down: ## Stop Docker Compose infrastructure
	@echo "Stopping Docker Compose..."
	@docker compose down
	@echo "✓ Infrastructure stopped"

dev-setup: ## Run development setup script
	@./scripts/dev-setup.sh

db-reset: ## Reset database
	@./scripts/db-reset.sh

test-e2e: ## Run E2E tests
	@./scripts/test-all.sh
