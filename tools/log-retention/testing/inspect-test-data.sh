#!/bin/bash

# Check the distribution of test data in Elasticsearch

ES_URL="${ES_URL:-http://localhost:9200}"
ES_INDEX="${ES_INDEX:-logharbour}"

echo "LogHarbour Data Check"
echo "===================="
echo ""

# Total count
total=$(curl -s "$ES_URL/$ES_INDEX/_count" | jq -r '.count')
echo "Total documents: $total"
echo ""

# Count by date ranges
echo "Documents by age:"
echo "----------------"

# Today
today=$(curl -s "$ES_URL/$ES_INDEX/_count" -H 'Content-Type: application/json' \
    -d '{"query": {"range": {"when": {"gte": "now/d", "lt": "now"}}}}' | jq -r '.count')
echo "Today: $today"

# 1-7 days old
week=$(curl -s "$ES_URL/$ES_INDEX/_count" -H 'Content-Type: application/json' \
    -d '{"query": {"range": {"when": {"gte": "now-7d", "lt": "now-1d"}}}}' | jq -r '.count')
echo "1-7 days old: $week"

# 7-30 days old
month=$(curl -s "$ES_URL/$ES_INDEX/_count" -H 'Content-Type: application/json' \
    -d '{"query": {"range": {"when": {"gte": "now-30d", "lt": "now-7d"}}}}' | jq -r '.count')
echo "7-30 days old: $month"

# Older than 30 days
old=$(curl -s "$ES_URL/$ES_INDEX/_count" -H 'Content-Type: application/json' \
    -d '{"query": {"range": {"when": {"lt": "now-30d"}}}}' | jq -r '.count')
echo "Older than 30 days: $old"

echo ""
echo "Distribution by type:"
echo "-------------------"
curl -s "$ES_URL/$ES_INDEX/_search" -H 'Content-Type: application/json' \
    -d '{
      "size": 0,
      "aggs": {
        "types": {
          "terms": {
            "field": "type",
            "size": 10
          }
        }
      }
    }' | jq -r '.aggregations.types.buckets[] | "\(.key): \(.doc_count)"'

echo ""
echo "Distribution by priority:"
echo "-----------------------"
curl -s "$ES_URL/$ES_INDEX/_search" -H 'Content-Type: application/json' \
    -d '{
      "size": 0,
      "aggs": {
        "priorities": {
          "terms": {
            "field": "pri",
            "size": 10
          }
        }
      }
    }' | jq -r '.aggregations.priorities.buckets[] | "\(.key): \(.doc_count)"'

echo ""
echo "Sample of oldest documents:"
echo "-------------------------"
curl -s "$ES_URL/$ES_INDEX/_search" -H 'Content-Type: application/json' \
    -d '{
      "size": 5,
      "_source": ["when", "app", "type", "pri"],
      "sort": [{"when": "asc"}]
    }' | jq -r '.hits.hits[]._source | "\(.when) | \(.app) | \(.type) | \(.pri)"'