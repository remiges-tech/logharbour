#!/bin/bash
set -e

# Wait for Kafka to be ready
until /opt/bitnami/kafka/bin/kafka-topics.sh --list --bootstrap-server localhost:9092; do
  sleep 1
done

# Create your topic
/opt/bitnami/kafka/bin/kafka-topics.sh --create --bootstrap-server localhost:9092 --replication-factor 1 --partitions 1 --topic log_topic