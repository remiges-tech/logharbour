package logharbour

// ESLogsMapping defines the Elasticsearch index mapping for log entries.
//
// Migration for existing indices:
//
// If you have an existing index created before trace_id/span_id fields were added,
// you can update the mapping in place without reindexing:
//
//	PUT /<index_name>/_mapping
//	{
//	  "properties": {
//	    "trace_id": { "type": "keyword" },
//	    "span_id": { "type": "keyword" }
//	  }
//	}
//
// This adds the new fields to existing indices. Documents without these fields
// will simply have null values when queried.
const ESLogsMapping = `{
  "mappings": {
    "properties": {
      "id": {
        "type": "keyword"
      },
      "app": {
        "type": "keyword"
      },
      "system": {
        "type": "keyword"
      },
      "module": {
        "type": "keyword"
      },
      "type": {
        "type": "keyword"
      },
      "pri": {
        "type": "keyword"
      },
      "when": {
        "type": "date"
      },
      "who": {
        "type": "keyword"
      },
      "op": {
        "type": "keyword"
      },
      "class": {
        "type": "keyword"
      },
      "instance": {
        "type": "keyword"
      },
      "status": {
        "type": "integer"
      },
      "error": {
        "type": "text"
      },
      "remote_ip": {
        "type": "ip"
      },
      "trace_id": {
        "type": "keyword"
      },
      "span_id": {
        "type": "keyword"
      },
      "msg": {
        "type": "text"
      },
      "data": {
        "properties": {
          "change_data": {
            "properties": {
              "entity": {
                "type": "keyword"
              },
              "op": {
                "type": "keyword"
              },
              "changes": {
                "type": "nested",
                "properties": {
                  "field": {
                    "type": "keyword"
                  },
                  "old_value": {
                    "type": "text"
                  },
                  "new_value": {
                    "type": "text"
                  }
                }
              }
            }
          },
          "activity_data": {
            "type": "text"
          },
          "debug_data": {
            "properties": {
              "pid": {
                "type": "integer"
              },
              "runtime": {
                "type": "keyword"
              },
              "file": {
                "type": "keyword"
              },
              "line": {
                "type": "integer"
              },
              "func": {
                "type": "keyword"
              },
              "stackTrace": {
                "type": "text"
              },
              "data": {
                "type": "object",
                "enabled": false
              }
            }
          }
        }
      }
    }
  }
}`
