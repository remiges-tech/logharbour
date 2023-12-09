package logharbour

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"testing"
)

type FailWriter struct{}

func (fw *FailWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("failed to write")
}

func TestFallbackWriter(t *testing.T) {
	// Create a custom writer that always fails
	failWriter := FailWriter{}

	// Create a buffer to act as the fallback writer
	fallbackBuffer := &bytes.Buffer{}

	lctx := NewLoggerContext(Info)

	// Create a logger with a FallbackWriter
	logger := NewLoggerWithFallback(lctx, "testApp", NewFallbackWriter(&failWriter, fallbackBuffer))

	// Create a log message
	message := "test message"

	// Call the LogActivity function
	logger.LogActivity(message, nil)

	// Check if the log entry is written to the fallback writer
	if !bytes.Contains(fallbackBuffer.Bytes(), []byte(message)) {
		t.Errorf("Expected log entry to be written to the fallback writer")
	}
}

func TestFallbackWriter_Stderr(t *testing.T) {
	// Create a custom writer that always fails
	failWriter := FailWriter{}

	// Create another custom writer that also always fails
	failFallbackWriter := FailWriter{}

	// Create a buffer to act as the fallback writer
	lctx := NewLoggerContext(Info)

	// Create a logger with a FallbackWriter
	logger := NewLoggerWithFallback(lctx, "testApp", NewFallbackWriter(&failWriter, &failFallbackWriter))

	// Create a log message
	message := "test message"

	// Redirect stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Call the LogActivity function
	logger.LogActivity(message, nil)

	// Restore stderr
	w.Close()
	os.Stderr = oldStderr

	// Read from the pipe
	out, _ := io.ReadAll(r)

	// Check if the log entry is written to stderr
	if !bytes.Contains(out, []byte(message)) {
		t.Errorf("Expected log entry to be written to stderr")
	}

	// Check if the error message is written to stderr
	expectedError := "failed to write"
	if !bytes.Contains(out, []byte(expectedError)) {
		t.Errorf("Expected error message to be written to stderr")
	}
}

func TestLog(t *testing.T) {
	lctx := NewLoggerContext(Info)
	var buf bytes.Buffer
	logger := NewLogger(lctx, "testApp", &buf)

	message := "test message"
	logger.Log(message)

	logged := buf.String()
	if !strings.Contains(logged, message) {
		t.Errorf("Expected '%s' to be logged. Got: %s", message, logged)
	}

	var loggedEntry LogEntry
	err := json.Unmarshal([]byte(logged), &loggedEntry)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if loggedEntry.Type != Activity {
		t.Errorf("Expected Type to be '%s'. Got: '%s'", Activity.String(), loggedEntry.Type.String())
	}
}

func TestErrMethods(t *testing.T) {
	lctx := NewLoggerContext(Info)
	// Create a new logger
	logger := NewLogger(lctx, "testApp", nil)

	// Test Error method
	err := errors.New("test error")
	logger = logger.Error(err)
	if logger.err != err.Error() {
		t.Errorf("Expected error to be '%s', got '%s'", err.Error(), logger.err)
	}

	// Create an error chain
	err1 := errors.New("error 1")
	err2 := fmt.Errorf("error 2: %w", err1)
	err3 := fmt.Errorf("error 3: %w", err2)

	// Use the ErrorChain method
	logger = logger.Error(err3)

	// The expected error chain is 'error 3: error 2: error 1'
	expectedErrChain := "error 3: error 2: error 1"

	// Check if the error chain in the logger is as expected
	if logger.err != expectedErrChain {
		t.Errorf("Expected error chain to be '%s', got '%s'", expectedErrChain, logger.err)
	}
}

func TestLoggerContextPriorityPropagation(t *testing.T) {
	// Create a shared LoggerContext with default priority
	lctx := NewLoggerContext(Info)

	// Create a new Logger with the shared context
	logger := NewLogger(lctx, "myApp", &bytes.Buffer{})

	// Create new loggers using WithModule
	logger1 := logger.WithModule("module1")
	logger2 := logger1.WithModule("module2")

	// Change the priority in the original context
	logger.ChangeMinLogPriority(Warn)

	// Check that the priority change propagated to all loggers
	t.Run("Priority propagation to logger1", func(t *testing.T) {
		if logger1.context.minLogPriority != Warn {
			t.Errorf("Expected logger1 priority to be %v, got %v", Warn, logger1.context.minLogPriority)
		}
	})

	t.Run("Priority propagation to logger2", func(t *testing.T) {
		if logger2.context.minLogPriority != Warn {
			t.Errorf("Expected logger2 priority to be %v, got %v", Warn, logger2.context.minLogPriority)
		}
	})
}

// Example of using With prefixed methods to set various fields of the logger.
func Example() {
	// Open a file for logging.
	file, err := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Create a fallback writer that uses the file as the primary writer and stdout as the fallback.
	fallbackWriter := NewFallbackWriter(file, os.Stdout)

	// Create a logger context with the default priority.
	lctx := NewLoggerContext(Info)

	// Initialize the logger.
	logger := NewLogger(lctx, "MyApp", fallbackWriter)

	// Create a new logger with various fields set.
	logger = logger.WithModule("Module1").
		WithWho("John Doe").
		WithStatus(Success).
		WithRemoteIP("192.168.1.1")

	// Use the new logger to log an activity.
	logger.LogActivity("User logged in", map[string]any{"username": "john"})

	// Log a data change entry.
	logger.LogDataChange("User updated profile", ChangeInfo{
		Entity:    "User",
		Operation: "Update",
		Changes:   map[string]any{"email": "john@example.com"},
	})

	// Change logger priority at runtime.
	logger.ChangeMinLogPriority(Debug2)

	// Log a debug entry.
	logger.LogDebug("Debugging user session", DebugInfo{
		Variables: map[string]any{"sessionID": "12345"},
	})

	//
	// {
	//     "app_name": "MyApp",
	//     "module": "Module1",
	//     "priority": "Info",
	//     "who": "John Doe",
	//     "status": 1,
	//     "remote_ip": "192.168.1.1",
	//     "type": "Activity",
	//     "message": "User logged in",
	//     "data": {
	//         "username": "john"
	//     }
	// }
	// {
	//     "app_name": "MyApp",
	//     "module": "Module1",
	//     "priority": "Info",
	//     "who": "John Doe",
	//     "status": 1,
	//     "remote_ip": "192.168.1.1",
	//     "type": "Change",
	//     "message": "User updated profile",
	//     "data": {
	//         "entity": "User",
	//         "operation": "Update",
	//         "changes": {
	//             "email": "john@example.com"
	//         }
	//     }
	// }
	// {
	//     "app_name": "MyApp",
	//     "module": "Module1",
	//     "priority": "Debug2",
	//     "who": "John Doe",
	//     "status": 1,
	//     "remote_ip": "192.168.1.1",
	//     "type": "Debug",
	//     "message": "Debugging user session",
	//     "data": {
	//         "variables": {
	//             "sessionID": "12345"
	//         },
	//         "fileName": "main.go",
	//         "lineNumber": 30,
	//         "functionName": "main",
	//         "stackTrace": "...",
	//         "pid": 1234,
	//         "runtime": "go1.15.6"
	//     }
	// }

}
