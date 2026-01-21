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
	"sync/atomic"
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
	lctx := NewLoggerContext(Info)
	logger := NewLogger(lctx, "myApp", &bytes.Buffer{})
	logger1 := logger.WithModule("module1")
	logger2 := logger1.WithModule("module2")

	lctx.ChangeMinLogPriority(Warn)

	t.Run("Priority propagation to logger1", func(t *testing.T) {
		minPri := LogPriority(atomic.LoadInt32(&logger1.context.minLogPriority))
		if minPri != Warn {
			t.Errorf("expected logger1 priority to be %v, got %v", Warn, minPri)
		}
	})

	t.Run("Priority propagation to logger2", func(t *testing.T) {
		minPri := LogPriority(atomic.LoadInt32(&logger2.context.minLogPriority))
		if minPri != Warn {
			t.Errorf("expected logger2 priority to be %v, got %v", Warn, minPri)
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
	changeInfo = changeInfo.AddChange("email", "oldEmail@example.com", "newEmail@example.com")

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
	changeData := loggedEntry.Data.ChangeData // Access ChangeInfo data directly

	// Check the entity and operation
	if changeData.Entity != "User" || changeData.Op != "Update" {
		t.Errorf("Expected entity 'User' and operation 'Update', got entity '%s' and operation '%s'", changeData.Entity, changeData.Op)
	}

	// Check the changes
	if len(changeData.Changes) == 0 {
		t.Fatalf("Expected changes, got %d", len(changeData.Changes))
	}

	// Assuming we know there's only one change for simplicity
	firstChange := changeData.Changes[0]

	// Check the old and new email values
	if firstChange.Field != "email" || firstChange.OldVal != "oldEmail@example.com" || firstChange.NewVal != "newEmail@example.com" {
		t.Errorf("Expected email change from 'oldEmail@example.com' to 'newEmail@example.com', got field '%s', old value '%s', new value '%s'", firstChange.Field, firstChange.OldVal, firstChange.NewVal)
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
		{"String", "test string", "test string"}, // Adjusted expected value
		{"Int", 42, "42"},                        // Adjusted expected value
		{"Bool", true, "true"},                   // Adjusted expected value
		{"Map", map[string]any{"key": "value"}, `{"key":"value"}`},
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

			// Access DebugData field directly
			dataField := logEntry.Data.DebugData
			if dataField == nil {
				t.Fatalf("Expected 'DebugData' field to be non-nil")
			}

			// Compare the actual data with the expected data
			if fmt.Sprintf("%v", dataField.Data) != td.ExpectedContains {
				t.Errorf("Expected 'data' field to be %s, got %v", td.ExpectedContains, dataField.Data)
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

	dataFieldMap := logEntryMap.Data.ActivityData
	if dataFieldMap == "" {
		t.Fatalf("Expected 'ActivityData' field to be non-empty")
	}

	expectedDataJSONMap := `{"key1":"value1","key2":42,"key3":true}`
	if dataFieldMap != expectedDataJSONMap {
		t.Errorf("Expected 'ActivityData' field to be %s, got %s", expectedDataJSONMap, dataFieldMap)
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

	dataFieldString := logEntryString.Data.ActivityData
	if dataFieldString == "" {
		t.Fatalf("Expected 'ActivityData' field to be non-empty")
	}

	expectedDataString := "Simple activity data" // Compare directly as a plain string
	if dataFieldString != expectedDataString {
		t.Errorf("Expected 'ActivityData' field to be %s, got %s", expectedDataString, dataFieldString)
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
	changeInfo = changeInfo.AddChange("email", "oldEmail@example.com", "john@example.com")
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

func TestLogger_Logf(t *testing.T) {
	var buf bytes.Buffer
	context := NewLoggerContext(Info)
	logger := NewLogger(context, "TestApp", &buf)

	logger.Logf("This is a formatted log message with value: %d", 42)

	var logEntry LogEntry
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	if err != nil {
		t.Fatalf("Failed to unmarshal logged message: %v", err)
	}

	expectedMsg := "This is a formatted log message with value: 42"
	if logEntry.Msg != expectedMsg {
		t.Errorf("expected msg to be %q, got %q", expectedMsg, logEntry.Msg)
	}
}

func TestLogging_SkippedWhenFiltered(t *testing.T) {
	tests := []struct {
		name       string
		logFunc    func(l *Logger)
		debugMode  bool
		loggerPri  LogPriority
		contextPri LogPriority
	}{
		{
			name: "Log skipped when priority below minimum",
			logFunc: func(l *Logger) {
				l.Log("should not appear")
			},
			loggerPri:  Info,
			contextPri: Warn,
		},
		{
			name: "Logf skipped when priority below minimum",
			logFunc: func(l *Logger) {
				l.Logf("should not appear: %d", 42)
			},
			loggerPri:  Info,
			contextPri: Warn,
		},
		{
			name: "LogActivity skipped when priority below minimum",
			logFunc: func(l *Logger) {
				l.LogActivity("should not appear", map[string]any{"key": "value"})
			},
			loggerPri:  Info,
			contextPri: Warn,
		},
		{
			name: "LogActivityf skipped when priority below minimum",
			logFunc: func(l *Logger) {
				l.LogActivityf(map[string]any{"key": "value"}, "user %s action", "john")
			},
			loggerPri:  Info,
			contextPri: Warn,
		},
		{
			name: "LogDataChange skipped when priority below minimum",
			logFunc: func(l *Logger) {
				changeInfo := NewChangeInfo("Entity", "Update")
				changeInfo.AddChange("field", "old", "new")
				l.LogDataChange("should not appear", *changeInfo)
			},
			loggerPri:  Info,
			contextPri: Warn,
		},
		{
			name: "LogDataChangef skipped when priority below minimum",
			logFunc: func(l *Logger) {
				changeInfo := NewChangeInfo("Entity", "Update")
				changeInfo.AddChange("field", "old", "new")
				l.LogDataChangef(*changeInfo, "change %d", 42)
			},
			loggerPri:  Info,
			contextPri: Warn,
		},
		{
			name: "LogDebug skipped when debug mode off",
			logFunc: func(l *Logger) {
				l.LogDebug("should not appear", map[string]any{"key": "value"})
			},
			debugMode:  false,
			loggerPri:  Debug0,
			contextPri: Debug0,
		},
		{
			name: "LogDebug skipped when priority below minimum",
			logFunc: func(l *Logger) {
				l.LogDebug("should not appear", map[string]any{"key": "value"})
			},
			debugMode:  true,
			loggerPri:  Info,
			contextPri: Warn,
		},
		{
			name: "LogDebugf skipped when debug mode off",
			logFunc: func(l *Logger) {
				l.LogDebugf(map[string]any{"key": "value"}, "debug %s", "message")
			},
			debugMode:  false,
			loggerPri:  Debug0,
			contextPri: Debug0,
		},
		{
			name: "LogDebugf skipped when priority below minimum",
			logFunc: func(l *Logger) {
				l.LogDebugf(map[string]any{"key": "value"}, "debug %s", "message")
			},
			debugMode:  true,
			loggerPri:  Info,
			contextPri: Warn,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			ctx := NewLoggerContext(tt.contextPri)
			ctx.SetDebugMode(tt.debugMode)
			logger := NewLogger(ctx, "TestApp", &buf).WithPriority(tt.loggerPri)

			tt.logFunc(logger)

			if buf.Len() > 0 {
				t.Errorf("expected no output when filtered, got: %s", buf.String())
			}
		})
	}
}

func TestLogDebugf(t *testing.T) {
	var buf bytes.Buffer
	ctx := NewLoggerContext(Debug0)
	ctx.SetDebugMode(true)
	logger := NewLogger(ctx, "TestApp", &buf)

	logger.LogDebugf(map[string]any{"session": "abc123"}, "user %s logged in", "john")

	var logEntry LogEntry
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	if err != nil {
		t.Fatalf("failed to unmarshal logged message: %v", err)
	}

	expectedMsg := "user john logged in"
	if logEntry.Msg != expectedMsg {
		t.Errorf("expected msg to be %q, got %q", expectedMsg, logEntry.Msg)
	}

	if logEntry.Type != Debug {
		t.Errorf("expected type to be Debug, got %v", logEntry.Type)
	}

	if logEntry.Data.DebugData == nil {
		t.Fatalf("expected DebugData to be non-nil")
	}
}

func TestLogActivityf(t *testing.T) {
	var buf bytes.Buffer
	ctx := NewLoggerContext(Info)
	logger := NewLogger(ctx, "TestApp", &buf)

	activityData := map[string]any{"action": "click", "element": "button"}
	logger.LogActivityf(activityData, "user %s performed action on %s", "john", "homepage")

	var logEntry LogEntry
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	if err != nil {
		t.Fatalf("failed to unmarshal logged message: %v", err)
	}

	expectedMsg := "user john performed action on homepage"
	if logEntry.Msg != expectedMsg {
		t.Errorf("expected msg to be %q, got %q", expectedMsg, logEntry.Msg)
	}

	if logEntry.Type != Activity {
		t.Errorf("expected type to be Activity, got %v", logEntry.Type)
	}
}

func TestLogDataChangef(t *testing.T) {
	var buf bytes.Buffer
	ctx := NewLoggerContext(Info)
	logger := NewLogger(ctx, "TestApp", &buf)

	changeInfo := NewChangeInfo("User", "Update")
	changeInfo.AddChange("email", "old@example.com", "new@example.com")

	logger.LogDataChangef(*changeInfo, "user %d updated profile", 42)

	var logEntry LogEntry
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	if err != nil {
		t.Fatalf("failed to unmarshal logged message: %v", err)
	}

	expectedMsg := "user 42 updated profile"
	if logEntry.Msg != expectedMsg {
		t.Errorf("expected msg to be %q, got %q", expectedMsg, logEntry.Msg)
	}

	if logEntry.Type != Change {
		t.Errorf("expected type to be Change, got %v", logEntry.Type)
	}

	if logEntry.Data.ChangeData == nil {
		t.Fatalf("expected ChangeData to be non-nil")
	}

	if logEntry.Data.ChangeData.Entity != "User" {
		t.Errorf("expected entity to be 'User', got %q", logEntry.Data.ChangeData.Entity)
	}
}

