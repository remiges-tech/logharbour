package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/remiges-tech/logharbour/logharbour"
)

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func main() {
	// Define flags with environment variables as default values
	kafkaBrokers := flag.String("kafkaBrokers", getEnv("KAFKA_BROKERS", "localhost:9092"), "Kafka brokers (comma-separated)")
	kafkaTopic := flag.String("kafkaTopic", getEnv("KAFKA_TOPIC", "benchmark_topic"), "Kafka topic")
	nMessages := flag.Int("nMessages", 1000, "Number of messages to send")
	nGoroutines := flag.Int("nGoroutines", 10, "Number of goroutines to use for sending messages")

	flag.Parse()

	// Use the flag values...
	log.Printf("Kafka Brokers: %s", *kafkaBrokers)
	log.Printf("Kafka Topic: %s", *kafkaTopic)
	log.Printf("Number of Messages: %d", *nMessages)
	log.Printf("Number of Goroutines: %d", *nGoroutines)
	// nGoroutines := 10
	// nMessages := 100000
	messagesPerGoroutine := *nMessages / *nGoroutines

	// Initialize Kafka connection pool and LogHarbour logger

	kafkaConfig := logharbour.KafkaConfig{
		Brokers: strings.Split(*kafkaBrokers, ","),
		Topic:   *kafkaTopic,
	}

	// Define the maximum number of connections in the pool
	poolSize := 100

	kafkaWriter, err := logharbour.NewKafkaWriter(kafkaConfig, logharbour.WithPoolSize(poolSize))
	if err != nil {
		panic(fmt.Sprintf("unable to create Kafka writer: %v", err))
	}

	// Assuming a function NewLoggerContext exists in your package
	lctx := logharbour.NewLoggerContext(logharbour.Info)

	// Create a fallback writer that uses stdout as the fallback.
	fallbackWriter := logharbour.NewFallbackWriter(kafkaWriter, os.Stdout)

	var wg sync.WaitGroup
	wg.Add(*nGoroutines)

	// Start measuring time
	startTime := time.Now()

	for i := 0; i < *nGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			appName := fmt.Sprintf("MyApp-%d", id)

			// Assuming a function NewLoggerWithFallback exists in your package
			logger := logharbour.NewLoggerWithFallback(lctx, appName, fallbackWriter)

			for j := 0; j < messagesPerGoroutine; j++ {
				message := fmt.Sprintf("Goroutine %d: message %d", id, j)
				// wait for 1 ms
				// this is the only way i can get around the fact that disk i/o is the bottleneck
				time.Sleep(1 * time.Millisecond)
				logMessage(logger, message)
			}
		}(i)
	}

	wg.Wait()

	// Stop measuring time
	duration := time.Since(startTime)

	// Close Kafka writer when done
	if err := kafkaWriter.Close(); err != nil {
		panic(fmt.Sprintf("failed to close Kafka writer: %v", err))
	}

	// Compute and display metrics
	fmt.Printf("Total execution time: %v\n", duration)
	fmt.Printf("Messages per second: %f\n", float64(*nMessages)/duration.Seconds())
}

func logMessage(logger *logharbour.Logger, message string) {
	// Replace with actual logging logic using LogHarbour
	// Assuming a method LogActivity exists in your Logger
	logger.LogActivity("Log Message", map[string]interface{}{
		"message": message,
	})
}
