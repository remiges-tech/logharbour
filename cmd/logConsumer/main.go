package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/IBM/sarama"
	"github.com/remiges-tech/logharbour/logharbour"
)

func main() {
	brokers := []string{"localhost:9092"}
	topic := "log_topic"

	// Define your message handler
	handler := func(messages []*sarama.ConsumerMessage) error {
		for _, message := range messages {
			// Process each message
			log.Printf("Received message from topic %s: %s", message.Topic, string(message.Value))
		}
		return nil
	}

	// Create a new consumer
	consumer, err := logharbour.NewConsumer(brokers, topic, handler)
	if err != nil {
		log.Fatalln("Failed to create consumer: ", err)
	}

	errs, err := consumer.Start(10)
	if err != nil {
		log.Fatalln("Failed to start consumer: ", err)
	}

	go func() {
		for err := range errs {
			log.Printf("Failed to process batch: %v", err)
			// Decide what to do here: retry, ignore, etc.
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the consumer
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	<-signals

	// Stop the consumer
	if err := consumer.Stop(); err != nil {
		log.Fatalln("Failed to stop consumer: ", err)
	}
}
