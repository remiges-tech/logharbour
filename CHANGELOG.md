# Changelog

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [v0.25.0] - 2026-01-22

### Performance

- **Early return in logging methods** - skip JSON marshaling, stack traces, and `fmt.Sprintf` when log priority is below minimum or debug mode is off
- **Lock-free priority check** - replaced mutex with atomic operations for `minLogPriority` reads

## [v0.24.0] - 2026-01-12

### Added

- **Dead Letter Queue (DLQ)** - captures failed messages instead of losing them
  - `KAFKA_DLQ_ENABLED` - enable DLQ (default: false)
  - `KAFKA_DLQ_TOPIC` - DLQ topic name (default: `{source_topic}_dlq`)
  - Captures validation errors (invalid JSON, missing ID)
  - Captures Elasticsearch indexing failures
  - Preserves original message with headers: `dlq_reason`, `original_topic`, `original_partition`, `original_offset`

- **Elasticsearch mapping** - `trace_id` and `span_id` fields
  - **Migration**: For existing indices, see `logharbour/es_logs_mapping.go` for mapping update instructions

### Fixed

- **Consumer offset timing** - messages marked as processed only after handler succeeds, preventing data loss on handler failures

### Changed

- **Elasticsearch image** - switched from Bitnami to official `docker.elastic.co/elasticsearch/elasticsearch:8.12.0` (Bitnami discontinued free images)
- **Memory limits** - added ES_JAVA_OPTS (512MB) in test containers to prevent OOM

## [v0.23.0] - 2026-01-10

### Added

- **Trace ID extraction** - for distributed tracing across services
  - `WithHTTPRequest(r *http.Request)` - extracts trace ID from `X-Trace-ID` header (generates KSUID if missing) and client IP from `X-Forwarded-For`/`X-Real-IP`/`RemoteAddr`
  - `WithTraceID(traceId string)` - set trace ID manually
  - `HeaderTraceID` constant
  - `TraceID` field in `GetLogsParam` for filtering

### Deprecated

- `WithSpanAndTrace()` - use `WithTraceID()` instead (does not follow fluent pattern)

## [v0.22.0] - 2025-08-11

### Added

- **X-Pack Security** - authentication support for Elasticsearch
  - Basic authentication (username/password)
  - TLS/HTTPS with CA certificate
  - Environment variables: `ELASTICSEARCH_USERNAME`, `ELASTICSEARCH_PASSWORD`, `ELASTICSEARCH_CA_CERT`

[v0.25.0]: https://github.com/remiges-tech/logharbour/compare/v0.24.0...v0.25.0
[v0.24.0]: https://github.com/remiges-tech/logharbour/compare/v0.23.0...v0.24.0
[v0.23.0]: https://github.com/remiges-tech/logharbour/compare/v0.22.0...v0.23.0
[v0.22.0]: https://github.com/remiges-tech/logharbour/releases/tag/v0.22.0
