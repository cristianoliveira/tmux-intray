#!/usr/bin/env bash
# Filters for the show command

filter_by_date() {
    local items="$1"
    local date_filter="$2"

    if [[ -z "$date_filter" ]]; then
        echo "$items"
        return
    fi

    echo "$items" | grep "$date_filter"
}

filter_by_source() {
    local items="$1"
    local source_filter="$2"

    if [[ -z "$source_filter" ]]; then
        echo "$items"
        return
    fi

    echo "$items" | grep "\[${source_filter}\]"
}

filter_by_priority() {
    local items="$1"
    local priority="$2"

    if [[ -z "$priority" ]]; then
        echo "$items"
        return
    fi

    local priority_symbol
    case "$priority" in
    high) priority_symbol="🔴" ;;
    normal) priority_symbol="⚪" ;;
    low) priority_symbol="🟢" ;;
    esac

    echo "$items" | grep "$priority_symbol"
}

count_items() {
    local items="$1"

    if [[ -z "$items" ]]; then
        echo "0"
        return
    fi

    echo "$items" | grep -c -v '^$'
}
