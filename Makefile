.PHONY: help build run install test vet lint check bench snapshot gacp

# Default target - show help
.DEFAULT_GOAL := help

## Help:
help: ## Show this help message
	@printf "\n\033[1mbiscuit\033[0m\n"
	@printf "Generate a production-ready CLI repository from an OpenAPI 3.x spec\n"
	@printf "\n\033[1mUsage:\033[0m make \033[36m<target>\033[0m\n"
	@awk 'BEGIN {FS = ":.*##"; section=""} \
		/^## [A-Za-z]/ { section=substr($$0, 4); next } \
		/^[a-zA-Z_-]+:.*##/ { \
			if (section != "") { printf "\n\033[1m%s\033[0m\n", section; section="" } \
			printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 \
		}' $(MAKEFILE_LIST)
	@printf "\n"

## Dev:
build: ## Build all packages
	go build ./...

run: ## Run the CLI locally (Usage: make run ARGS="doctor --spec openapi.yaml")
	go run ./cmd/biscuit $(ARGS)

install: ## Install biscuit into GOPATH/bin from source
	go install ./cmd/biscuit

## Quality:
test: ## Run all tests
	go test ./...

vet: ## Run go vet
	go vet ./...

lint: ## Run golangci-lint (same linter as CI)
	@command -v golangci-lint >/dev/null || { printf "golangci-lint not installed: brew install golangci-lint\n"; exit 1; }
	golangci-lint run ./...

check: build vet test ## Build, vet, and test (add 'lint' for full CI parity)

bench: ## Run the parse→IR generation benchmark over the spec ladder
	go test ./internal/mapping/ -run xxx -bench . -benchtime 5x

## Release:
snapshot: ## Build a local goreleaser snapshot (no publish, no tag)
	@command -v goreleaser >/dev/null || { printf "goreleaser not installed: brew install goreleaser\n"; exit 1; }
	goreleaser release --snapshot --clean

## Git:
gacp: ## Git add, commit, push (Usage: make gacp M="type(scope): message")
	git add -A && git commit -m "$(M)" && git push
