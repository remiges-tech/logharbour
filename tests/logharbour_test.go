package main

import (
	"bytes"
	"errors"
	"sync"
	"testing"

	"github.com/remiges-tech/logharbour/logharbour"
)

// mockWriter is a simple in-memory writer to capture log outputs for testing.
type mockWriter struct {
	bytes.Buffer
}

func (m *mockWriter) Close() error {
	return nil
}

// TestPriorityLevelPrinting checks that a more verbose priority level prints all less verbose messages.
func TestPriorityLevelPrinting(t *testing.T) {
	// Create a new mock writer to capture log outputs.
	output := new(mockWriter)

	// Create a fallback writer that uses the mock writer for both primary and fallback outputs.
	fallbackWriter := logharbour.NewFallbackWriter(output, output)

	// Create a logger context with the default priority.
	lctx := logharbour.NewLoggerContext(logharbour.Info)

	lctx.SetDebugMode(true)

	// Initialize the logger with a basic context and validator, and a test priority level.
	logger := logharbour.NewLogger(lctx, "TestApp", fallbackWriter)

	// log a message at Debug1 level.
	logger.LogDebug("Debug1 message", logharbour.DebugInfo{})

	// Change logger priority to a more verbose level (Debug2).
	lctx.ChangeMinLogPriority(logharbour.Debug2)

	// log another message at Debug2 level.
	logger.LogDebug("Debug2 message", logharbour.DebugInfo{})

	// Check if both messages are present in the output.
	outputStr := output.String()
	if !bytes.Contains(output.Bytes(), []byte("Debug1 message")) {
		t.Errorf("Expected Debug1 message to be logged, got: %s", outputStr)
	}
	if !bytes.Contains(output.Bytes(), []byte("Debug2 message")) {
		t.Errorf("Expected Debug2 message to be logged, got: %s", outputStr)
	}
}

// mockFailingWriter is a writer that fails when attempting to write to it.
type mockFailingWriter struct {
	fail bool // Determines if the writer should fail.
}

func (mfw *mockFailingWriter) Write(p []byte) (n int, err error) {
	if mfw.fail {
		return 0, errors.New("primary writer failed")
	}
	return len(p), nil
}

// TestFallbackWriter verifies that the fallback writer is used when the primary writer fails.
func TestFallbackWriter(t *testing.T) {
	// Create a primary writer that is set to fail.
	primary := &mockFailingWriter{fail: true}
	// Create a fallback writer that will capture the output.
	fallback := &bytes.Buffer{}

	// Create a new FallbackWriter with the primary and fallback writers.
	fw := logharbour.NewFallbackWriter(primary, fallback)

	// Write a message using the FallbackWriter.
	message := []byte("test message")
	_, err := fw.Write(message)
	if err != nil {
		t.Errorf("Did not expect an error when fallback writer succeeds, but got: %v", err)
	}

	// Check if the fallback writer has captured the message.
	if fallback.String() != string(message) {
		t.Errorf("Expected fallback writer to capture the message, got: %s", fallback.String())
	}
}

// Helper function to create a default LoggerContext for tests
func newTestLoggerContext() *logharbour.LoggerContext {
	return &logharbour.LoggerContext{}
}

func TestLoggerContext_SetDebugMode(t *testing.T) {
	lc := newTestLoggerContext()
	var wg sync.WaitGroup

	// Number of goroutines to spawn
	numGoroutines := 100

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			lc.SetDebugMode(true)
			if !lc.IsDebugModeSet() {
				t.Errorf("Expected debug mode to be true")
			}
		}()
	}
	wg.Wait()

	// Final check to ensure debugMode is true after all goroutines have run
	if !lc.IsDebugModeSet() {
		t.Errorf("Expected final debug mode to be true")
	}
}
