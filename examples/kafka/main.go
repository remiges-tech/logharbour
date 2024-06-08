package main

import (
	"fmt"
	"log"
	"math/rand"
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

	// Sample data for generating log entries
	users := []string{"john", "jane", "alice", "bob"}
	activities := []string{"logged in", "logged out", "created post", "liked post", "commented on post"}
	changeTypes := []string{"update", "delete", "create"}
	fields := []string{"email", "username", "password", "bio"}

	for i := 0; i < 100_000_000; i++ {
		// Increment the serial number
		serialNumber++

		// Generate random values for log entries
		user := users[rand.Intn(len(users))]
		activity := activities[rand.Intn(len(activities))]
		changeType := changeTypes[rand.Intn(len(changeTypes))]
		field := fields[rand.Intn(len(fields))]
		oldValue := fmt.Sprintf("%s@example.com", user)
		newValue := fmt.Sprintf("%s@yahoo.com", user)

		// Log a data change
		if rand.Float32() < 0.3 {
			changeInfo := logharbour.NewChangeInfo(user, changeType)
			changeDetail := logharbour.NewChangeDetail(field, oldValue, newValue)
			changeInfo.Changes = append(changeInfo.Changes, changeDetail)
			logger.LogDataChange(fmt.Sprintf("User %s %s", user, changeType), changeInfo)
		}

		// Log an activity
		if rand.Float32() < 0.5 {
			logger.LogActivity(fmt.Sprintf("User %s %s", user, activity), true)
		}

		// Enable debug mode and log a debug message
		if rand.Float32() < 0.2 {
			lctx.SetDebugMode(true)
			logger.LogDebug("Debug message", map[string]interface{}{
				"user":         user,
				"serialNumber": serialNumber,
			})
		}
		// slee for random time between 1 and 5 seconds
		// time.Sleep(time.Duration(rand.Intn(4)+1) * time.Second)
	}

	// Close the Kafka writer when done
	if err := kafkaWriter.Close(); err != nil {
		log.Fatalf("failed to close Kafka writer: %v", err)
	}
}
