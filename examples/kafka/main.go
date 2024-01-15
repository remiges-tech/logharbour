package main

import (
	"log"
	"os"
	"time"

	"github.com/remiges-tech/logharbour/logharbour"
)

func main() {
	// Create a Kafka writer
	// Define your Kafka configuration
	kafkaConfig := logharbour.KafkaConfig{
		Brokers: []string{"localhost:9092"}, // replace with your Kafka brokers
		Topic:   "log_topic",                // replace with your Kafka topic
	}

	// Define the maximum number of connections in the pool
	poolSize := 10

	kafkaWriter, err := logharbour.NewKafkaWriter(kafkaConfig, logharbour.WithPoolSize(poolSize))
	if err != nil {
		log.Fatalf("Failed to create Kafka writer: %v", err)
	}

	// Use kafkaWriter with your LogHarbour instances

	// Create a fallback writer that uses stdout as the fallback.
	fallbackWriter := logharbour.NewFallbackWriter(kafkaWriter, os.Stdout)

	// Create a logger context with the default priority.
	lctx := logharbour.NewLoggerContext(logharbour.Info)

	// Initialize the logger with the context, validator, default priority, and fallback writer.
	logger := logharbour.NewLoggerWithFallback(lctx, "MyApp", fallbackWriter)

	// Log a message every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Initialize a counter for the serial number
	var serialNumber int

	for range ticker.C {
		// Increment the serial number
		serialNumber++

		// Add the serial number to the log data
		data := map[string]interface{}{
			"username":     "john",
			"serialNumber": serialNumber,
		}

		logger.LogActivity("User logged in", data)
	}

	// Close the Kafka writer when done
	if err := kafkaWriter.Close(); err != nil {
		log.Fatalf("failed to close Kafka writer: %v", err)
	}
}
