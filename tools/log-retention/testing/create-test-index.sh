#!/bin/sh

# Setup script for Elasticsearch
# Creates the logharbour index with proper mappings

echo "Waiting for Elasticsearch to be ready..."
sleep 5

# Create index with mappings
echo "Creating logharbour index..."
curl -s -X PUT "${ES_URL}/logharbour" \
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
  }'

echo ""
echo "Index created successfully"