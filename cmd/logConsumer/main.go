package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/remiges-tech/logharbour/logharbour"
)

var logger *slog.Logger

func setupLogger(level string) {
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
		AddSource: true,
	}
	
	handler := slog.NewTextHandler(os.Stdout, opts)
	logger = slog.New(handler)
	slog.SetDefault(logger)
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func startKafkaConsumer(consumer logharbour.Consumer, batchSize int, batchTimeout time.Duration) (<-chan error, error) {
	return consumer.Start(batchSize, batchTimeout)
}

// handleErrors monitors the consumer error channel and detects when the consumer dies.
// 
// Consumer Death Detection Mechanism:
// When the Kafka consumer encounters a fatal error (e.g., connection failure), the
// consumer goroutine exits and closes its error channel. This function detects that
// closure and signals the main goroutine via the returned 'done' channel.
//
// This addresses a critical issue where the consumer would silently fail but the
// process would continue running, appearing healthy to monitoring systems while
// actually doing no work.
//
// Returns a channel that will be closed when the error handler goroutine exits,
// which happens when:
// 1. The consumer's error channel is closed (consumer died)
// 2. A panic error is detected (process exits immediately via os.Exit)
func handleErrors(errs <-chan error) (<-chan struct{}) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		errorCount := 0
		// Range over the error channel - this loop exits when the channel is closed
		for err := range errs {
			errorCount++
			logger.Error("Consumer error occurred", 
				slog.String("error", err.Error()),
				slog.Int("total_errors", errorCount))
			
			// Check if this is a panic error and exit immediately
			if err != nil && strings.Contains(err.Error(), "panic in") {
				logger.Error("Panic detected in consumer, exiting process", 
					slog.String("error", err.Error()))
				os.Exit(1)
			}
		}
		// If we reach here, the error channel was closed, meaning the consumer died
		logger.Error("Consumer error channel closed - consumer has stopped running")
	}()
	return done
}


func stopKafkaConsumer(consumer logharbour.Consumer) {
	stopStartTime := time.Now()
	if err := consumer.Stop(); err != nil {
		logger.Error("Failed to stop consumer", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Debug("Consumer stopped", slog.Duration("stop_duration", time.Since(stopStartTime)))
}

func retryOperation(operation func() error, maxAttempts int, initialBackoff time.Duration) error {
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		attemptStart := time.Now()
		err := operation()
		attemptDuration := time.Since(attemptStart)
		
		if err == nil {
			if attempt > 1 {
				logger.Info("Retry operation succeeded",
					slog.Int("attempt", attempt),
					slog.Duration("attempt_duration", attemptDuration))
			}
			return nil // Success
		}

		if attempt == maxAttempts {
			logger.Error("Retry operation failed after all attempts",
				slog.Int("max_attempts", maxAttempts),
				slog.String("final_error", err.Error()),
				slog.Duration("final_attempt_duration", attemptDuration))
			return fmt.Errorf("after %d attempts, last error: %s", attempt, err)
		}

		wait := initialBackoff * time.Duration(1<<(attempt-1)) // Exponential backoff
		logger.Warn("Retry operation failed, retrying",
			slog.Int("attempt", attempt),
			slog.Int("max_attempts", maxAttempts),
			slog.Duration("wait_duration", wait),
			slog.String("error", err.Error()),
			slog.Duration("attempt_duration", attemptDuration))
		time.Sleep(wait)
	}
	return fmt.Errorf("reached max attempts without success")
}

func main() {
	// Define flags with environment variables as default values
	esAddresses := flag.String("esAddresses", getEnv("ELASTICSEARCH_ADDRESSES", "http://localhost:9200"), "Elasticsearch addresses (comma-separated)")
	kafkaBrokers := flag.String("kafkaBrokers", getEnv("KAFKA_BROKERS", "localhost:9092"), "Kafka brokers (comma-separated)")
	kafkaTopic := flag.String("kafkaTopic", getEnv("KAFKA_TOPIC", "log_topic"), "Kafka topic")
	esIndex := flag.String("esIndex", getEnv("ELASTICSEARCH_INDEX", "logs"), "Elasticsearch index name")
	offsetType := flag.String("offsetType", getEnv("KAFKA_OFFSET_TYPE", "earliest"), "Kafka offset type: 'earliest', 'latest', or a specific number")
	batchSize := flag.Int("batchSize", func() int {
		if val := getEnv("KAFKA_BATCH_SIZE", "1000"); val != "" {
			if size, err := strconv.Atoi(val); err == nil {
				return size
			}
		}
		return 1000
	}(), "Kafka consumer batch size")
	batchTimeout := flag.Duration("batchTimeout", func() time.Duration {
		if val := getEnv("KAFKA_BATCH_TIMEOUT", "60s"); val != "" {
			if duration, err := time.ParseDuration(val); err == nil {
				return duration
			}
		}
		return 60 * time.Second
	}(), "Kafka consumer batch timeout")
	consumerGroup := flag.String("consumerGroup", getEnv("KAFKA_CONSUMER_GROUP", "logharbour-consumer-group"), "Kafka consumer group ID")
	useConsumerGroup := flag.Bool("useConsumerGroup", getEnv("USE_CONSUMER_GROUP", "true") == "true", "Use consumer group instead of regular consumer")
	logLevel := flag.String("logLevel", getEnv("LOG_LEVEL", "info"), "Log level: debug, info, warn, error")
	
	esUsername := flag.String("esUsername", getEnv("ELASTICSEARCH_USERNAME", ""), "Elasticsearch username (optional, enables authentication when provided)")
	esPassword := flag.String("esPassword", getEnv("ELASTICSEARCH_PASSWORD", ""), "Elasticsearch password (optional)")
	esCACert := flag.String("esCACert", getEnv("ELASTICSEARCH_CA_CERT", ""), "Path to Elasticsearch CA certificate (optional, for HTTPS)")

	dlqEnabled := flag.Bool("dlqEnabled", getEnv("KAFKA_DLQ_ENABLED", "false") == "true", "Enable Dead Letter Queue for failed messages")
	dlqTopic := flag.String("dlqTopic", getEnv("KAFKA_DLQ_TOPIC", ""), "DLQ topic name (default: <source_topic>_dlq)")

	// Parse flags
	flag.Parse()

	// Setup logging
	setupLogger(*logLevel)
	
	logger.Info("Starting LogHarbour Consumer",
		slog.String("version", "1.0.0"),
		slog.String("log_level", *logLevel))

	// Process offset type
	var specificOffset int64 = -1
	var offsetTypeEnum logharbour.OffsetType
	switch *offsetType {
	case "earliest":
		offsetTypeEnum = logharbour.OffsetEarliest
	case "latest":
		offsetTypeEnum = logharbour.OffsetLatest
	default:
		// Try to parse as a specific offset
		if parsedOffset, err := strconv.ParseInt(*offsetType, 10, 64); err == nil {
			specificOffset = parsedOffset
			offsetTypeEnum = logharbour.OffsetSpecific
			logger.Debug("Using specific offset", slog.Int64("offset", specificOffset))
		} else {
			// If not a valid number, default to earliest
			logger.Warn("Invalid offset type, using 'earliest' instead",
				slog.String("provided_offset", *offsetType),
				slog.String("fallback", "earliest"))
			offsetTypeEnum = logharbour.OffsetEarliest
		}
	}

	logger.Info("Consumer configuration",
		slog.String("elasticsearch_addresses", *esAddresses),
		slog.String("kafka_brokers", *kafkaBrokers),
		slog.String("kafka_topic", *kafkaTopic),
		slog.String("elasticsearch_index", *esIndex),
		slog.String("offset_type", *offsetType),
		slog.Any("offset_enum", offsetTypeEnum),
		slog.Int64("specific_offset", specificOffset),
		slog.Int("batch_size", *batchSize),
		slog.Duration("batch_timeout", *batchTimeout),
		slog.Bool("use_consumer_group", *useConsumerGroup),
		slog.String("consumer_group_id", *consumerGroup),
		slog.Bool("elasticsearch_auth_enabled", *esPassword != ""),
		slog.String("elasticsearch_username", *esUsername),
		slog.Bool("elasticsearch_tls_ca_provided", *esCACert != ""))

	logger.Debug("Creating Elasticsearch client")
	startTime := time.Now()
	esClient, err := createElasticsearchClient(*esAddresses, *esUsername, *esPassword, *esCACert)
	if err != nil {
		logger.Error("Failed to create Elasticsearch client", 
			slog.String("error", err.Error()),
			slog.String("addresses", *esAddresses))
		os.Exit(1)
	}
	logger.Debug("Elasticsearch client created", 
		slog.Duration("duration", time.Since(startTime)))

	logger.Debug("Setting up Elasticsearch index")
	startTime = time.Now()
	err = setupElasticsearchIndex(esClient, *esIndex)
	if err != nil {
		logger.Error("Failed to setup Elasticsearch index", 
			slog.String("error", err.Error()),
			slog.String("index", *esIndex))
		os.Exit(1)
	}
	logger.Debug("Elasticsearch index setup completed",
		slog.Duration("duration", time.Since(startTime)),
		slog.String("index", *esIndex))

	// Set default DLQ topic if not provided
	if *dlqTopic == "" {
		*dlqTopic = *kafkaTopic + "_dlq"
	}

	// Create DLQ producer if enabled
	var dlqProducer sarama.SyncProducer
	if *dlqEnabled {
		var err error
		dlqProducer, err = createDLQProducer(*kafkaBrokers)
		if err != nil {
			logger.Error("Failed to create DLQ producer",
				slog.String("error", err.Error()))
			os.Exit(1)
		}
		defer dlqProducer.Close()
		logger.Info("DLQ enabled",
			slog.String("dlq_topic", *dlqTopic))
	}

	handler := func(messages []*sarama.ConsumerMessage) error {
		// Error Handling Overview:
		// 1. Validation Phase: Invalid messages are logged and skipped (not sent to ES)
		// 2. Bulk Indexing Phase: Documents are sent to ES in a single bulk request
		// 3. Retry Logic: Network/connection failures trigger retries of entire batch
		// 4. Partial Failures: Some documents may fail (e.g., mapping errors) while others succeed
		// 5. Error Propagation: Indexing failures return error (without DLQ) or nil (with DLQ)
		//
		// Error Categories:
		// - Validation Errors: Bad data that can't be parsed (invalid JSON, missing ID)
		//   Action: Skip message, send to DLQ if enabled, otherwise LOST
		// - Indexing Errors: Valid data that Elasticsearch rejects (mapping conflicts, doc too large)
		//   Action: Send to DLQ if enabled, otherwise return error to block offset commit
		//
		// WARNING: Without DLQ (disabled by default), validation errors cause silent message loss.
		// Enable DLQ with KAFKA_DLQ_ENABLED=true to preserve failed messages.
		
		batchStartTime := time.Now()
		batchSize := len(messages)
		
		logger.Debug("Processing message batch",
			slog.Int("batch_size", batchSize),
			slog.String("topic", messages[0].Topic),
			slog.Int("partition", int(messages[0].Partition)))

		// Phase 1: Validation - Prepare documents for bulk indexing
		bulkDocs := make([]logharbour.BulkDocument, 0, batchSize)
		docIDToMessage := make(map[string]*sarama.ConsumerMessage)
		validationErrors := 0
		
		for i, message := range messages {
			logger.Debug("Processing message",
				slog.Int("message_index", i),
				slog.Int64("offset", message.Offset),
				slog.Int("partition", int(message.Partition)),
				slog.Time("timestamp", message.Timestamp),
				slog.Int("value_size_bytes", len(message.Value)))

			var logEntry map[string]interface{}
			err := json.Unmarshal(message.Value, &logEntry)
			if err != nil {
				// Validation error: Skip this message, it won't be sent to Elasticsearch
				validationErrors++
				logger.Warn("Failed to unmarshal log message",
					slog.String("error", err.Error()),
					slog.Int64("offset", message.Offset),
					slog.Int("message_size", len(message.Value)))
				if dlqProducer != nil {
					sendToDLQ(dlqProducer, *dlqTopic, message, "json_unmarshal_error: "+err.Error())
				}
				continue
			}

			id, ok := logEntry["id"].(string)
			if !ok || id == "" {
				// Validation error: Document must have an ID for Elasticsearch
				validationErrors++
				logger.Warn("Missing or invalid 'id' field in log message",
					slog.Int64("offset", message.Offset),
					slog.Any("log_entry_keys", getMapKeys(logEntry)))
				if dlqProducer != nil {
					sendToDLQ(dlqProducer, *dlqTopic, message, "missing_id_field")
				}
				continue
			}

			bulkDocs = append(bulkDocs, logharbour.BulkDocument{
				ID:   id,
				Body: string(message.Value),
			})
			docIDToMessage[id] = message
		}

		if len(bulkDocs) == 0 {
			// All messages failed validation - nothing to send to Elasticsearch
			logger.Warn("No valid documents to index in batch",
				slog.Int("batch_size", batchSize),
				slog.Int("validation_errors", validationErrors))
			return nil
		}

		// Phase 2: Bulk Indexing - Send all valid documents to Elasticsearch
		writeStartTime := time.Now()
		var bulkResult *logharbour.BulkWriteResult
		err := retryOperation(func() error {
			var err error
			bulkResult, err = esClient.BulkWrite(*esIndex, bulkDocs)
			if err != nil {
				// Network/connection error - retry entire batch
				return err
			}
			// If all documents failed (e.g., index closed), treat as error for retry
			if bulkResult.Failed == len(bulkDocs) {
				return fmt.Errorf("all documents failed to index")
			}
			return nil
		}, 10, 1*time.Second)
		
		writeDuration := time.Since(writeStartTime)
		
		if err != nil {
			logger.Error("Failed to bulk write to Elasticsearch after retries", 
				slog.String("error", err.Error()),
				slog.Int("documents_count", len(bulkDocs)),
				slog.Duration("write_duration", writeDuration))
			return err
		}
		
		// Phase 3: Error Analysis - Log results and handle partial failures
		if bulkResult.Failed > 0 {
			// Partial failure: Some documents succeeded, some failed
			// Common causes: mapping conflicts, field validation errors, document too large
			logger.Warn("Bulk write completed with some failures",
				slog.Int("successful", bulkResult.Successful),
				slog.Int("failed", bulkResult.Failed),
				slog.Int("error_count", len(bulkResult.Errors)),
				slog.Duration("write_duration", writeDuration))
			
			// Log first few errors for debugging (full list available in bulkResult.Errors)
			maxErrors := 5
			if len(bulkResult.Errors) < maxErrors {
				maxErrors = len(bulkResult.Errors)
			}
			for i := 0; i < maxErrors; i++ {
				logger.Error("Document indexing failed",
					slog.String("document_id", bulkResult.Errors[i].DocumentID),
					slog.String("error", bulkResult.Errors[i].Error),
					slog.Int("status", bulkResult.Errors[i].Status))
			}
			if len(bulkResult.Errors) > maxErrors {
				logger.Warn("More errors occurred",
					slog.Int("additional_errors", len(bulkResult.Errors)-maxErrors))
			}

			// Send failed documents to DLQ
			if dlqProducer != nil {
				sentToDLQ := handleIndexingFailures(bulkResult.Errors, docIDToMessage, dlqProducer, *dlqTopic)
				logger.Info("Sent failed documents to DLQ",
					slog.Int("sent_count", sentToDLQ),
					slog.Int("failed_count", bulkResult.Failed))
			}
		}
		
		batchDuration := time.Since(batchStartTime)
		logger.Info("Batch processing completed",
			slog.Int("batch_size", batchSize),
			slog.Int("validation_errors", validationErrors),
			slog.Int("documents_sent", len(bulkDocs)),
			slog.Int("success_count", bulkResult.Successful),
			slog.Int("error_count", bulkResult.Failed),
			slog.Duration("batch_duration", batchDuration),
			slog.Float64("messages_per_second", float64(bulkResult.Successful)/batchDuration.Seconds()))
		
		// Phase 4: Error Propagation - Prevent Kafka offset commit on failures
		// By returning an error, we ensure the Kafka consumer doesn't commit the offset,
		// allowing these messages to be reprocessed on next startup.
		// With DLQ enabled, failed messages are safely stored, so we allow offset commit.
		if bulkResult.Failed > 0 && dlqProducer == nil {
			return fmt.Errorf("%d documents failed to index", bulkResult.Failed)
		}
		
		return nil
	}

	var consumer logharbour.Consumer
	logger.Debug("Creating Kafka consumer")
	consumerCreateStart := time.Now()
	
	if *useConsumerGroup {
		logger.Info("Creating consumer group",
			slog.String("group_id", *consumerGroup),
			slog.String("topic", *kafkaTopic))
		consumer, err = createKafkaConsumerGroup(*kafkaBrokers, *consumerGroup, *kafkaTopic, handler, offsetTypeEnum)
	} else {
		logger.Info("Creating regular consumer",
			slog.String("topic", *kafkaTopic))
		consumer, err = createKafkaConsumer(*kafkaBrokers, *kafkaTopic, handler, offsetTypeEnum, specificOffset)
	}
	
	if err != nil {
		logger.Error("Failed to create consumer", 
			slog.String("error", err.Error()),
			slog.Bool("consumer_group_mode", *useConsumerGroup))
		os.Exit(1)
	}
	
	logger.Debug("Kafka consumer created", 
		slog.Duration("creation_duration", time.Since(consumerCreateStart)))

	logger.Info("Starting Kafka consumer")
	consumerStartTime := time.Now()
	errs, err := startKafkaConsumer(consumer, *batchSize, *batchTimeout)
	if err != nil {
		logger.Error("Failed to start consumer", 
			slog.String("error", err.Error()))
		os.Exit(1)
	}
	
	logger.Info("Kafka consumer started successfully",
		slog.Duration("startup_duration", time.Since(consumerStartTime)))

	// Start monitoring consumer errors and get a channel that signals consumer death
	consumerDone := handleErrors(errs)

	logger.Info("Consumer is running. Press Ctrl+C to stop.")
	
	// Consumer Health Monitoring:
	// This select statement waits for one of two events:
	// 1. Consumer death (via consumerDone channel)
	// 2. User interrupt (Ctrl+C)
	//
	// This fixes a critical issue where the Kafka consumer goroutine would exit
	// on connection errors but the main process would continue running indefinitely,
	// making it appear healthy to monitoring systems while doing no actual work.
	//
	// Example scenario this addresses:
	// 1. Consumer starts and connects to Kafka successfully
	// 2. Network issue or Kafka becomes unavailable
	// 3. Consumer's Consume() method returns error and goroutine exits
	// 4. Error channel closes, triggering consumerDone
	// 5. Process exits with code 1, signaling failure to orchestrators
	select {
	case <-consumerDone:
		// Consumer died unexpectedly - exit immediately with error code
		// This ensures the failure is visible to container orchestrators,
		// systemd, or other process managers that can restart the service
		logger.Error("Consumer has stopped unexpectedly, exiting process")
		// TODO: Implement retry mechanism with exponential backoff instead of exiting
		// - Add configurable retry attempts and backoff duration
		// - Attempt to recreate the consumer and restart consumption
		// - Only exit after max retries are exhausted
		// - Consider implementing circuit breaker pattern for better resilience
		os.Exit(1)
	case <-func() <-chan os.Signal {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt)
		return signals
	}():
		// Normal shutdown path - user pressed Ctrl+C
		logger.Info("Shutdown signal received, stopping consumer")
	}

	stopKafkaConsumer(consumer)
	logger.Info("Consumer stopped gracefully")
}

func createElasticsearchClient(addresses, username, password, caCertPath string) (*logharbour.ElasticsearchClient, error) {
	esConfig := elasticsearch.Config{
		Addresses: strings.Split(addresses, ","),
	}

	if password != "" {
		esConfig.Username = username
		esConfig.Password = password
		
		if username == "" {
			esConfig.Username = "elastic"
		}
		
		logger.Info("Elasticsearch authentication enabled",
			slog.String("username", esConfig.Username))
	} else {
		logger.Info("Elasticsearch authentication disabled (no credentials provided)")
	}

	if caCertPath != "" {
		transport, err := createTLSTransport(caCertPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS transport: %w", err)
		}
		esConfig.Transport = transport
		logger.Info("Elasticsearch TLS configuration enabled",
			slog.String("ca_cert_path", caCertPath))
	}

	return logharbour.NewElasticsearchClient(esConfig)
}

func createTLSTransport(caCertPath string) (*http.Transport, error) {
	caCert, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return transport, nil
}

func createKafkaConsumer(brokers, topic string, handler logharbour.MessageHandler, offsetType logharbour.OffsetType, specificOffset int64) (logharbour.Consumer, error) {
	return logharbour.NewConsumer(strings.Split(brokers, ","), topic, handler, offsetType, specificOffset)
}

func createKafkaConsumerGroup(brokers, groupID, topic string, handler logharbour.MessageHandler, offsetType logharbour.OffsetType) (logharbour.Consumer, error) {
	return logharbour.NewConsumerGroup(strings.Split(brokers, ","), groupID, topic, handler, offsetType)
}

func indexExists(client *logharbour.ElasticsearchClient, indexName string) (bool, error) {
	exists, err := client.IndexExists(indexName)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func createIndexWithMapping(client *logharbour.ElasticsearchClient, indexName string) error {
	err := client.CreateIndex(indexName, logharbour.ESLogsMapping)
	if err != nil {
		return fmt.Errorf("failed to create index: %v", err)
	}
	return nil
}

func setupElasticsearchIndex(client *logharbour.ElasticsearchClient, indexName string) error {
	exists, err := indexExists(client, indexName)
	if err != nil {
		return fmt.Errorf("error checking if index exists: %v", err)
	}
	if !exists {
		if err := createIndexWithMapping(client, indexName); err != nil {
			return err
		}
	}
	return nil
}

func createDLQProducer(brokers string) (sarama.SyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 3

	return sarama.NewSyncProducer(strings.Split(brokers, ","), config)
}

func sendToDLQ(producer sarama.SyncProducer, topic string, originalMsg *sarama.ConsumerMessage, reason string) {
	headers := []sarama.RecordHeader{
		{Key: []byte("dlq_reason"), Value: []byte(reason)},
		{Key: []byte("original_topic"), Value: []byte(originalMsg.Topic)},
		{Key: []byte("original_partition"), Value: []byte(fmt.Sprintf("%d", originalMsg.Partition))},
		{Key: []byte("original_offset"), Value: []byte(fmt.Sprintf("%d", originalMsg.Offset))},
	}

	msg := &sarama.ProducerMessage{
		Topic:   topic,
		Value:   sarama.ByteEncoder(originalMsg.Value),
		Headers: headers,
	}

	_, _, err := producer.SendMessage(msg)
	if err != nil {
		logger.Error("Failed to send message to DLQ",
			slog.String("error", err.Error()),
			slog.String("dlq_topic", topic),
			slog.Int64("original_offset", originalMsg.Offset))
	} else {
		logger.Debug("Message sent to DLQ",
			slog.String("dlq_topic", topic),
			slog.String("reason", reason),
			slog.Int64("original_offset", originalMsg.Offset))
	}
}

// handleIndexingFailures sends documents that failed ES indexing to the DLQ.
// Returns the number of documents sent to DLQ.
func handleIndexingFailures(
	errors []logharbour.BulkError,
	docIDToMessage map[string]*sarama.ConsumerMessage,
	dlqProducer sarama.SyncProducer,
	dlqTopic string,
) int {
	sentCount := 0
	for _, docErr := range errors {
		if originalMsg, ok := docIDToMessage[docErr.DocumentID]; ok {
			sendToDLQ(dlqProducer, dlqTopic, originalMsg, "indexing_error: "+docErr.Error)
			sentCount++
		}
	}
	return sentCount
}
