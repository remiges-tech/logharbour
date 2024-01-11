package main

import (
	"log"
	"os"
	"time"

	"github.com/remiges-tech/logharbour/logharbour"
)

func main() {
	// Create a Kafka writer
	sw, err := logharbour.NewKafkaWriter(logharbour.KafkaConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "log_topic",
	})
	if err != nil {
		log.Fatalf("unable to create Kafka writer: %v", err)
	}

	// Create a fallback writer that uses stdout as the fallback.
	fallbackWriter := logharbour.NewFallbackWriter(sw, os.Stdout)

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
	if err := sw.Close(); err != nil {
		log.Fatalf("failed to close Kafka writer: %v", err)
	}
}
