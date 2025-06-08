# LogHarbour Consumer

A Kafka consumer service that reads log messages from Kafka topics and writes them to Elasticsearch for storage and indexing.

## Features

- **Dual Consumer Modes**: Regular consumer and consumer group support
- **Configurable Offset Management**: Start from earliest, latest, or specific offset
- **Batch Processing**: Configurable batch sizes for optimal performance
- **Fault Tolerance**: Automatic retry with exponential backoff for failed operations
- **Load Balancing**: Consumer groups automatically distribute load across multiple instances
- **Offset Persistence**: Consumer groups automatically persist and resume from last committed offset

## Consumer Modes

### Consumer Group (Default)
- **Default mode** - automatically uses consumer group `logharbour-consumer-group`
- Multiple consumers can join the same group for load balancing
- Automatic partition rebalancing when consumers join/leave
- Offset persistence ensures no message loss on restart
- Fault tolerance with automatic failover

### Regular Consumer
- Directly consumes from all partitions of a topic
- Manual partition assignment
- Suitable for single-instance processing
- To use: set `--useConsumerGroup=false` or `USE_CONSUMER_GROUP=false`

## Configuration

The consumer can be configured using command-line flags or environment variables:

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--esAddresses` | `ELASTICSEARCH_ADDRESSES` | `http://localhost:9200` | Elasticsearch addresses (comma-separated) |
| `--kafkaBrokers` | `KAFKA_BROKERS` | `localhost:9092` | Kafka brokers (comma-separated) |
| `--kafkaTopic` | `KAFKA_TOPIC` | `log_topic` | Kafka topic to consume from |
| `--esIndex` | `ELASTICSEARCH_INDEX` | `logs` | Elasticsearch index name |
| `--offsetType` | `KAFKA_OFFSET_TYPE` | `earliest` | Offset type: `earliest`, `latest`, or specific number |
| `--batchSize` | `KAFKA_BATCH_SIZE` | `10` | Number of messages to process in each batch |
| `--consumerGroup` | `KAFKA_CONSUMER_GROUP` | `logharbour-consumer-group` | Consumer group ID |
| `--useConsumerGroup` | `USE_CONSUMER_GROUP` | `true` | Enable consumer group mode |
| `--logLevel` | `LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |

## Usage

### Consumer Group (Default Mode)

```bash
# Basic usage - automatically uses consumer group "logharbour-consumer-group"
./lh-consumer

# With custom group ID
./lh-consumer --consumerGroup="my-custom-group"

# Multiple consumers in the same group (run in separate instances)
# They will automatically load balance partitions
./lh-consumer  # Instance 1
./lh-consumer  # Instance 2
./lh-consumer  # Instance 3
```

### Regular Consumer (Legacy Mode)

```bash
# Disable consumer group to use regular consumer
./lh-consumer --useConsumerGroup=false

# With custom configuration
./lh-consumer \
  --useConsumerGroup=false \
  --esAddresses="http://es1:9200,http://es2:9200" \
  --kafkaBrokers="kafka1:9092,kafka2:9092" \
  --kafkaTopic="application_logs" \
  --batchSize=50 \
  --offsetType="latest"
```

### Using Environment Variables

```bash
# Set environment variables
export ELASTICSEARCH_ADDRESSES="http://localhost:9200"
export KAFKA_BROKERS="localhost:9092"
export KAFKA_TOPIC="log_topic"
export KAFKA_CONSUMER_GROUP="my-log-group"  # Override default group ID
export KAFKA_OFFSET_TYPE="earliest"
export KAFKA_BATCH_SIZE="50"

# Run consumer (uses consumer group by default)
./lh-consumer

# To disable consumer groups
export USE_CONSUMER_GROUP="false"
./lh-consumer
```

## Docker Usage

### Build
```bash
# Build consumer image
make docker_build_consumer
```

### Run with Docker

```bash
# Default consumer group mode
docker run --rm \
  -e ELASTICSEARCH_ADDRESSES="http://elasticsearch:9200" \
  -e KAFKA_BROKERS="kafka:9092" \
  -e KAFKA_TOPIC="log_topic" \
  -e KAFKA_OFFSET_TYPE="earliest" \
  -e KAFKA_BATCH_SIZE="25" \
  lhconsumer:latest

# Custom consumer group
docker run --rm \
  -e ELASTICSEARCH_ADDRESSES="http://elasticsearch:9200" \
  -e KAFKA_BROKERS="kafka:9092" \
  -e KAFKA_TOPIC="log_topic" \
  -e KAFKA_CONSUMER_GROUP="docker-log-group" \
  -e KAFKA_OFFSET_TYPE="earliest" \
  -e KAFKA_BATCH_SIZE="50" \
  lhconsumer:latest

# Regular consumer (disable consumer group)
docker run --rm \
  -e ELASTICSEARCH_ADDRESSES="http://elasticsearch:9200" \
  -e KAFKA_BROKERS="kafka:9092" \
  -e KAFKA_TOPIC="log_topic" \
  -e USE_CONSUMER_GROUP="false" \
  -e KAFKA_OFFSET_TYPE="earliest" \
  -e KAFKA_BATCH_SIZE="25" \
  lhconsumer:latest
```

### Docker Compose
```yaml
version: '3'
services:
  lhconsumer1:
    image: lhconsumer:latest
    environment:
      ELASTICSEARCH_ADDRESSES: "http://elasticsearch:9200"
      KAFKA_BROKERS: "kafka:9092"
      KAFKA_TOPIC: "log_topic"
      USE_CONSUMER_GROUP: "true"
      KAFKA_CONSUMER_GROUP: "log-processors"
      KAFKA_OFFSET_TYPE: "earliest"
      KAFKA_BATCH_SIZE: "50"
    depends_on:
      - kafka
      - elasticsearch

  lhconsumer2:
    image: lhconsumer:latest
    environment:
      ELASTICSEARCH_ADDRESSES: "http://elasticsearch:9200"
      KAFKA_BROKERS: "kafka:9092"
      KAFKA_TOPIC: "log_topic"
      USE_CONSUMER_GROUP: "true"
      KAFKA_CONSUMER_GROUP: "log-processors"
      KAFKA_OFFSET_TYPE: "earliest"
      KAFKA_BATCH_SIZE: "50"
    depends_on:
      - kafka
      - elasticsearch
```

## Offset Types

### earliest
- Starts consuming from the oldest available message in the topic
- Useful for processing all historical data
- Default behavior for new consumer groups

### latest
- Starts consuming only new messages after the consumer starts
- Skips all existing messages in the topic
- Useful for real-time processing only

### Specific Offset
- Starts consuming from a specific message offset
- Provide a numeric offset value (e.g., `--offsetType="1000"`)
- Useful for replaying from a known point

## Consumer Groups

Consumer groups provide several advantages:

### Load Balancing
- Multiple consumers in the same group automatically share the workload
- Kafka distributes partitions among group members
- Adding more consumers increases processing capacity

### Fault Tolerance
- If a consumer fails, its partitions are reassigned to other group members
- No message loss occurs during consumer failures
- Automatic recovery when failed consumers restart

### Offset Management
- Group offsets are automatically persisted to Kafka
- Consumers resume exactly where they left off after restart
- No duplicate processing or message loss

### Rebalancing
- Adding/removing consumers triggers automatic rebalancing
- Partitions are redistributed evenly among active consumers
- Minimal processing interruption during rebalancing

## Monitoring

### Check Consumer Group Status
```bash
# List all consumer groups
kafka-consumer-groups.sh --bootstrap-server localhost:9092 --list

# Describe specific group
kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --describe --group log-processors
```

### Monitor Lag
```bash
# Check consumer lag
kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --describe --group log-processors

# Output shows:
# GROUP           TOPIC     PARTITION  CURRENT-OFFSET  LOG-END-OFFSET  LAG
# log-processors  log_topic 0          1500            1500            0
# log-processors  log_topic 1          1200            1250            50
```

### Reset Offsets (if needed)
```bash
# Reset to earliest
kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group log-processors --reset-offsets --to-earliest \
  --topic log_topic --execute

# Reset to latest
kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group log-processors --reset-offsets --to-latest \
  --topic log_topic --execute
```

## Error Handling

### Retry Mechanism
- Failed Elasticsearch writes are automatically retried up to 10 times
- Uses exponential backoff (1s, 2s, 4s, 8s, ...)
- Prevents overwhelming Elasticsearch during temporary issues

### Batch Processing
- Messages are processed in configurable batches
- Failed batches are retried as a unit
- Prevents partial batch processing

### Logging
- Comprehensive logging for monitoring and debugging
- Error details logged for failed operations
- Consumer group status logged on startup

## Performance Tuning

### Batch Size
- Larger batches improve throughput but increase memory usage
- Smaller batches reduce latency but may decrease throughput
- Recommended: 25-100 messages per batch

### Multiple Consumers
- Add more consumers to the same group for horizontal scaling
- Optimal number: one consumer per topic partition
- Monitor CPU and memory usage when scaling

### Elasticsearch Optimization
- Use bulk indexing for better performance
- Configure appropriate refresh intervals
- Monitor Elasticsearch cluster health

## Troubleshooting

### Consumer Not Processing Messages
1. Check Kafka connectivity: `telnet kafka-host 9092`
2. Verify topic exists: `kafka-topics.sh --list --bootstrap-server kafka:9092`
3. Check consumer group lag: `kafka-consumer-groups.sh --describe --group GROUP_NAME`

### High Lag
1. Increase number of consumers in the group
2. Increase batch size for better throughput
3. Check Elasticsearch performance and scaling

### Connection Issues
1. Verify network connectivity to Kafka and Elasticsearch
2. Check firewall rules and security groups
3. Validate connection strings and ports

### Offset Issues
1. Check consumer group status for partition assignment
2. Verify offset commit frequency
3. Consider resetting offsets if needed

## Development

### Building
```bash
# Build binary
go build -o lh-consumer ./cmd/logConsumer

# Build Docker image
docker build -t lhconsumer:latest .
```

### Testing
```bash
# Run tests
go test ./...

# Integration tests with Testcontainers
go test -v ./logharbour/test/
```


The consumer reads messages from Kafka topics, processes them in configurable batches, and writes them to Elasticsearch indices for storage and searching.