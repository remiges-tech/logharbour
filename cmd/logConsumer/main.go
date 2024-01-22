package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/IBM/sarama"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/remiges-tech/logharbour/logharbour"
)

func createElasticsearchClient() (*logharbour.ElasticsearchClient, error) {
	esConfig := elasticsearch.Config{
		Addresses: []string{"http://localhost:9200"},
	}
	return logharbour.NewElasticsearchClient(esConfig)
}

func createKafkaConsumer(brokers []string, topic string, handler logharbour.MessageHandler) (logharbour.Consumer, error) {
	return logharbour.NewConsumer(brokers, topic, handler)
}

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
	brokers := []string{"localhost:9092"}
	topic := "log_topic"

	esClient, err := createElasticsearchClient()
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

	consumer, err := createKafkaConsumer(brokers, topic, handler)
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
