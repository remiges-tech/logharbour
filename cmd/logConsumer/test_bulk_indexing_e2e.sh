#!/bin/bash

# Get the script directory and project root
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/../.." && pwd )"

# Change to project root for all operations
cd "$PROJECT_ROOT"

echo "End-to-End Bulk Indexing Test for LogHarbour Consumer"
echo "===================================================="
echo "This script performs a complete test of the bulk indexing feature:"
echo "- Builds the consumer with bulk indexing support"
echo "- Starts Elasticsearch and Kafka"
echo "- Creates the logharbour index"
echo "- Runs the consumer and producer"
echo "- Verifies bulk indexing is working correctly"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Step 1: Ensure clean state
echo -e "\n${YELLOW}Step 1: Cleaning up existing setup${NC}"
# Stop and remove custom test container if it exists
docker stop logharbour_lhconsumer_test 2>/dev/null || true
docker rm logharbour_lhconsumer_test 2>/dev/null || true
docker-compose down
sleep 2

# Step 2: Build fresh consumer image
echo -e "\n${YELLOW}Step 2: Building consumer with bulk indexing${NC}"
make docker_build_consumer
if [ $? -ne 0 ]; then
    echo -e "${RED}Failed to build consumer!${NC}"
    exit 1
fi

# Step 3: Start only infrastructure
echo -e "\n${YELLOW}Step 3: Starting Elasticsearch and Kafka${NC}"
docker-compose up -d elasticsearch kafka
sleep 20

# Step 4: Verify Elasticsearch
echo -e "\n${YELLOW}Step 4: Verifying Elasticsearch${NC}"
if curl -s http://localhost:9200 > /dev/null; then
    echo -e "${GREEN}Elasticsearch is running${NC}"
else
    echo -e "${RED}Elasticsearch is not reachable!${NC}"
    exit 1
fi

# Step 5: Create index with proper mapping
echo -e "\n${YELLOW}Step 5: Creating logharbour index${NC}"
echo "Deleting any existing index..."
curl -s -X DELETE http://localhost:9200/logharbour 2>/dev/null | jq . 2>/dev/null || echo "No existing index to delete"
echo "Creating new index with mappings..."
RESPONSE=$(curl -s -X PUT http://localhost:9200/logharbour \
  -H 'Content-Type: application/json' \
  -d '{
    "settings": {
      "number_of_shards": 1,
      "number_of_replicas": 0
    },
    "mappings": {
      "properties": {
        "id": {"type": "keyword"},
        "priority_id": {"type": "integer"},
        "priority": {"type": "keyword"},
        "when": {"type": "date"},
        "app": {"type": "keyword"},
        "module": {"type": "keyword"},
        "type_id": {"type": "integer"},
        "type": {"type": "keyword"},
        "msg": {"type": "text"},
        "data": {"type": "object", "enabled": false}
      }
    }
  }')
if echo "$RESPONSE" | jq -e '.acknowledged == true' >/dev/null 2>&1; then
    echo -e "${GREEN}Index created successfully${NC}"
else
    echo -e "${RED}Failed to create index: $RESPONSE${NC}"
    exit 1
fi

# Step 6: Start consumer with batch size 1000
echo -e "\n${YELLOW}Step 6: Starting consumer with batch size 1000${NC}"
# Stop any existing consumer
docker-compose stop lhconsumer 2>/dev/null || true

# Start consumer with batch size 1000
docker-compose run -d --name logharbour_lhconsumer_test \
  -e ELASTICSEARCH_ADDRESSES="http://elasticsearch:9200" \
  -e ELASTICSEARCH_INDEX="logharbour" \
  -e KAFKA_BROKERS="kafka:9092" \
  -e KAFKA_TOPIC="log_topic" \
  -e KAFKA_OFFSET_TYPE="earliest" \
  -e KAFKA_BATCH_SIZE="1000" \
  lhconsumer

echo "Waiting 25 seconds for consumer to start (it has a 20s delay)..."
sleep 25

# Step 7: Check consumer started properly
echo -e "\n${YELLOW}Step 7: Checking consumer startup${NC}"
docker ps | grep logharbour_lhconsumer_test || echo "Consumer container status"
echo -e "\nConsumer logs:"
docker logs --tail=20 logharbour_lhconsumer_test 2>&1 || docker-compose logs --tail=20 lhconsumer
echo -e "\nChecking consumer configuration:"
docker exec -T logharbour_lhconsumer_test env 2>/dev/null | grep KAFKA_BATCH_SIZE || echo "Could not check batch size"

# Step 8: Generate test messages
echo -e "\n${YELLOW}Step 8: Generating 10,000 test messages${NC}"
# Set batch size for consumer to 1000
docker-compose exec -T lhconsumer sh -c 'echo "Consumer will process messages in batches of 1000"' 2>/dev/null || true

# Override the batch size environment variable for this test
export KAFKA_BATCH_SIZE=1000

# Run producer with 10,000 messages using multiple goroutines for faster generation
docker-compose run --rm -e KAFKA_BATCH_SIZE=1000 lhproducer ./lh-producer -nMessages=10000 -nGoroutines=10

# Step 9: Wait and check results
echo -e "\n${YELLOW}Step 9: Waiting for processing 10,000 messages...${NC}"
echo "This may take 30-60 seconds..."
sleep 30

# Step 10: Check consumer logs for bulk operations
echo -e "\n${YELLOW}Step 10: Checking for bulk indexing in logs${NC}"
echo "Last 50 lines of consumer logs:"
docker logs --tail=50 logharbour_lhconsumer_test 2>&1 || docker-compose logs --tail=50 lhconsumer

echo -e "\n${YELLOW}Filtered for bulk operations:${NC}"
docker logs logharbour_lhconsumer_test 2>&1 | grep -E "Batch processing completed|documents_sent|success_count" | tail -20 || echo -e "${RED}No bulk messages found${NC}"

echo -e "\n${YELLOW}Batch statistics:${NC}"
docker logs logharbour_lhconsumer_test 2>&1 | grep "batch_size=1000" | tail -5 || echo "No 1000-message batches found"

# Step 11: Check Elasticsearch
echo -e "\n${YELLOW}Step 11: Checking documents in Elasticsearch${NC}"
COUNT=$(curl -s http://localhost:9200/logharbour/_count | jq -r '.count' 2>/dev/null || echo "0")
echo -e "Documents indexed: ${COUNT}"

if [ "$COUNT" != "0" ] && [ "$COUNT" != "" ]; then
    echo -e "${GREEN}Success! $COUNT documents were indexed.${NC}"
    
    # Calculate performance metrics
    if [ "$COUNT" -eq "10000" ]; then
        echo -e "${GREEN}All 10,000 messages were successfully indexed!${NC}"
    else
        echo -e "${YELLOW}Expected 10,000 documents but found $COUNT${NC}"
    fi
    
    # Show batch processing summary
    echo -e "\n${YELLOW}Batch processing summary:${NC}"
    docker logs logharbour_lhconsumer_test 2>&1 | grep "Batch processing completed" | tail -3
    
    echo -e "\nSample document:"
    curl -s http://localhost:9200/logharbour/_search?size=1 | jq '.hits.hits[0]._source' 2>/dev/null
else
    echo -e "${RED}No documents found!${NC}"
    echo -e "\nTroubleshooting info:"
    echo "- Consumer container: $(docker ps | grep logharbour_lhconsumer_test)"
    echo "- Check full logs: docker logs logharbour_lhconsumer_test"
fi

echo -e "\n${GREEN}Verification complete!${NC}"
echo -e "\nScript run from: $PROJECT_ROOT"