# Developer and CI entry points. Run `make help` for the list.

VERSION ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
IMAGE   ?= cart-service:$(VERSION)
BIN     := bin/cart-service

.DEFAULT_GOAL := help
.PHONY: help fmt lint test cover vuln build run docker compose-up compose-down ci

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-14s %s\n", $$1, $$2}'

fmt: ## Format code (gofumpt via golangci-lint)
	golangci-lint fmt ./...

lint: ## Run golangci-lint (install: https://golangci-lint.run/docs/welcome/install/)
	golangci-lint run ./...

test: ## Run all tests with the race detector
	go test -race -count=1 ./...

cover: ## Run tests and write coverage.out / coverage.html
	go test -race -count=1 -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

vuln: ## Scan dependencies and code for known vulnerabilities
	go tool govulncheck ./...

build: ## Build the service binary into bin/
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o $(BIN) ./cmd/cart-service

run: ## Run locally (reads .env if present)
	go run ./cmd/cart-service

docker: ## Build the container image
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE) .

compose-up: ## Start the service and local dependencies
	docker compose up --build -d

compose-down: ## Stop local dependencies
	docker compose down

ci: lint test vuln build ## Run the same checks as CI
