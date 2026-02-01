.PHONY: all tests fmt check-fmt lint clean install install-homebrew install-docker install-npm install-go install-all verify-install security-check docs

all: tests lint

tests:
	@echo "Running tests..."
	bats tests

fmt:
	@echo "Formatting shell scripts..."
	find . -type f \( -name "*.sh" -o -name "*.bats" -o -name "*.tmux" \) -not -path "*/.git/*" -not -path "*/.tmp/*" -not -path "*/.bv/*" -not -path "*/.local/*" -not -path "*/.gwt/*" -not -path "*/tmp/*" -not -path "*/tmp*/*" -print0 | xargs -0 shfmt -w

check-fmt:
	@echo "Checking shell script formatting..."
	@if find . -type f \( -name "*.sh" -o -name "*.bats" -o -name "*.tmux" \) -not -path "*/.git/*" -not -path "*/.tmp/*" -not -path "*/.bv/*" -not -path "*/.local/*" -not -path "*/.gwt/*" -not -path "*/tmp/*" -not -path "*/tmp*/*" -print0 | xargs -0 shfmt -d 2>/dev/null; then \
		echo "All shell scripts are formatted correctly"; \
	else \
		echo "Some shell scripts need formatting. Run 'make fmt' to fix."; \
		exit 1; \
	fi

lint: check-fmt
	@echo "Running linter..."
	./scripts/lint.sh

docs:
	@echo "Generating documentation..."
	./scripts/generate-docs.sh

security-check:
	@echo "Running security checks..."
	./scripts/security-check.sh

clean:
	@echo "Cleaning..."
	rm -rf .tmp

install:
	@echo "Installing tmux-intray..."
	chmod +x bin/tmux-intray
	chmod +x scripts/lint.sh
	chmod +x scripts/security-check.sh
	chmod +x tmux-intray.tmux

verify-install:
	@echo "Verifying install.sh..."
	shellcheck install.sh

install-homebrew:
	@echo "Installing via Homebrew..."
	brew install ./Formula/tmux-intray.rb

install-docker:
	@echo "Building Docker image..."
	docker build -t tmux-intray .

install-npm:
	@echo "Installing via npm..."
	npm install -g .

install-go:
	@echo "Building Go binary..."
	go build -o tmux-intray-go ./cmd/tmux-intray

install-all: install-homebrew install-docker install-npm install-go
