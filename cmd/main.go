package main

import (
	"os"

	"github.com/ssd532/logdock/logharbour"
)

func main() {
	// Create a fallback writer that uses stdout as the fallback.
	fallbackWriter := logharbour.NewFallbackWriter(os.Stdout, os.Stdout)

	// Initialize the logger with the context, validator, default priority, and fallback writer.
	logger := logharbour.NewLogger("MyApp", fallbackWriter)

	// log an activity entry.
	logger.LogActivity("User logged in", map[string]any{"username": "john"})

	// log a data change entry.
	logger.LogDataChange("User updated profile", logharbour.ChangeInfo{
		Entity:    "User",
		Operation: "Update",
		Changes:   map[string]any{"email": "john@example.com"},
	})

	// log a debug entry.
	logger.LogDebug("Debugging user session", logharbour.DebugInfo{
		Variables: map[string]any{"sessionID": "12345"},
	})
	// Change logger priority at runtime.
	logger.ChangePriority(logharbour.Debug2)

	// log another debug entry with a higher verbosity level.
	logger.LogDebug("Detailed debugging info", logharbour.DebugInfo{
		Variables: map[string]any{"sessionID": "12345", "userID": "john"},
	})

	outerFunction(logger)

}

func innerFunction(logger *logharbour.Logger) {
	// log a debug entry.
	logger.LogDebug("Debugging inner function", logharbour.DebugInfo{
		Variables: map[string]any{"innerVar": "innerValue"},
	})
}

func outerFunction(logger *logharbour.Logger) {
	innerFunction(logger)
}
