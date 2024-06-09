package main

import (
	"os"

	"github.com/remiges-tech/logharbour/logharbour"
)

func main() {
	// Create a fallback writer that uses stdout as the fallback.
	fallbackWriter := logharbour.NewFallbackWriter(os.Stdout, os.Stdout)

	// Create a logger context with the default priority.
	lctx := logharbour.NewLoggerContext(logharbour.Info)

	// Initialize the logger with the context, validator, default priority, and fallback writer.
	logger := logharbour.NewLogger(lctx, "MyApp", fallbackWriter)

	// log an activity entry.
	logger.LogActivity("User logged in", map[string]any{"username": "john"})

	// log a data change entry.
	// log a data change entry.
	logger.LogDataChange("User updated profile",
		*logharbour.NewChangeInfo("User", "Update").
			AddChange("email", "oldEmail@example.com", "john@example.com"))

	// log a debug entry.
	logger.LogDebug("Debugging user session", map[string]any{"sessionID": "12345"})

	// Change logger priority at runtime.
	lctx.ChangeMinLogPriority(logharbour.Debug2)

	// log another debug entry with a higher verbosity level.
	logger.LogDebug("Detailed debugging info", map[string]any{"sessionID": "12345", "userID": "john"})

	logger.Debug0().LogActivity("debug0 test", nil)

	outerFunction(logger)

}

func innerFunction(logger *logharbour.Logger) {
	// log a debug entry.
	logger.LogDebug("Debugging inner function", map[string]any{"innerVar": "innerValue"})
}

func outerFunction(logger *logharbour.Logger) {
	innerFunction(logger)
}
