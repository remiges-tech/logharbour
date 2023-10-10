// sample Kafka consumer to consume log messages from Kafka
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func kafkaConsumer() {
	fmt.Println("starting consumer")
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "0.0.0.0:9092",
		"group.id":          "logGroup",
		"auto.offset.reset": "earliest"})

	if err != nil {
		fmt.Printf("Failed to create consumer: %s", err)
		panic(err)
	}
	fmt.Println("consumer started")
	topic := "logHarbour"
	err = consumer.SubscribeTopics([]string{topic}, nil)
	if err != nil {
		panic(err)
	}
	// Set up a channel for handling Ctrl-C, etc
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	run := true
	for run {
		select {
		case sig := <-sigchan:
			fmt.Printf("Caught signal %v: terminating\n", sig)
			run = false
		default:
			ev, err := consumer.ReadMessage(10 * time.Millisecond)
			if err != nil {
				// Errors are informational and automatically handled by the consumer
				continue
			}
			fmt.Printf("========>>>Consumed event from topic %s: key = %-10s value = %s\n",
				*ev.TopicPartition.Topic, string(ev.Key), string(ev.Value))
		}
	}

	consumer.Close()
}
