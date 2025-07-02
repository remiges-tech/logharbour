#!/bin/bash

# Generate test data for LogHarbour cleanup testing
# Creates logs with different ages to test retention policies

ES_URL="${ES_URL:-http://localhost:9200}"
ES_INDEX="${ES_INDEX:-logharbour}"

# Arrays for test data variety
APPS=("auth-service" "user-api" "payment-gateway" "notification-service" "reporting-engine")
SYSTEMS=("prod-server-1" "prod-server-2" "staging-server" "dev-server")
MODULES=("login" "signup" "payment" "notification" "report" "export" "import")
TYPES=("A" "C" "D")  # Activity, Change, Debug
PRIORITIES=("Debug2" "Debug1" "Debug0" "Info" "Warn" "Err" "Crit" "Sec")
USERS=("user123" "admin456" "test789" "service001" "bot002")
IPS=("192.168.1.100" "10.0.0.50" "172.16.0.1" "192.168.2.200" "10.1.1.1")
OPS=("CREATE" "UPDATE" "DELETE" "READ" "LOGIN" "LOGOUT")
CLASSES=("User" "Order" "Payment" "Config" "Session")
STATUSES=(0 1)  # 0=failure, 1=success

# Function to generate a single log entry
generate_log() {
    local date=$1
    local app=${APPS[$RANDOM % ${#APPS[@]}]}
    local system=${SYSTEMS[$RANDOM % ${#SYSTEMS[@]}]}
    local module=${MODULES[$RANDOM % ${#MODULES[@]}]}
    local type=${TYPES[$RANDOM % ${#TYPES[@]}]}
    local priority=${PRIORITIES[$RANDOM % ${#PRIORITIES[@]}]}
    local user=${USERS[$RANDOM % ${#USERS[@]}]}
    local ip=${IPS[$RANDOM % ${#IPS[@]}]}
    local op=${OPS[$RANDOM % ${#OPS[@]}]}
    local class=${CLASSES[$RANDOM % ${#CLASSES[@]}]}
    local status=${STATUSES[$RANDOM % ${#STATUSES[@]}]}
    local instance_id=$((RANDOM % 10000))
    local id=$(uuidgen | tr -d '-')
    
    # Build data field based on type
    local data_json="{}"
    case $type in
        "A")  # Activity
            data_json="{\"activity_data\": \"User $user performed $op on $class\"}"
            ;;
        "C")  # Change
            data_json="{\"change_data\": {\"entity\": \"$class\", \"op\": \"$op\", \"changes\": [{\"field\": \"name\", \"old_value\": \"old_value\", \"new_value\": \"new_value\"}]}}"
            ;;
        "D")  # Debug
            data_json="{\"debug_data\": {\"pid\": $((RANDOM % 10000)), \"runtime\": \"go1.21\", \"file\": \"main.go\", \"line\": $((RANDOM % 1000)), \"func\": \"TestFunction\"}}"
            ;;
    esac
    
    # Build error field if status is failure
    local error_field=""
    if [ $status -eq 0 ]; then
        error_field=", \"error\": \"Operation failed: simulated error for testing\""
    fi
    
    cat << EOF
{"index": {"_index": "$ES_INDEX", "_id": "$id"}}
{"id": "$id", "app": "$app", "system": "$system", "module": "$module", "type": "$type", "pri": "$priority", "when": "$date", "who": "$user", "op": "$op", "class": "$class", "instance": "$instance_id", "status": $status$error_field, "remote_ip": "$ip", "msg": "$op operation on $class by $user", "data": $data_json}
EOF
}

# Function to generate logs for a specific date
generate_logs_for_date() {
    local count=$1
    local days_ago=$2
    local date=$(date -u -d "$days_ago days ago" '+%Y-%m-%dT%H:%M:%SZ')
    
    echo "Generating $count logs from $days_ago days ago ($date)..."
    
    local bulk_data=""
    for i in $(seq 1 $count); do
        # Vary the time within the day
        local hours=$((RANDOM % 24))
        local minutes=$((RANDOM % 60))
        local seconds=$((RANDOM % 60))
        local log_date=$(date -u -d "$days_ago days ago + $hours hours + $minutes minutes + $seconds seconds" '+%Y-%m-%dT%H:%M:%SZ')
        
        bulk_data+=$(generate_log "$log_date")
        bulk_data+=$'\n'
        
        # Send batch every 50 logs
        if [ $((i % 50)) -eq 0 ]; then
            echo "$bulk_data" | curl -s -X POST "$ES_URL/_bulk" \
                -H 'Content-Type: application/x-ndjson' \
                --data-binary @- > /dev/null
            bulk_data=""
            echo -n "."
        fi
    done
    
    # Send remaining logs
    if [ -n "$bulk_data" ]; then
        echo "$bulk_data" | curl -s -X POST "$ES_URL/_bulk" \
            -H 'Content-Type: application/x-ndjson' \
            --data-binary @- > /dev/null
    fi
    
    echo " Done!"
}

# Main execution
echo "LogHarbour Test Data Generator"
echo "=============================="
echo "Elasticsearch URL: $ES_URL"
echo "Index: $ES_INDEX"
echo ""

# Check if Elasticsearch is available
if ! curl -s "$ES_URL" > /dev/null; then
    echo "Error: Cannot connect to Elasticsearch at $ES_URL"
    exit 1
fi

# Check if index exists
if ! curl -s -f "$ES_URL/$ES_INDEX" > /dev/null 2>&1; then
    echo "Creating index..."
    # Create index with mappings
    curl -s -X PUT "$ES_URL/$ES_INDEX" \
        -H 'Content-Type: application/json' \
        -d '{
          "settings": {
            "number_of_shards": 1,
            "number_of_replicas": 0
          },
          "mappings": {
            "properties": {
              "id": {"type": "keyword"},
              "app": {"type": "keyword"},
              "system": {"type": "keyword"},
              "module": {"type": "keyword"},
              "type": {"type": "keyword"},
              "pri": {"type": "keyword"},
              "when": {"type": "date"},
              "who": {"type": "keyword"},
              "op": {"type": "keyword"},
              "class": {"type": "keyword"},
              "instance": {"type": "keyword"},
              "status": {"type": "integer"},
              "error": {"type": "text"},
              "remote_ip": {"type": "ip"},
              "msg": {"type": "text"},
              "data": {
                "properties": {
                  "change_data": {
                    "properties": {
                      "entity": {"type": "keyword"},
                      "op": {"type": "keyword"},
                      "changes": {
                        "type": "nested",
                        "properties": {
                          "field": {"type": "keyword"},
                          "old_value": {"type": "text"},
                          "new_value": {"type": "text"}
                        }
                      }
                    }
                  },
                  "activity_data": {"type": "text"},
                  "debug_data": {
                    "properties": {
                      "pid": {"type": "integer"},
                      "runtime": {"type": "keyword"},
                      "file": {"type": "keyword"},
                      "line": {"type": "integer"},
                      "func": {"type": "keyword"},
                      "stackTrace": {"type": "text"},
                      "data": {"type": "object", "enabled": false}
                    }
                  }
                }
              }
            }
          }
        }' > /dev/null
    echo "Index created"
fi

# Generate test data
echo ""
echo "Generating test data..."

# Today's logs
generate_logs_for_date 100 0

# 5 days old logs
generate_logs_for_date 100 5

# 10 days old logs
generate_logs_for_date 100 10

# 35 days old logs (older than default 30-day retention)
generate_logs_for_date 100 35

# Force refresh to make documents searchable immediately
curl -s -X POST "$ES_URL/$ES_INDEX/_refresh" > /dev/null

echo ""
echo "Test data generation complete!"
echo ""

# Show summary
total_count=$(curl -s "$ES_URL/$ES_INDEX/_count" | grep -o '"count":[0-9]*' | cut -d':' -f2)
echo "Total documents in index: $total_count"
echo ""
echo "Distribution by age:"
echo "- Today: ~100 logs"
echo "- 5 days ago: ~100 logs"
echo "- 10 days ago: ~100 logs"
echo "- 35 days ago: ~100 logs"
echo ""
echo "You can now test the cleanup script with:"
echo "  ./logharbour-cleanup.sh --dry-run              # Preview deletion (30+ days old)"
echo "  ./logharbour-cleanup.sh --days 7 --dry-run     # Preview deletion (7+ days old)"
echo "  ./logharbour-cleanup.sh --days 7               # Actually delete logs 7+ days old"