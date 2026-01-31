#!/usr/bin/env bash
# Display formatters for the show command

format_items() {
    local items="$1"
    local count
    count=$(count_items "$items")
    
    echo "=== Intray Items (${count}) ==="
    echo "$items" | nl -w2 -s'. ' | sed 's/^[[:space:]]*//'
}

format_table() {
    local items="$1"
    
    echo "=== Intray Items ==="
    printf "%-4s %-20s %s\n" "#" "Time" "Message"
    printf "%-4s %-20s %s\n" "----" "--------------------" "--------------------"
    
    echo "$items" | while IFS=']' read -r timestamp message; do
        # Extract just the time from timestamp
        local time
        time=$(echo "$timestamp" | grep -oE '[0-9]{2}:[0-9]{2}:[0-9]{2}' || echo "??")
        
        # Clean up message
        message=$(echo "$message" | sed 's/^\[//' | xargs)
        
        printf "%-4s %-20s %s\n" "#" "$time" "$message"
    done
}

format_compact() {
    local items="$1"
    local count
    count=$(count_items "$items")
    
    echo "Tray: ${count} items"
}

format_json() {
    local items="$1"
    local count
    count=$(count_items "$items")
    
    echo '{'
    echo '  "count": '"$count"','
    echo '  "items": ['
    
    local first=true
    echo "$items" | while IFS= read -r item; do
        if [[ -n "$item" && -n "$first" ]]; then
            first=false
        elif [[ -n "$item" ]]; then
            echo ','
        fi
        
        if [[ -n "$item" ]]; then
            local timestamp
            local message
            timestamp=$(echo "$item" | grep -oE '\[[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}\]' | tr -d '[]')
            message=$(echo "$item" | sed 's/^\[[^]]*\] //' | sed 's/^\[[^]]*\]\s*//')
            
            printf '    {"timestamp": "%s", "message": "%s"}' "$timestamp" "$message"
        fi
    done
    
    echo ''
    echo '  ]'
    echo '}'
}
