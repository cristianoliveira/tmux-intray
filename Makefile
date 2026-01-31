.PHONY: all tests fmt check-fmt lint clean install

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

clean:
	@echo "Cleaning..."
	rm -rf .tmp

install:
	@echo "Installing tmux-intray..."
	chmod +x bin/tmux-intray
	chmod +x scripts/lint.sh
	chmod +x tmux-intray.tmux
