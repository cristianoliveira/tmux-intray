#!/usr/bin/env bash
# Cleanup command - Clean up old dismissed notifications

# Source core libraries
_CMD_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=../lib/core.sh disable=SC1091
source "$_CMD_DIR/../lib/core.sh"

cleanup_command() {
    local dry_run=false
    local days=""

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
        --days=*)
            days="${1#*=}"
            shift
            ;;
        --days)
            days="$2"
            shift 2
            ;;
        --dry-run)
            dry_run=true
            shift
            ;;
        --help | -h)
            cat <<EOF
tmux-intray cleanup - Clean up old dismissed notifications

USAGE:
    tmux-intray cleanup [OPTIONS]

OPTIONS:
    --days N          Clean up notifications dismissed more than N days ago
                      (default: TMUX_INTRAY_AUTO_CLEANUP_DAYS config value)
    --dry-run         Show what would be deleted without actually deleting
    -h, --help        Show this help

Automatically cleans up notifications that have been dismissed and are older
than the configured auto-cleanup days. This helps prevent storage bloat.

EOF
            return 0
            ;;
        *)
            error "Unknown argument: $1"
            echo "Usage: tmux-intray cleanup [OPTIONS]" >&2
            return 1
            ;;
        esac
    done

    ensure_tmux_running

    # Determine days threshold
    if [[ -z "$days" ]]; then
        days="$TMUX_INTRAY_AUTO_CLEANUP_DAYS"
    fi

    if ! [[ "$days" =~ ^[0-9]+$ ]]; then
        error "Invalid days value: $days (must be a positive integer)"
        return 1
    fi

    info "Starting cleanup of notifications dismissed more than $days days ago"

    # Run pre-cleanup hooks
    hooks_run "cleanup" \
        "CLEANUP_DAYS=$days" \
        "DRY_RUN=$dry_run"

    # Perform cleanup
    storage_cleanup_old_notifications "$days" "$dry_run"

    success "Cleanup completed"
}
