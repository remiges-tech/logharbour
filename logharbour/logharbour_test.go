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
	lctx.ChangeMinLogPriority(Warn)

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

func TestLogDataChange(t *testing.T) {
	// Create a buffer to use as the logger's writer
	var buf bytes.Buffer

	// Create a new logger with the buffer as its writer
	logger := NewLogger(NewLoggerContext(Info), "TestApp", &buf)

	// Create a new ChangeInfo and add a change
	changeInfo := NewChangeInfo("User", "Update")
	changeInfo = changeInfo.AddChange(logger, "email", "oldEmail@example.com", "newEmail@example.com")

	// Log the data change
	logger.LogDataChange("User updated profile", *changeInfo)

	// Get the logged message
	loggedMessage := buf.String()

	// Unmarshal the logged message into a LogEntry struct
	var loggedEntry LogEntry
	err := json.Unmarshal([]byte(loggedMessage), &loggedEntry)
	if err != nil {
		t.Fatalf("Failed to unmarshal logged message: %v", err)
	}

	// Check that the logged message contains the expected data
	if loggedEntry.Msg != "User updated profile" {
		t.Errorf("Expected message 'User updated profile', got '%s'", loggedEntry.Msg)
	}

	// Extract and assert the ChangeInfo data from the LogEntry
	changeData, ok := loggedEntry.Data.(map[string]any) // Type assertion to access ChangeInfo data
	if !ok {
		t.Fatalf("Expected ChangeInfo data, got %T", loggedEntry.Data)
	}

	// Check the entity and operation
	if changeData["entity"] != "User" || changeData["op"] != "Update" {
		t.Errorf("Expected entity 'User' and operation 'Update', got entity '%s' and operation '%s'", changeData["entity"], changeData["op"])
	}

	// Check the changes
	changes, ok := changeData["changes"].([]any)
	if !ok || len(changes) == 0 {
		t.Fatalf("Expected changes, got %T", changeData["changes"])
	}

	// Assuming we know there's only one change for simplicity
	firstChange, ok := changes[0].(map[string]any)
	if !ok {
		t.Fatalf("Expected a change detail, got %T", changes[0])
	}

	// Check the old and new email values
	if firstChange["field"] != "email" || firstChange["old_value"] != `"oldEmail@example.com"` || firstChange["new_value"] != `"newEmail@example.com"` {
		t.Errorf("Expected email change from 'oldEmail@example.com' to 'newEmail@example.com', got field '%s', old value '%s', new value '%s'", firstChange["field"], firstChange["old_value"], firstChange["new_value"])
	}
}

// mockWriter captures writes to it, allowing us to inspect the output of the logger.
type mockWriter struct {
	bytes.Buffer
}

func (mw *mockWriter) Write(p []byte) (n int, err error) {
	return mw.Buffer.Write(p)
}

func TestLogDebugWithAnyData(t *testing.T) {
	// Setup
	mockW := &mockWriter{}
	loggerContext := NewLoggerContext(Debug0) // Ensure debug level allows for logging
	logger := NewLogger(loggerContext, "TestApp", mockW)
	loggerContext.SetDebugMode(true) // Enable debug mode

	// Sample data of various types
	testData := []struct {
		Name             string
		Data             any
		ExpectedContains string
	}{
		{"String", "test string", `"\"test string\""`},
		{"Int", 42, `"42"`},
		{"Bool", true, `"true"`},
		{"Map", map[string]any{"key": "value"}, `"{\"key\":\"value\"}"`},
	}

	for _, td := range testData {
		t.Run(td.Name, func(t *testing.T) {
			// Act
			logger.LogDebug("Debug message", td.Data)

			// Assert
			output := mockW.String()
			var logEntry LogEntry
			err := json.Unmarshal([]byte(output), &logEntry)
			if err != nil {
				t.Fatalf("Failed to unmarshal logged message: %v", err)
			}

			dataField, ok := logEntry.Data.(map[string]any)
			if !ok {
				t.Fatalf("Expected 'data' field to be a map, got %T", logEntry.Data)
			}

			dataJSON, err := json.Marshal(dataField["data"])
			fmt.Println("data field:")
			fmt.Println(dataField["data"])
			fmt.Println(dataJSON)
			if err != nil {
				t.Fatalf("Failed to marshal 'data' field: %v", err)
			}

			if string(dataJSON) != td.ExpectedContains {
				t.Errorf("Expected 'data' field to be %s, got %s", td.ExpectedContains, string(dataJSON))
			}

			mockW.Reset()
		})
	}
}

func TestLogActivity(t *testing.T) {
	// Setup
	mockW := &mockWriter{}
	loggerContext := NewLoggerContext(Info)
	logger := NewLogger(loggerContext, "TestApp", mockW)

	// Sample activity data (map)
	activityDataMap := map[string]any{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}

	// Sample activity data (string)
	activityDataString := "Simple activity data"

	// Act
	logger.LogActivity("Activity message (map)", activityDataMap)
	logger.LogActivity("Activity message (string)", activityDataString)

	// Assert
	output := mockW.String()
	logEntries := strings.Split(output, "\n")

	// Assert for activity data (map)
	var logEntryMap LogEntry
	err := json.Unmarshal([]byte(logEntries[0]), &logEntryMap)
	if err != nil {
		t.Fatalf("Failed to unmarshal logged message: %v", err)
	}

	if logEntryMap.Msg != "Activity message (map)" {
		t.Errorf("Expected message to be 'Activity message (map)', got '%s'", logEntryMap.Msg)
	}

	if logEntryMap.Type != Activity {
		t.Errorf("Expected type to be 'Activity', got '%s'", logEntryMap.Type)
	}

	dataFieldMap, ok := logEntryMap.Data.(string)
	if !ok {
		t.Fatalf("Expected 'data' field to be a string, got %T", logEntryMap.Data)
	}

	expectedDataJSONMap := `{"key1":"value1","key2":42,"key3":true}`
	if dataFieldMap != expectedDataJSONMap {
		t.Errorf("Expected 'data' field to be %s, got %s", expectedDataJSONMap, dataFieldMap)
	}

	// Assert for activity data (string)
	var logEntryString LogEntry
	err = json.Unmarshal([]byte(logEntries[1]), &logEntryString)
	if err != nil {
		t.Fatalf("Failed to unmarshal logged message: %v", err)
	}

	if logEntryString.Msg != "Activity message (string)" {
		t.Errorf("Expected message to be 'Activity message (string)', got '%s'", logEntryString.Msg)
	}

	if logEntryString.Type != Activity {
		t.Errorf("Expected type to be 'Activity', got '%s'", logEntryString.Type)
	}

	dataFieldString, ok := logEntryString.Data.(string)
	if !ok {
		t.Fatalf("Expected 'data' field to be a string, got %T", logEntryString.Data)
	}

	expectedDataString := `"Simple activity data"`
	if dataFieldString != expectedDataString {
		t.Errorf("Expected 'data' field to be %s, got %s", expectedDataString, dataFieldString)
	}
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
	changeInfo := NewChangeInfo("User", "Update")
	changeInfo = changeInfo.AddChange(logger, "email", "oldEmail@example.com", "john@example.com")
	if err != nil {
		// Handle the error
		fmt.Println("Error adding change:", err)
		return
	}
	logger.LogDataChange("User updated profile", *changeInfo)

	// Change logger priority at runtime.
	lctx.ChangeMinLogPriority(Debug2)

	// Log a debug entry.
	logger.LogDebug("Debugging user session", map[string]any{"sessionID": "12345"})
}
