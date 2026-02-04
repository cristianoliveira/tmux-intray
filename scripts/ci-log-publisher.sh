#!/usr/bin/env bash
# CI Log Publisher - Publishes and views CI debug logs

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Find log files
find_log_files() {
    local -a log_files=()

    # Look for CI debug logs
    while IFS= read -r -d '' file; do
        if [[ "$file" == *ci-debug.log ]]; then
            log_files+=("$file")
        fi
    done < <(find "$PROJECT_ROOT" -type f -name "*ci-debug.log" -print0)

    # Look for general debug logs
    while IFS= read -r -d '' file; do
        if [[ "$file" == *debug.log ]]; then
            log_files+=("$file")
        fi
    done < <(find "$PROJECT_ROOT" -type f -name "*debug.log" -print0)

    # Look for audit logs
    while IFS= read -r -d '' file; do
        if [[ "$file" == *audit.log ]]; then
            log_files+=("$file")
        fi
    done < <(find "$PROJECT_ROOT" -type f -name "*audit.log" -print0)

    printf '%s\n' "${log_files[@]}"
}

publish_log() {
    local log_file="$1"
    local log_name="$2"

    echo "=== Publishing $log_name ==="
    echo "File: $log_file"
    echo "Size: $(du -h "$log_file" | cut -f1)"
    echo "Last modified: $(stat -f "%Sm" "$log_file")"
    echo ""

    # Show the beginning of the log
    echo "=== Log Preview (first 50 lines) ==="
    head -n 50 "$log_file" || echo "Log file is empty or cannot be read"
    echo ""

    # Show summary
    echo "=== Log Summary ==="
    echo "Total lines: $(wc -l <"$log_file")"
    echo "Debug messages: $(grep -c "DEBUG:" "$log_file" 2>/dev/null || echo "0")"
    echo "Errors: $(grep -c -i "error\|fail\|failed" "$log_file" 2>/dev/null || echo "0")"
    echo ""
}

view_log() {
    local log_file="$1"
    local log_name="$2"

    echo "=== Viewing $log_name ==="
    echo "Press 'q' to quit, 'g' to go to line, 'G' to go to end, '/pattern' to search"
    echo ""

    # Use less with syntax highlighting if available
    if command -v highlight &>/dev/null; then
        highlight -O ansi "$log_file" | less -R
    else
        less "$log_file"
    fi
}

main() {
    local action="${1:-publish}"
    local log_file="${2:-}"

    case "$action" in
    "publish")
        local -a log_files=()
        mapfile -t log_files < <(find_log_files)

        if [[ ${#log_files[@]} -eq 0 ]]; then
            echo "No log files found. Run 'make ci' to generate CI logs."
            exit 1
        fi

        echo "Found ${#log_files[@]} log files:"
        for i in "${!log_files[@]}"; do
            local file="${log_files[$i]}"
            local name="Log $((i + 1))"
            echo "$((i + 1)). $name: $file"
        done

        if [[ -z "$log_file" ]]; then
            echo ""
            read -r -p "Enter number to publish (1-${#log_files[@]}), or press Enter to publish all: " choice

            if [[ -z "$choice" ]]; then
                for file in "${log_files[@]}"; do
                    publish_log "$file" "$(basename "$file")"
                done
            else
                local selected_file="${log_files[$((choice - 1))]}"
                publish_log "$selected_file" "$(basename "$selected_file")"
            fi
        else
            publish_log "$log_file" "$(basename "$log_file")"
        fi
        ;;

    "view")
        local -a log_files=()
        mapfile -t log_files < <(find_log_files)

        if [[ ${#log_files[@]} -eq 0 ]]; then
            echo "No log files found. Run 'make ci' to generate CI logs."
            exit 1
        fi

        if [[ -z "$log_file" ]]; then
            echo "Found ${#log_files[@]} log files:"
            for i in "${!log_files[@]}"; do
                echo "$((i + 1)). $(basename "${log_files[$i]}")"
            done

            echo ""
            read -r -p "Enter number to view (1-${#log_files[@]}): " choice

            if [[ -z "$choice" ]]; then
                echo "No selection made."
                exit 1
            fi

            local selected_file="${log_files[$((choice - 1))]}"
            view_log "$selected_file" "$(basename "$selected_file")"
        else
            if [[ ! -f "$log_file" ]]; then
                echo "Log file not found: $log_file"
                exit 1
            fi
            view_log "$log_file" "$(basename "$log_file")"
        fi
        ;;

    "clean")
        local -a log_files=()
        mapfile -t log_files < <(find_log_files)

        if [[ ${#log_files[@]} -eq 0 ]]; then
            echo "No log files found to clean."
            exit 0
        fi

        echo "Found ${#log_files[@]} log files to clean:"
        for file in "${log_files[@]}"; do
            echo "  - $file"
        done

        read -r -p "Are you sure you want to delete these files? (y/N): " confirm
        if [[ "$confirm" =~ ^[Yy]$ ]]; then
            for file in "${log_files[@]}"; do
                rm -f "$file"
                echo "Deleted: $file"
            done
            echo "Log files cleaned successfully."
        else
            echo "Cleanup cancelled."
        fi
        ;;

    *)
        echo "Usage: $0 [publish|view|clean] [log_file]"
        echo ""
        echo "Commands:"
        echo "  publish  - Publish CI log files (default)"
        echo "  view     - View a log file with less"
        echo "  clean    - Clean all log files"
        echo ""
        echo "Examples:"
        echo "  $0 publish"
        echo "  $0 view ci-debug.log"
        echo "  $0 clean"
        exit 1
        ;;
    esac
}

main "$@"
