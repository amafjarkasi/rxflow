.PHONY: all build test clean run-api run-relay run-routing docker-up docker-down migrate lint

# Build configuration
BINARY_DIR := bin
GO := go
GOFLAGS := -v

# Services
SERVICES := ingestion-api outbox-relay routing-service migrate

all: build

build: $(SERVICES)

$(SERVICES):
	$(GO) build $(GOFLAGS) -o $(BINARY_DIR)/$@ ./cmd/$@

test:
	$(GO) test -v -race -cover ./...

test-integration:
	$(GO) test -v -tags=integration ./...

clean:
	rm -rf $(BINARY_DIR)
	$(GO) clean

# Development commands
run-api:
	$(GO) run ./cmd/ingestion-api

run-relay:
	$(GO) run ./cmd/outbox-relay

run-routing:
	$(GO) run ./cmd/routing-service

# Docker commands
docker-up:
	docker-compose -f deploy/docker/docker-compose.yml up -d

docker-down:
	docker-compose -f deploy/docker/docker-compose.yml down

docker-logs:
	docker-compose -f deploy/docker/docker-compose.yml logs -f

# Database
migrate:
	$(GO) run ./cmd/migrate

migrate-down:
	$(GO) run ./cmd/migrate -down

# Code quality
lint:
	golangci-lint run ./...

fmt:
	$(GO) fmt ./...

# Dependencies
deps:
	$(GO) mod download
	$(GO) mod tidy

# Load testing
load-test:
	k6 run test/load/prescription_load.js

# Build all for production
build-all:
	CGO_ENABLED=0 GOOS=linux $(GO) build -o $(BINARY_DIR)/ingestion-api ./cmd/ingestion-api
	CGO_ENABLED=0 GOOS=linux $(GO) build -o $(BINARY_DIR)/outbox-relay ./cmd/outbox-relay
	CGO_ENABLED=0 GOOS=linux $(GO) build -o $(BINARY_DIR)/routing-service ./cmd/routing-service
