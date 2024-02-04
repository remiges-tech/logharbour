package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"

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

// func createElasticsearchClient() (*logharbour.ElasticsearchClient, error) {
// 	esAddresses := getEnv("ELASTICSEARCH_ADDRESSES", "http://localhost:9200")
// 	esConfig := elasticsearch.Config{
// 		Addresses: strings.Split(esAddresses, ","),
// 	}
// 	return logharbour.NewElasticsearchClient(esConfig)
// }

// func createKafkaConsumer(brokers []string, topic string, handler logharbour.MessageHandler) (logharbour.Consumer, error) {
// 	return logharbour.NewConsumer(brokers, topic, handler)
// }

func startKafkaConsumer(consumer logharbour.Consumer) (<-chan error, error) {
	return consumer.Start(10)
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

func main() {
	// Define flags with environment variables as default values
	esAddresses := flag.String("esAddresses", getEnv("ELASTICSEARCH_ADDRESSES", "http://localhost:9200"), "Elasticsearch addresses (comma-separated)")
	kafkaBrokers := flag.String("kafkaBrokers", getEnv("KAFKA_BROKERS", "localhost:9092"), "Kafka brokers (comma-separated)")
	kafkaTopic := flag.String("kafkaTopic", getEnv("KAFKA_TOPIC", "log_topic"), "Kafka topic")
	// brokers := strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	// topic := getEnv("KAFKA_TOPIC", "log_topic")

	log.Printf("Elasticsearch Addresses: %s", *esAddresses)
	log.Printf("Kafka Brokers: %s", *kafkaBrokers)
	log.Printf("Kafka Topic: %s", *kafkaTopic)

	esClient, err := createElasticsearchClient(*esAddresses)
	// esClient, err := createElasticsearchClient()
	if err != nil {
		log.Fatalf("Error creating the Elasticsearch client: %s", err)
	}

	handler := func(messages []*sarama.ConsumerMessage) error {
		for _, message := range messages {
			log.Printf("Received message from topic %s: %s", message.Topic, string(message.Value))
			err := esClient.Write("logs", string(message.Key), string(message.Value))
			if err != nil {
				log.Printf("Failed to write message to Elasticsearch: %v", err)
				return err
			}
		}
		return nil
	}

	consumer, err := createKafkaConsumer(*kafkaBrokers, *kafkaTopic, handler)
	if err != nil {
		log.Fatalln("Failed to create consumer: ", err)
	}

	errs, err := startKafkaConsumer(consumer)
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

func createKafkaConsumer(brokers, topic string, handler logharbour.MessageHandler) (logharbour.Consumer, error) {
	return logharbour.NewConsumer(strings.Split(brokers, ","), topic, handler)
}
