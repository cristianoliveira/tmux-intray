#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

module_path="$(go list -m -f '{{.Path}}')"
tmp_edges="$(mktemp)"
tmp_violations="$(mktemp)"
tmp_unknown="$(mktemp)"

trap 'rm -f "$tmp_edges" "$tmp_violations" "$tmp_unknown"' EXIT

detect_layer() {
    local package_path="$1"

    case "$package_path" in
    "$module_path/cmd" | "$module_path/cmd/"*)
        printf 'cli\n'
        ;;
    "$module_path/internal/tui" | "$module_path/internal/tui/"* | "$module_path/internal/format" | "$module_path/internal/formatter" | "$module_path/internal/formatter/"* | "$module_path/internal/status" | "$module_path/internal/app" | "$module_path/internal/app/"*)
        printf 'presentation\n'
        ;;
    "$module_path/internal/core" | "$module_path/internal/tmuxintray")
        printf 'application\n'
        ;;
    "$module_path/internal/domain" | "$module_path/internal/notification" | "$module_path/internal/search" | "$module_path/internal/dedup" | "$module_path/internal/ports")
        printf 'domain\n'
        ;;
    "$module_path/internal/storage" | "$module_path/internal/storage/"* | "$module_path/internal/tmux" | "$module_path/internal/config" | "$module_path/internal/dedupconfig" | "$module_path/internal/settings" | "$module_path/internal/hooks" | "$module_path/internal/colors" | "$module_path/internal/errors" | "$module_path/internal/logging" | "$module_path/internal/version")
        printf 'infrastructure\n'
        ;;
    *)
        printf 'unknown\n'
        ;;
    esac
}

is_denied_edge() {
    local importer_layer="$1"
    local importee_layer="$2"

    case "${importer_layer}:${importee_layer}" in
    presentation:cli | \
        application:cli | \
        application:presentation | \
        domain:cli | \
        domain:presentation | \
        domain:application | \
        domain:infrastructure | \
        infrastructure:cli | \
        infrastructure:presentation | \
        infrastructure:application)
        return 0
        ;;
    *)
        return 1
        ;;
    esac
}

go list -f '{{.ImportPath}}{{range .Imports}}{{"\n"}}{{$.ImportPath}}{{"\t"}}{{.}}{{end}}' ./... |
    awk -F '\t' -v module="$module_path" '$2 ~ "^" module { print $1 "\t" $2 }' |
    LC_ALL=C sort -u >"$tmp_edges"

while IFS=$'\t' read -r importer importee; do
    importer_layer="$(detect_layer "$importer")"
    importee_layer="$(detect_layer "$importee")"

    if [[ "$importer_layer" == "unknown" ]]; then
        printf '%s\n' "$importer" >>"$tmp_unknown"
    fi

    if [[ "$importee_layer" == "unknown" ]]; then
        printf '%s\n' "$importee" >>"$tmp_unknown"
    fi

    if [[ "$importer_layer" == "unknown" || "$importee_layer" == "unknown" ]]; then
        continue
    fi

    if is_denied_edge "$importer_layer" "$importee_layer"; then
        printf '%s\t%s\t%s\t%s\n' "$importer" "$importer_layer" "$importee" "$importee_layer" >>"$tmp_violations"
    fi
done <"$tmp_edges"

if [[ -s "$tmp_unknown" ]]; then
    printf 'ERROR: package-to-layer mapping is incomplete.\n' >&2
    printf 'Add these packages to scripts/check-import-deny-rules.sh:\n' >&2
    LC_ALL=C sort -u "$tmp_unknown" | while IFS= read -r package_path; do
        printf '  - %s\n' "$package_path" >&2
    done
    exit 1
fi

if [[ -s "$tmp_violations" ]]; then
    printf 'ERROR: forbidden package imports detected:\n' >&2
    while IFS=$'\t' read -r importer importer_layer importee importee_layer; do
        printf '  - %s (%s) -> %s (%s)\n' "$importer" "$importer_layer" "$importee" "$importee_layer" >&2
    done <"$tmp_violations"
    printf '\nDenied edges: presentation->cli, application->cli, application->presentation, domain->cli, domain->presentation, domain->application, domain->infrastructure, infrastructure->cli, infrastructure->presentation, infrastructure->application\n' >&2
    exit 1
fi

printf 'Dependency deny-rules check passed.\n'
