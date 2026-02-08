#!/usr/bin/env bash

set -euo pipefail

# Run goimports on staged Go files
echo "Running goimports..."
goimports_output=$(goimports -l "$@")
if [[ -n "$goimports_output" ]]; then
    echo "The following files have import formatting issues:"
    echo "$goimports_output"
    echo "Please run 'goimports -w .'"
    exit 1
fi

# Run gofmt on staged Go files
echo "Running gofmt..."
gofmt_output=$(gofmt -l "$@")
if [[ -n "$gofmt_output" ]]; then
    echo "The following files need formatting:"
    echo "$gofmt_output"
    echo "Please run 'go fmt ./...' or 'gofmt -w .'"
    exit 1
fi

# Run go vet
echo "Running go vet..."
go vet ./...
