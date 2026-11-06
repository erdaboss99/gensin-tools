# gensin-scan build automation. Run "make" or "make help" for the target list.
#
# The scanner only runs on Windows. Everything here targets Windows, and works from
# WSL: Go cross-compiles, and interop runs the resulting tests and tools on Windows.

export GOOS        := windows
export GOARCH      := amd64
export CGO_ENABLED := 0

BINARY     := gensin-scan
PACKAGE    := ./cmd/$(BINARY)
BIN_DIR    := bin
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS    := -s -w -X main.version=$(VERSION)
BUILDFLAGS := -trimpath
COVER_FILE := coverage.out
# The scanner and its libraries. tools/genfixtures is left out because checking it
# needs its dependency downloaded; "make fixtures" builds it when it is needed.
PACKAGES   := ./... ./control/... ./ocr/...
MODULES    := . control ocr

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-zA-Z0-9_-]+:.*## ' $(MAKEFILE_LIST) | \
		awk -F ':.*## ' '{ printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2 }'

.PHONY: all
all: check build ## Run all checks and build

.PHONY: build
build: ## Build bin/gensin-scan.exe
	go build $(BUILDFLAGS) -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/$(BINARY).exe $(PACKAGE)

.PHONY: test
test: ## Run tests on Windows (real-OCR tests are skipped when Tesseract is missing)
	go test $(PACKAGES)

.PHONY: cover
cover: ## Run tests and print total coverage
	go test -coverprofile=$(COVER_FILE) $(PACKAGES)
	go tool cover -func=$(COVER_FILE) | tail -n 1

.PHONY: vet
vet: ## Run go vet
	go vet $(PACKAGES)

.PHONY: fmt
fmt: ## Format all Go code
	gofmt -s -w .

.PHONY: fmt-check
fmt-check: ## Fail if any Go file needs formatting
	@unformatted="$$(gofmt -s -l .)"; \
	if [ -n "$$unformatted" ]; then echo "Needs gofmt:"; echo "$$unformatted"; exit 1; fi

.PHONY: deps-check
deps-check: ## Fail if the scanner or its libraries depend on anything unexpected
	@status=0; \
	for module in $(MODULES); do \
		path="$$(cd $$module && GOWORK=off go list -m)"; \
		if [ "$$module" = "." ]; then allowed="^$$path"; else allowed="^$$path$$"; fi; \
		deps="$$(cd $$module && go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./... | grep -Ev "$$allowed" || true)"; \
		if [ -n "$$deps" ]; then echo "$$path has unexpected dependencies:"; echo "$$deps"; status=1; fi; \
	done; \
	exit $$status

.PHONY: tidy
tidy: ## Tidy every module's go.mod
	@for module in $(MODULES) tools/genfixtures; do (cd $$module && GOWORK=off go mod tidy) || exit 1; done

.PHONY: check
check: fmt-check vet deps-check test ## Run every check

.PHONY: update-data
update-data: ## Refresh the reference data embedded in the binary
	go run $(PACKAGE) update-data -o internal/catalog/data

.PHONY: fixtures
fixtures: ## Regenerate OCR test images
	go run ./tools/genfixtures -o internal/scan/testdata

.PHONY: clean
clean: ## Remove build output
	rm -rf $(BIN_DIR) $(COVER_FILE)
