.PHONY: all build test clean docker-up docker-down docker-build migrate proto lint help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt

# Directories
SERVICES_DIR=services
PKG_DIR=pkg
PROTO_DIR=proto
MIGRATIONS_DIR=migrations

# Docker
DOCKER_COMPOSE=docker-compose

# Proto
PROTOC=protoc

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

all: proto build ## Build everything

# ==================== Development ====================

dev: docker-infra ## Start infrastructure and run services locally
	@echo "Infrastructure started. Run services manually or use 'make run-users', 'make run-chat', etc."

docker-infra: ## Start only infrastructure (postgres, rabbitmq, centrifugo)
	$(DOCKER_COMPOSE) up -d postgres rabbitmq centrifugo
	@echo "Waiting for services to be healthy..."
	@sleep 5

docker-up: ## Start all services with docker-compose
	$(DOCKER_COMPOSE) up -d

docker-down: ## Stop all docker-compose services
	$(DOCKER_COMPOSE) down

docker-build: ## Build all docker images
	$(DOCKER_COMPOSE) build

docker-logs: ## Show logs from all services
	$(DOCKER_COMPOSE) logs -f

docker-clean: ## Remove all containers and volumes
	$(DOCKER_COMPOSE) down -v --remove-orphans

# ==================== Build ====================

build: build-users build-chat build-files build-gateway ## Build all services

build-users: ## Build users service
	$(GOBUILD) -o bin/users-server ./$(SERVICES_DIR)/users/cmd/server
	$(GOBUILD) -o bin/users-cli ./$(SERVICES_DIR)/users/cmd/cli

build-chat: ## Build chat service
	$(GOBUILD) -o bin/chat-server ./$(SERVICES_DIR)/chat/cmd/server

build-files: ## Build files service
	$(GOBUILD) -o bin/files-server ./$(SERVICES_DIR)/files/cmd/server

build-gateway: ## Build api-gateway service
	$(GOBUILD) -o bin/api-gateway ./$(SERVICES_DIR)/api-gateway/cmd/server

# ==================== Run locally ====================

run-users: ## Run users service locally
	$(GOCMD) run ./$(SERVICES_DIR)/users/cmd/server

run-chat: ## Run chat service locally
	$(GOCMD) run ./$(SERVICES_DIR)/chat/cmd/server

run-files: ## Run files service locally
	$(GOCMD) run ./$(SERVICES_DIR)/files/cmd/server

run-gateway: ## Run api-gateway locally
	$(GOCMD) run ./$(SERVICES_DIR)/api-gateway/cmd/server

# ==================== Testing ====================

test: ## Run all tests
	$(GOTEST) -v -race ./...

test-users: ## Run users service tests
	$(GOTEST) -v -race ./$(SERVICES_DIR)/users/...

test-chat: ## Run chat service tests
	$(GOTEST) -v -race ./$(SERVICES_DIR)/chat/...

test-files: ## Run files service tests
	$(GOTEST) -v -race ./$(SERVICES_DIR)/files/...

test-gateway: ## Run api-gateway tests
	$(GOTEST) -v -race ./$(SERVICES_DIR)/api-gateway/...

test-pkg: ## Run pkg tests
	$(GOTEST) -v -race ./$(PKG_DIR)/...

test-integration: ## Run integration tests
	$(GOTEST) -v -race -tags=integration ./tests/integration/...

test-coverage: ## Run tests with coverage
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# ==================== Proto ====================

proto: ## Generate protobuf code
	$(PROTOC) --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/chat/*.proto

proto-install: ## Install protoc plugins
	$(GOGET) google.golang.org/protobuf/cmd/protoc-gen-go@latest
	$(GOGET) google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# ==================== Database ====================

migrate-up: ## Run all migrations
	@echo "Running migrations..."
	@for dir in $(MIGRATIONS_DIR)/*/; do \
		echo "Applying migrations from $$dir"; \
		cat $$dir*.sql | docker exec -i chatapp-postgres psql -U chatapp -d chatapp; \
	done

migrate-create: ## Create a new migration (usage: make migrate-create name=<name> service=<service>)
	@mkdir -p $(MIGRATIONS_DIR)/$(service)
	@touch $(MIGRATIONS_DIR)/$(service)/$$(date +%Y%m%d%H%M%S)_$(name).sql
	@echo "Created migration: $(MIGRATIONS_DIR)/$(service)/$$(date +%Y%m%d%H%M%S)_$(name).sql"

# ==================== Linting ====================

lint: ## Run linters
	golangci-lint run ./...

lint-fix: ## Run linters and fix issues
	golangci-lint run --fix ./...

fmt: ## Format code
	$(GOFMT) -w -s .

# ==================== Dependencies ====================

deps: ## Download dependencies
	$(GOMOD) download

deps-tidy: ## Tidy dependencies
	$(GOMOD) tidy

deps-update: ## Update dependencies
	$(GOGET) -u ./...
	$(GOMOD) tidy

# ==================== CLI ====================

cli-add-user: ## Add user via CLI (usage: make cli-add-user username=<name> email=<email> role=<role>)
	./bin/users-cli user add --username $(username) --email $(email) --role $(role)

cli-list-users: ## List all users via CLI
	./bin/users-cli user list

# ==================== Vue SPA ====================

web-install: ## Install Vue SPA dependencies
	cd $(SERVICES_DIR)/api-gateway/web && npm install

web-dev: ## Run Vue SPA in development mode
	cd $(SERVICES_DIR)/api-gateway/web && npm run dev

web-build: ## Build Vue SPA for production
	cd $(SERVICES_DIR)/api-gateway/web && npm run build

web-lint: ## Lint Vue SPA code
	cd $(SERVICES_DIR)/api-gateway/web && npm run lint

# ==================== Cleanup ====================

clean: ## Remove build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html
