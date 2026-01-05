// +build ignore

// Manual test program to send various message types to Kafka for DLQ testing.
// Run: go run dlq_test_producer.go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/IBM/sarama"
)

func main() {
	brokers := []string{"localhost:9092"}
	topic := "log_topic"

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		log.Fatalf("Failed to create producer: %v", err)
	}
	defer producer.Close()

	testMessages := []struct {
		name  string
		value string
	}{
		{
			name:  "Valid message",
			value: `{"id":"valid-001","app":"test-app","type":"A","pri":"Info","when":"2024-01-15T10:00:00Z","msg":"valid log message"}`,
		},
		{
			name:  "Invalid JSON",
			value: `this is not valid json at all`,
		},
		{
			name:  "Missing ID field",
			value: `{"app":"test-app","type":"A","msg":"missing id"}`,
		},
		{
			name:  "Empty ID field",
			value: `{"id":"","app":"test-app","msg":"empty id"}`,
		},
		{
			name:  "Another valid message",
			value: `{"id":"valid-002","app":"test-app","type":"A","pri":"Warn","when":"2024-01-15T10:01:00Z","msg":"another valid message"}`,
		},
	}

	fmt.Printf("Sending %d test messages to topic '%s'\n\n", len(testMessages), topic)

	for i, tm := range testMessages {
		msg := &sarama.ProducerMessage{
			Topic: topic,
			Value: sarama.StringEncoder(tm.value),
		}

		partition, offset, err := producer.SendMessage(msg)
		if err != nil {
			log.Printf("[%d] Failed to send '%s': %v", i+1, tm.name, err)
			continue
		}

		fmt.Printf("[%d] %s\n", i+1, tm.name)
		fmt.Printf("    Partition: %d, Offset: %d\n", partition, offset)
		fmt.Printf("    Value: %s\n\n", tm.value)

		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("Done. Check consumer logs and DLQ topic for results.")
	fmt.Println("\nTo view DLQ messages:")
	fmt.Println("  docker exec -it kafka kafka-console-consumer.sh \\")
	fmt.Println("    --bootstrap-server localhost:9092 \\")
	fmt.Println("    --topic log_topic_dlq --from-beginning --property print.headers=true")
}
