package main

import (
	"context"
	"fmt"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

func main() {
	ctx := context.Background()

	// Configure the Kafka client
	client, err := kgo.NewClient(
		kgo.SeedBrokers("localhost:9092"), // Replace with your Kafka broker address
	)
	if err != nil {
		panic("failed to create client: " + err.Error())
	}
	defer client.Close()

	topic := "your_topic"
	for {
		// Construct the message
		key := []byte("key")
		value := []byte(fmt.Sprintf("Message at %s", time.Now()))

		// Produce the message
		err = client.ProduceSync(ctx, &kgo.Record{
			Topic: topic,
			Key:   key,
			Value: value,
		}).FirstErr()
		if err != nil {
			fmt.Printf("failed to produce message: %v\n", err)
		} else {
			fmt.Printf("Produced message to topic %s: %s\n", topic, value)
		}

		// Wait for 2 seconds
		time.Sleep(2 * time.Second)
	}
}
