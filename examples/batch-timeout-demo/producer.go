package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/remiges-tech/logharbour/logharbour"
)

func main() {
	fmt.Println("=== Demo Producer for Batch Timeout Test ===")
	fmt.Println("This will send 20 messages slowly to demonstrate timeout")
	fmt.Println("")

	// Get Kafka config from environment
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "kafka:9092"
	}
	topic := os.Getenv("KAFKA_TOPIC")
	if topic == "" {
		topic = "log_topic"
	}

	// Create Kafka writer
	kafkaConfig := logharbour.KafkaConfig{
		Brokers: []string{brokers},
		Topic:   topic,
	}

	kafkaWriter, err := logharbour.NewKafkaWriter(kafkaConfig)
	if err != nil {
		log.Fatalf("Failed to create Kafka writer: %v", err)
	}
	defer kafkaWriter.Close()

	// Create logger without fallback to ensure messages go to Kafka
	lctx := logharbour.NewLoggerContext(logharbour.Info)
	logger := logharbour.NewLogger(lctx, "BatchTimeoutDemo", kafkaWriter)

	fmt.Printf("Sending messages to Kafka at %s, topic: %s\n", brokers, topic)
	fmt.Println("Batch size is 100, timeout is 10s")
	fmt.Println("Sending 1 message every 2 seconds...")
	fmt.Println("")

	// Send 20 messages slowly
	for i := 1; i <= 20; i++ {
		start := time.Now()
		
		// Send log message
		logger.LogActivity(fmt.Sprintf("Test message %d", i), map[string]interface{}{
			"message_number": i,
			"timestamp":      start.Format(time.RFC3339),
		})

		fmt.Printf("[%s] Sent message %d\n", start.Format("15:04:05"), i)
		
		// Check if we're at a 10-second boundary
		if i%5 == 0 {
			fmt.Printf("  --> Batch timeout should trigger around now (10s elapsed)\n")
		}
		
		// Wait 2 seconds before next message
		if i < 20 {
			time.Sleep(2 * time.Second)
		}
	}

	fmt.Println("\nAll messages sent!")
	fmt.Println("Waiting 5 seconds for Kafka writer to flush...")
	time.Sleep(5 * time.Second)
	
	fmt.Println("\nExpected result:")
	fmt.Println("- 4 batches processed (at 10s, 20s, 30s, 40s)")
	fmt.Println("- Each batch contains 5 messages")
	fmt.Println("- Consumer logs show 'Batch timeout reached' for each batch")
}