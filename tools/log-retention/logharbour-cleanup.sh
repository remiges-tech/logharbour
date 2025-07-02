#!/bin/bash

# LogHarbour Log Cleanup Script
# This script deletes old logs from Elasticsearch based on retention policies

set -e

# Default configuration
ES_URL="${ES_URL:-http://localhost:9200}"
ES_INDEX="${ES_INDEX:-logharbour}"
ES_USERNAME="${ES_USERNAME:-}"
ES_PASSWORD="${ES_PASSWORD:-}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"
DRY_RUN="${DRY_RUN:-false}"
APP="${APP:-}"
TYPE="${TYPE:-}"
PRIORITY="${PRIORITY:-}"
BATCH_SIZE="${BATCH_SIZE:-1000}"
VERBOSE="${VERBOSE:-false}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to build curl authentication
build_auth() {
    local auth=""
    if [ -n "$ES_USERNAME" ] && [ -n "$ES_PASSWORD" ]; then
        auth="-u ${ES_USERNAME}:${ES_PASSWORD}"
    fi
    echo "$auth"
}

# Function to check Elasticsearch connectivity
check_elasticsearch() {
    log_info "Checking Elasticsearch connectivity..."
    local auth=$(build_auth)
    
    if ! curl -s $auth "$ES_URL" > /dev/null; then
        log_error "Cannot connect to Elasticsearch at $ES_URL"
        exit 1
    fi
    
    log_info "Successfully connected to Elasticsearch"
}

# Function to build the query
build_query() {
    local query='{"bool": {"must": [{"range": {"when": {"lt": "now-'$RETENTION_DAYS'd"}}}'
    
    # Add optional filters
    if [ -n "$APP" ]; then
        query+=',{"term": {"app": "'$APP'"}}'
    fi
    
    if [ -n "$TYPE" ]; then
        query+=',{"term": {"type": "'$TYPE'"}}'
    fi
    
    if [ -n "$PRIORITY" ]; then
        query+=',{"term": {"pri": "'$PRIORITY'"}}'
    fi
    
    query+=']}}'
    
    echo '{"query": '$query'}'
}

# Function to count documents to be deleted
count_documents() {
    local auth=$(build_auth)
    local query=$(build_query)
    
    if [ "$VERBOSE" = "true" ]; then
        log_info "Query: $query"
    fi
    
    local response=$(curl -s $auth -X GET "$ES_URL/$ES_INDEX/_count" \
        -H 'Content-Type: application/json' \
        -d "$query")
    
    local count=$(echo "$response" | grep -o '"count":[0-9]*' | cut -d':' -f2)
    
    if [ -z "$count" ]; then
        log_error "Failed to count documents. Response: $response"
        exit 1
    fi
    
    echo "$count"
}

# Function to get sample documents
get_sample_documents() {
    local auth=$(build_auth)
    local query=$(build_query | jq '. + {"size": 5, "_source": ["when", "app", "type", "pri", "msg"]}')
    
    log_info "Sample documents to be deleted:"
    
    local response=$(curl -s $auth -X GET "$ES_URL/$ES_INDEX/_search" \
        -H 'Content-Type: application/json' \
        -d "$query")
    
    echo "$response" | jq -r '.hits.hits[] | ._source | "  \(.when) | app: \(.app) | type: \(.type) | priority: \(.pri) | msg: \(.msg[0:50])..."' 2>/dev/null || echo "  Unable to fetch samples"
}

# Function to delete documents
delete_documents() {
    local auth=$(build_auth)
    local query=$(build_query)
    
    # Add batch size and conflicts handling
    local delete_query=$(echo "$query" | jq '. + {"conflicts": "proceed"}')
    
    log_info "Starting deletion process..."
    
    local response=$(curl -s $auth -X POST "$ES_URL/$ES_INDEX/_delete_by_query?slices=auto&scroll_size=$BATCH_SIZE" \
        -H 'Content-Type: application/json' \
        -d "$delete_query")
    
    local deleted=$(echo "$response" | grep -o '"deleted":[0-9]*' | cut -d':' -f2)
    local took=$(echo "$response" | grep -o '"took":[0-9]*' | cut -d':' -f2)
    local failures=$(echo "$response" | grep -o '"failures":\[[^]]*\]')
    
    if [ -n "$deleted" ]; then
        log_info "Deleted $deleted documents in ${took}ms"
        
        if [ -n "$failures" ] && [ "$failures" != '"failures":[]' ]; then
            log_warn "Some failures occurred during deletion"
            if [ "$VERBOSE" = "true" ]; then
                echo "$response" | jq '.failures'
            fi
        fi
    else
        log_error "Deletion failed. Response: $response"
        exit 1
    fi
}

# Function to show usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Delete old logs from LogHarbour Elasticsearch index based on retention policies.

OPTIONS:
    -h, --help              Show this help message
    -d, --days DAYS         Retention period in days (default: 30)
    -a, --app APP           Filter by application
    -t, --type TYPE         Filter by log type
    -p, --priority PRIORITY Filter by priority
    -n, --dry-run           Show what would be deleted without actually deleting
    -v, --verbose           Enable verbose output
    --url URL               Elasticsearch URL (default: http://localhost:9200)
    --index INDEX           Elasticsearch index (default: logharbour)
    --username USERNAME     Elasticsearch username
    --password PASSWORD     Elasticsearch password
    --batch-size SIZE       Batch size for deletion (default: 1000)

EXAMPLES:
    # Delete all logs older than 30 days (dry run)
    $0 --dry-run

    # Delete logs older than 7 days
    $0 --days 7

    # Delete debug logs older than 1 day
    $0 --days 1 --type D --priority Debug0

    # Delete with authentication
    $0 --username elastic --password mypassword --days 90

ENVIRONMENT VARIABLES:
    ES_URL              Elasticsearch URL
    ES_INDEX            Elasticsearch index name
    ES_USERNAME         Elasticsearch username
    ES_PASSWORD         Elasticsearch password
    RETENTION_DAYS      Default retention period
    DRY_RUN             Set to 'true' for dry run
    VERBOSE             Set to 'true' for verbose output
EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            exit 0
            ;;
        -d|--days)
            RETENTION_DAYS="$2"
            shift 2
            ;;
        -a|--app)
            APP="$2"
            shift 2
            ;;
        -t|--type)
            TYPE="$2"
            shift 2
            ;;
        -p|--priority)
            PRIORITY="$2"
            shift 2
            ;;
        -n|--dry-run)
            DRY_RUN="true"
            shift
            ;;
        -v|--verbose)
            VERBOSE="true"
            shift
            ;;
        --url)
            ES_URL="$2"
            shift 2
            ;;
        --index)
            ES_INDEX="$2"
            shift 2
            ;;
        --username)
            ES_USERNAME="$2"
            shift 2
            ;;
        --password)
            ES_PASSWORD="$2"
            shift 2
            ;;
        --batch-size)
            BATCH_SIZE="$2"
            shift 2
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Main execution
log_info "LogHarbour Cleanup Script"
log_info "========================"
log_info "Elasticsearch URL: $ES_URL"
log_info "Index: $ES_INDEX"
log_info "Retention: $RETENTION_DAYS days"

if [ -n "$APP" ]; then log_info "App filter: $APP"; fi
if [ -n "$TYPE" ]; then log_info "Type filter: $TYPE"; fi
if [ -n "$PRIORITY" ]; then log_info "Priority filter: $PRIORITY"; fi

echo ""

# Check connectivity
check_elasticsearch

# Count documents
log_info "Counting documents older than $RETENTION_DAYS days..."
count=$(count_documents)

if [ "$count" -eq 0 ]; then
    log_info "No documents found matching the criteria"
    exit 0
fi

log_warn "Found $count documents to be deleted"
echo ""

# Show sample documents
get_sample_documents
echo ""

# Dry run or actual deletion
if [ "$DRY_RUN" = "true" ]; then
    log_warn "DRY RUN MODE - No documents will be deleted"
    log_info "To perform actual deletion, run without --dry-run flag"
else
    # Confirm deletion for large numbers
    if [ "$count" -gt 10000 ]; then
        log_warn "About to delete $count documents. This is a large number!"
        read -p "Are you sure you want to continue? (yes/no): " confirm
        if [ "$confirm" != "yes" ]; then
            log_info "Deletion cancelled"
            exit 0
        fi
    fi
    
    delete_documents
    log_info "Cleanup completed successfully"
fi