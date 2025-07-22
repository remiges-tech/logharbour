# Batch Timeout Demo

This demo shows how the batch timeout mechanism prevents logs from waiting indefinitely in the consumer.

## What It Demonstrates

The consumer has a configurable timeout that forces processing of partial batches. Without this feature, logs would wait forever if the batch size isn't reached.

## Configuration

- **Batch Size**: 100 (intentionally high)
- **Batch Timeout**: 10 seconds
- **Message Rate**: 1 message every 2 seconds
- **Total Messages**: 20

## How to Run

1. **Build the consumer image** (from project root):
   ```bash
   make docker_build_consumer
   ```

2. **Run the demo**:
   ```bash
   cd examples/batch-timeout-demo
   docker compose up
   ```

3. **Watch the output** for these key messages:
   ```
   msg="Batch timeout reached, processing partial batch" batch_size=5 expected_size=100
   ```

## Expected Results

With 20 messages sent over 40 seconds at 2-second intervals:

- **10s**: First batch timeout → 5 messages processed
- **20s**: Second batch timeout → 5 messages processed  
- **30s**: Third batch timeout → 5 messages processed
- **40s**: Fourth batch timeout → 3 messages processed

Each batch is triggered by the timeout, not by reaching the batch size of 100.

## How It Works

1. The producer (`producer.go`) runs inside Docker and sends messages slowly
2. The consumer accumulates messages but won't reach the batch size of 100
3. Every 10 seconds, the timeout forces processing of whatever messages are accumulated
4. This prevents indefinite waiting in low-volume scenarios

## Key Implementation Files

- `/logharbour/kafkaconsumer.go` - Contains the timeout mechanism
- `/cmd/logConsumer/main.go` - Configures the timeout via environment variable

## Configuration Options

- `KAFKA_BATCH_TIMEOUT`: Timeout duration (default: "60s")
- `KAFKA_BATCH_SIZE`: Number of messages per batch (default: 1000)

Valid timeout formats: "10s", "1m", "500ms", "1h30m"