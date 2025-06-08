package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/remiges-tech/logharbour/logharbour"
)

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func startKafkaConsumer(consumer logharbour.Consumer, batchSize int) (<-chan error, error) {
	return consumer.Start(batchSize)
}

func handleErrors(errs <-chan error) {
	go func() {
		for err := range errs {
			log.Printf("Failed to process batch: %v", err)
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
	if err := consumer.Stop(); err != nil {
		log.Fatalln("Failed to stop consumer: ", err)
	}
}

func retryOperation(operation func() error, maxAttempts int, initialBackoff time.Duration) error {
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := operation()
		if err == nil {
			return nil // Success
		}

		if attempt == maxAttempts {
			return fmt.Errorf("after %d attempts, last error: %s", attempt, err)
		}

		wait := initialBackoff * time.Duration(1<<(attempt-1)) // Exponential backoff
		log.Printf("Attempt %d failed, retrying in %v: %v", attempt, wait, err)
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

	// Parse flags
	flag.Parse()

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
		} else {
			// If not a valid number, default to earliest
			log.Printf("Invalid offset type: %s. Using 'earliest' instead", *offsetType)
			offsetTypeEnum = logharbour.OffsetEarliest
		}
	}

	log.Printf("Elasticsearch Addresses: %s", *esAddresses)
	log.Printf("Kafka Brokers: %s", *kafkaBrokers)
	log.Printf("Kafka Topic: %s", *kafkaTopic)
	log.Printf("Elasticsearch Index: %s", *esIndex)
	log.Printf("Kafka Offset Type: %s (value: %v, specific offset: %d)", *offsetType, offsetTypeEnum, specificOffset)
	log.Printf("Kafka Batch Size: %d", *batchSize)

	esClient, err := createElasticsearchClient(*esAddresses)
	if err != nil {
		log.Fatalf("Error creating the Elasticsearch client: %s", err)
	}

	err = setupElasticsearchIndex(esClient, *esIndex)
	if err != nil {
		log.Fatalf("Error setting up Elasticsearch index: %s", err)
	}

	handler := func(messages []*sarama.ConsumerMessage) error {
		for _, message := range messages {
			// log debug
			// log.Printf("Received message from topic %s: %s", message.Topic, string(message.Value))
			// err := esClient.Write(*esIndex, string(message.Key), string(message.Value)) // Use the esIndex variable here
			var logEntry map[string]interface{}
			err := json.Unmarshal(message.Value, &logEntry)
			if err != nil {
				log.Printf("Failed to unmarshal log message: %v", err)
				continue
			}

			id, ok := logEntry["id"].(string)
			if !ok {
				log.Printf("Missing or invalid 'id' field in log message")
				continue
			}
			err = retryOperation(func() error {
				return esClient.Write(*esIndex, id, string(message.Value))
			}, 10, 1*time.Second) // Adjust maxAttempts and initialBackoff as needed
			if err != nil {
				log.Printf("Failed to write message %v to Elasticsearch: %v", message.Value, err)
				return err
			}
		}
		return nil
	}

	consumer, err := createKafkaConsumer(*kafkaBrokers, *kafkaTopic, handler, offsetTypeEnum, specificOffset)
	if err != nil {
		log.Fatalln("Failed to create consumer: ", err)
	}

	errs, err := startKafkaConsumer(consumer, *batchSize)
	if err != nil {
		log.Fatalln("Failed to start consumer: ", err)
	}

	handleErrors(errs)

	waitForInterrupt()

	stopKafkaConsumer(consumer)
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
