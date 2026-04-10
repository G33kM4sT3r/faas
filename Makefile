BINARY := faas
VERSION ?= dev
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)

.PHONY: build test test-coverage check fmt-check vet lint compile-audit clean help vuln ci fmt e2e

build: ## Build production binary to bin/faas
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/faas/

test: ## Run all tests with race detection
	go test -race -count=1 ./...

test-coverage: ## Run tests and generate HTML coverage report
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

check: fmt-check vet lint compile-audit ## Run all checks (fmt + vet + lint + compile)

fmt-check: ## Check that all Go files are formatted
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed on:"; gofmt -l .; exit 1)

vet: ## Run go vet static analysis
	go vet ./...

lint: ## Run golangci-lint
	golangci-lint run ./...

vuln: ## Run govulncheck against the module
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

compile-audit: ## Verify production packages compile (excludes docs/examples fragments and test-only e2e)
	go build $$(go list ./... | grep -Ev '/(docs/examples|test/e2e)(/|$$)')

clean: ## Remove build artifacts and coverage files
	rm -rf bin/ coverage.out coverage.html

fmt: ## Auto-format Go files (gofmt -w)
	gofmt -w .

ci: check test vuln ## Run the full CI suite locally (check + test + vuln)

e2e: ## Run end-to-end tests (requires Docker)
	go test -race -count=1 -tags=e2e ./test/e2e/...

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'
