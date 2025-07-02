# LogHarbour Log Retention Tool

This tool provides a shell script for cleaning up old logs from LogHarbour's Elasticsearch index based on configurable retention policies.

## Features

- **Time-based deletion**: Delete logs older than specified days
- **Flexible filtering**: Filter by application, type, or priority
- **Dry run mode**: Preview what will be deleted before actual deletion
- **Authentication support**: Works with secured Elasticsearch clusters
- **Batch processing**: Efficient deletion of large volumes of logs
- **Safety features**: Confirmation prompts for large deletions
- **Verbose mode**: Detailed output for debugging

## Quick Start

### 1. Basic Testing with Docker Compose

```bash
# Start Elasticsearch
cd testing
docker-compose up -d

# Wait for Elasticsearch to be ready (about 30 seconds)
sleep 30

# Generate test data
./populate-test-data.sh

# Check the data
cd ..
./logharbour-cleanup.sh --dry-run

# Delete logs older than 7 days
./logharbour-cleanup.sh --days 7

# Delete specific types of logs
./logharbour-cleanup.sh --days 1 --type D --dry-run
```

### 2. Usage Examples

```bash
# Show help
./logharbour-cleanup.sh --help

# Dry run - see what would be deleted (default: 30 days)
./logharbour-cleanup.sh --dry-run

# Delete logs older than 7 days
./logharbour-cleanup.sh --days 7

# Delete logs for specific application
./logharbour-cleanup.sh --days 30 --app auth-service

# Delete debug logs older than 1 day
./logharbour-cleanup.sh --days 1 --type D

# Delete with multiple filters
./logharbour-cleanup.sh --days 7 --app auth-service --priority Err

# Verbose output
./logharbour-cleanup.sh --days 30 --verbose --dry-run

# Custom Elasticsearch URL
./logharbour-cleanup.sh --url http://my-es-cluster:9200 --index my-logs --days 90
```

## Testing Guide

### Step 1: Start the Test Environment

```bash
cd tools/log-retention/testing
docker-compose up -d
```

This starts a local Elasticsearch instance on port 9200.

### Step 2: Create Test Data

```bash
# Make scripts executable
chmod +x *.sh

# Generate test data with various dates
./populate-test-data.sh
```

This creates:
- 100 logs from today
- 100 logs from 5 days ago
- 100 logs from 10 days ago
- 100 logs from 35 days ago
- Logs with different types, priorities, and applications

### Step 3: Verify Test Data

```bash
# Count all documents
curl -s http://localhost:9200/logharbour/_count | jq

# See document distribution by date
./inspect-test-data.sh
```

### Step 4: Test Dry Run

```bash
# Go back to parent directory
cd ..

# See what would be deleted (older than 30 days)
./logharbour-cleanup.sh --dry-run

# See what would be deleted (older than 7 days)
./logharbour-cleanup.sh --days 7 --dry-run

# Test with filters
./logharbour-cleanup.sh --days 7 --app auth-service --dry-run
```

### Step 5: Perform Actual Deletion

```bash
# Delete logs older than 7 days
./logharbour-cleanup.sh --days 7

# Verify deletion
curl -s http://localhost:9200/logharbour/_count | jq
```

### Step 6: Clean Up

```bash
# Stop and remove containers
cd testing
docker-compose down

# Remove volumes (optional)
docker-compose down -v
```

## Script Options

| Option | Description | Default |
|--------|-------------|---------|
| `-h, --help` | Show help message | - |
| `-d, --days DAYS` | Retention period in days | 30 |
| `-a, --app APP` | Filter by application | - |
| `-t, --type TYPE` | Filter by log type | - |
| `-p, --priority PRIORITY` | Filter by priority | - |
| `-n, --dry-run` | Preview without deleting | false |
| `-v, --verbose` | Enable verbose output | false |
| `--url URL` | Elasticsearch URL | http://localhost:9200 |
| `--index INDEX` | Index name | logharbour |
| `--username USERNAME` | ES username | - |
| `--password PASSWORD` | ES password | - |
| `--batch-size SIZE` | Batch size for deletion | 1000 |

## Environment Variables

The script supports configuration via environment variables:

```bash
export ES_URL="http://localhost:9200"
export ES_INDEX="logharbour"
export RETENTION_DAYS="30"
export DRY_RUN="true"
export VERBOSE="true"

# For authenticated clusters
export ES_USERNAME="elastic"
export ES_PASSWORD="mypassword"
```

## Scheduling with Cron

To run the cleanup automatically:

```bash
# Edit crontab
crontab -e

# Run daily at 2 AM, delete logs older than 30 days
0 2 * * * /path/to/logharbour-cleanup.sh --days 30 >> /var/log/logharbour-cleanup.log 2>&1

# Run weekly on Sunday, delete debug logs older than 7 days
0 3 * * 0 /path/to/logharbour-cleanup.sh --days 7 --type D >> /var/log/logharbour-cleanup.log 2>&1
```

## Safety Features

1. **Dry Run Mode**: Always test with `--dry-run` first
2. **Document Count**: Shows number of documents to be deleted
3. **Sample Documents**: Displays 5 sample documents before deletion
4. **Large Deletion Confirmation**: Prompts for confirmation when deleting >10,000 documents
5. **Error Handling**: Stops on any error to prevent partial deletions

## Production Usage

Example production command:
```bash
./logharbour-cleanup.sh \
  --url https://es-cluster.example.com:9200 \
  --username elastic \
  --password $ES_PASSWORD \
  --days 90 \
  --app payment-gateway \
  --batch-size 5000
```