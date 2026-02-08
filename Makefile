.PHONY: all tests fmt check-fmt lint clean install install-docker install-npm install-go install-all verify-install security-check docs test bench-tui

# tmux-intray is a pure Go implementation

# Version and commit for build injection
VERSION ?= 1.0.0
COMMIT ?= $(shell git rev-parse --short HEAD)
LDFLAGS = -ldflags "-X github.com/cristianoliveira/tmux-intray/internal/version.Version=$(VERSION) -X github.com/cristianoliveira/tmux-intray/internal/version.Commit=$(COMMIT)"

all: tests lint
	@echo "✓ Build and test complete"

tests:
	@echo "Running tests..."
	go test ./...
	bats tests

bench-tui:
	@echo "Running TUI benchmarks..."
	go test ./internal/tui/state -bench 'Benchmark(BuildTree|ComputeVisibleNodes|UpdateViewportContentGrouped|ApplySearchFilterGrouped)$$' -benchmem -run '^$$'

fmt:
	@echo "Formatting shell scripts..."
	find . -type f -name "*.bats" -not -path "*/.git/*" -not -path "*/.tmp/*" -not -path "*/_tmp/*" -not -path "*/.bv/*" -not -path "*/.local/*" -not -path "*/.gwt/*" -not -path "*/tmp/*" -not -path "*/tmp*/*" -print0 | xargs -0 shfmt -ln bats -i 4 -w
	find . -type f \( -name "*.sh" -o -name "*.tmux" \) -not -path "*/.git/*" -not -path "*/.tmp/*" -not -path "*/_tmp/*" -not -path "*/.bv/*" -not -path "*/.local/*" -not -path "*/.gwt/*" -not -path "*/tmp/*" -not -path "*/tmp*/*" -print0 | xargs -0 shfmt -ln bash -i 4 -w

check-fmt:
	@echo "Checking shell script formatting..."
	@if ! command -v shfmt >/dev/null 2>&1; then \
		echo "shfmt is not installed. Install it to run formatting checks."; \
		exit 1; \
	fi
	@if find . -type f -name "*.bats" -not -path "*/.git/*" -not -path "*/.tmp/*" -not -path "*/_tmp/*" -not -path "*/.bv/*" -not -path "*/.local/*" -not -path "*/.gwt/*" -not -path "*/tmp/*" -not -path "*/tmp*/*" -print0 | xargs -0 shfmt -ln bats -i 4 -d; then \
		true; \
	else \
		echo "Some shell scripts need formatting. Run 'make fmt' to fix."; \
		exit 1; \
	fi
	@if find . -type f \( -name "*.sh" -o -name "*.tmux" \) -not -path "*/.git/*" -not -path "*/.tmp/*" -not -path "*/_tmp/*" -not -path "*/.bv/*" -not -path "*/.local/*" -not -path "*/.gwt/*" -not -path "*/tmp/*" -not -path "*/tmp*/*" -print0 | xargs -0 shfmt -ln bash -i 4 -d; then \
		true; \
	else \
		echo "Some shell scripts need formatting. Run 'make fmt' to fix."; \
		exit 1; \
	fi

go-fmt:
	@echo "Formatting Go code..."
	gofmt -w .

go-fmt-check:
	@echo "Checking Go formatting..."
	@if gofmt -d . | grep -q '^'; then \
		echo "Some Go files need formatting. Run 'make go-fmt' to fix."; \
		exit 1; \
	else \
		echo "All Go files are formatted correctly"; \
	fi

go-vet:
	@echo "Running go vet..."
	go vet ./...

go-lint: go-fmt-check go-vet

go-cover:
	@echo "Running Go test coverage..."
	go test ./... -coverprofile=coverage.out
	@echo "Coverage summary:"
	@go tool cover -func=coverage.out | tail -n 1

go-cover-html: go-cover
	@echo "Generating HTML coverage report..."
	@go tool cover -html=coverage.out -o coverage.html
	@echo "HTML report: coverage.html"

go-build:
	@echo "Building Go binary..."
	@echo "  Version: $(VERSION)"
	@echo "  Commit: $(COMMIT)"
	go build $(LDFLAGS) -o tmux-intray ./cmd/tmux-intray

lint: check-fmt go-lint
	@echo "Running linter..."
	./scripts/lint.sh

docs:
	@echo "Generating documentation..."
	./scripts/generate-docs.sh
	@echo "✓ Documentation generated"

security-check:
	@echo "Running security checks..."
	./scripts/security-check.sh

verify-install:
	@echo "Verifying install.sh..."
	shellcheck install.sh

clean:
	@echo "Cleaning..."
	rm -rf bin/tmux-intray

install:
	@echo "Installing tmux-intray..."
	@echo "  Version: $(VERSION)"
	@echo "  Commit: $(COMMIT)"
	go install $(LDFLAGS) ./cmd/tmux-intray
	chmod +x scripts/lint.sh
	chmod +x scripts/security-check.sh
	chmod +x tmux-intray.tmux
	chmod +x install.sh
	@echo "✓ Installation complete"
	@echo "  - Go binary installed to: $$(go env GOPATH)/bin/tmux-intray"


install-npm:
	@echo "Installing via npm..."
	@echo "Building Go binary for npm package..."
	@echo "  Version: $(VERSION)"
	@echo "  Commit: $(COMMIT)"
	@mkdir -p bin
	@go build $(LDFLAGS) -o bin/tmux-intray ./cmd/tmux-intray
	npm install -g .

install-go:
	@echo "Building and installing Go binary..."
	@echo "  Version: $(VERSION)"
	@echo "  Commit: $(COMMIT)"
	go build $(LDFLAGS) -o tmux-intray ./cmd/tmux-intray
	@echo "✓ Go binary built: ./tmux-intray"
	@echo "  You can run it with: ./tmux-intray --help"

install-docker:
	@echo "Docker support not yet implemented for Go-only version"
	@echo "  Skipping Docker build (no Dockerfile present)"
	@echo "✓ Docker install check passed (no-op)"

install-all: install-docker install-npm install-go
