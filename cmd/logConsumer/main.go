package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
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

func startKafkaConsumer(consumer logharbour.Consumer, batchSize int) (<-chan error, error) {
	return consumer.Start(batchSize)
}

func handleErrors(errs <-chan error) {
	go func() {
		errorCount := 0
		for err := range errs {
			errorCount++
			logger.Error("Consumer error occurred", 
				slog.String("error", err.Error()),
				slog.Int("total_errors", errorCount))
			// Decide what to do here: retry, ignore, etc.
		}
	}()
}

func waitForInterrupt() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	<-signals
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
		if val := getEnv("KAFKA_BATCH_SIZE", "10"); val != "" {
			if size, err := strconv.Atoi(val); err == nil {
				return size
			}
		}
		return 10
	}(), "Kafka consumer batch size")
	consumerGroup := flag.String("consumerGroup", getEnv("KAFKA_CONSUMER_GROUP", "logharbour-consumer-group"), "Kafka consumer group ID")
	useConsumerGroup := flag.Bool("useConsumerGroup", getEnv("USE_CONSUMER_GROUP", "true") == "true", "Use consumer group instead of regular consumer")
	logLevel := flag.String("logLevel", getEnv("LOG_LEVEL", "info"), "Log level: debug, info, warn, error")

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
		slog.Bool("use_consumer_group", *useConsumerGroup),
		slog.String("consumer_group_id", *consumerGroup))

	logger.Debug("Creating Elasticsearch client")
	startTime := time.Now()
	esClient, err := createElasticsearchClient(*esAddresses)
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

	handler := func(messages []*sarama.ConsumerMessage) error {
		batchStartTime := time.Now()
		batchSize := len(messages)
		
		logger.Debug("Processing message batch",
			slog.Int("batch_size", batchSize),
			slog.String("topic", messages[0].Topic),
			slog.Int("partition", int(messages[0].Partition)))

		successCount := 0
		errorCount := 0
		
		for i, message := range messages {
			messageStartTime := time.Now()
			
			logger.Debug("Processing message",
				slog.Int("message_index", i),
				slog.Int64("offset", message.Offset),
				slog.Int("partition", int(message.Partition)),
				slog.Time("timestamp", message.Timestamp),
				slog.Int("value_size_bytes", len(message.Value)))

			var logEntry map[string]interface{}
			err := json.Unmarshal(message.Value, &logEntry)
			if err != nil {
				errorCount++
				logger.Warn("Failed to unmarshal log message", 
					slog.String("error", err.Error()),
					slog.Int64("offset", message.Offset),
					slog.Int("message_size", len(message.Value)))
				continue
			}

			id, ok := logEntry["id"].(string)
			if !ok {
				errorCount++
				logger.Warn("Missing or invalid 'id' field in log message",
					slog.Int64("offset", message.Offset),
					slog.Any("log_entry_keys", getMapKeys(logEntry)))
				continue
			}

			writeStartTime := time.Now()
			err = retryOperation(func() error {
				return esClient.Write(*esIndex, id, string(message.Value))
			}, 10, 1*time.Second)
			
			if err != nil {
				errorCount++
				logger.Error("Failed to write message to Elasticsearch after retries", 
					slog.String("error", err.Error()),
					slog.String("document_id", id),
					slog.Int64("offset", message.Offset),
					slog.Duration("write_duration", time.Since(writeStartTime)))
				return err
			}
			
			successCount++
			logger.Debug("Message written to Elasticsearch successfully",
				slog.String("document_id", id),
				slog.Int64("offset", message.Offset),
				slog.Duration("write_duration", time.Since(writeStartTime)),
				slog.Duration("total_message_duration", time.Since(messageStartTime)))
		}
		
		batchDuration := time.Since(batchStartTime)
		logger.Info("Batch processing completed",
			slog.Int("batch_size", batchSize),
			slog.Int("success_count", successCount),
			slog.Int("error_count", errorCount),
			slog.Duration("batch_duration", batchDuration),
			slog.Float64("messages_per_second", float64(successCount)/batchDuration.Seconds()))
		
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
	errs, err := startKafkaConsumer(consumer, *batchSize)
	if err != nil {
		logger.Error("Failed to start consumer", 
			slog.String("error", err.Error()))
		os.Exit(1)
	}
	
	logger.Info("Kafka consumer started successfully",
		slog.Duration("startup_duration", time.Since(consumerStartTime)))

	handleErrors(errs)

	logger.Info("Consumer is running. Press Ctrl+C to stop.")
	waitForInterrupt()

	logger.Info("Shutdown signal received, stopping consumer")
	stopKafkaConsumer(consumer)
	logger.Info("Consumer stopped gracefully")
}

func createElasticsearchClient(addresses string) (*logharbour.ElasticsearchClient, error) {
	esConfig := elasticsearch.Config{
		Addresses: strings.Split(addresses, ","),
	}
	return logharbour.NewElasticsearchClient(esConfig)
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
