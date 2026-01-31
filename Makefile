.PHONY: all tests lint clean install

all: tests lint

tests:
	@echo "Running tests..."
	bats tests

lint:
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
