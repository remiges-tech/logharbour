# DLQ Manual Testing Guide

## Prerequisites

Start the infrastructure:
```bash
docker-compose up -d elasticsearch kafka zookeeper
```

## Test 1: Validation Error - Invalid JSON

Send invalid JSON to Kafka (will fail unmarshaling):

```bash
# Produce invalid JSON message
echo "this is not valid json" | docker exec -i kafka kafka-console-producer.sh \
  --broker-list localhost:9092 \
  --topic log_topic
```

## Test 2: Validation Error - Missing ID

Send valid JSON but missing required 'id' field:

```bash
echo '{"app":"test","message":"no id field"}' | docker exec -i kafka kafka-console-producer.sh \
  --broker-list localhost:9092 \
  --topic log_topic
```

## Test 3: Valid Message

Send a valid log message:

```bash
echo '{"id":"test-123","app":"test","type":"A","pri":"Info","when":"2024-01-15T10:00:00Z","msg":"test message"}' | \
  docker exec -i kafka kafka-console-producer.sh \
  --broker-list localhost:9092 \
  --topic log_topic
```

## Running the Consumer with DLQ Enabled

```bash
# Build the consumer
go build -o logConsumer ./cmd/logConsumer

# Run with DLQ enabled
./logConsumer \
  -esAddresses=http://localhost:9200 \
  -kafkaBrokers=localhost:9092 \
  -kafkaTopic=log_topic \
  -dlqEnabled=true \
  -dlqTopic=log_topic_dlq \
  -logLevel=debug
```

## Verify DLQ Messages

Check messages in the DLQ topic:

```bash
docker exec -it kafka kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic log_topic_dlq \
  --from-beginning \
  --property print.headers=true
```

## Expected Results

1. **Invalid JSON**: Should appear in DLQ with header `dlq_reason: json_unmarshal_error: ...`
2. **Missing ID**: Should appear in DLQ with header `dlq_reason: missing_id_field`
3. **Valid message**: Should be indexed in Elasticsearch, NOT in DLQ

## Verify Elasticsearch

```bash
curl -s "http://localhost:9200/logs/_search?pretty" | jq '.hits.hits[]._source'
```

## Clean Up

```bash
docker-compose down -v
```
