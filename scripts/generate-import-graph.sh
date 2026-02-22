#!/usr/bin/env bash

set -euo pipefail

readonly output_path="docs/design/import-graph-baseline.tsv"
readonly output_header="# importer\timportee"

module_path="$(go list -m -f '{{.Path}}')"
tmp_file="$(mktemp)"

trap 'rm -f "$tmp_file"' EXIT

go list -f '{{.ImportPath}}{{range .Imports}}{{"\n"}}{{$.ImportPath}}{{"\t"}}{{.}}{{end}}' ./... |
    awk -F '\t' -v module="$module_path" '$2 ~ "^" module { print $1 "\t" $2 }' |
    LC_ALL=C sort -u >"$tmp_file"

{
    printf '%s\n' "$output_header"
    cat "$tmp_file"
} >"$output_path"
