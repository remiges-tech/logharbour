package logharbour

import (
	"strings"
	"testing"
)

func TestGetDebugInfo(t *testing.T) {
	_, _, funcName, stackTrace := GetDebugInfo(1)

	// Check if the function name is correctly identified
	if !strings.Contains(funcName, "TestGetDebugInfo") {
		t.Errorf("Expected function name to contain 'TestGetDebugInfo', got %v", funcName)
	}

	// Stack trace should contain some content
	if len(stackTrace) == 0 {
		t.Errorf("Expected stack trace to have content, got empty string")
	}
}

func TestGetDebugInfoNested(t *testing.T) {
	nestedFunction(t)

}

func nestedFunction(t *testing.T) {
	_, _, funcName, stackTrace := GetDebugInfo(1)

	if !strings.Contains(funcName, "nestedFunction") {
		t.Errorf("Expected function name to contain 'nestedFunction', got %v", funcName)
	}

	// Stack trace should contain some content
	if len(stackTrace) == 0 {
		t.Errorf("Expected stack trace to have content, got empty string")
	}
}

func TestGetDebugInfoNestedInline(t *testing.T) {
	nestedFunction := func() {
		_, _, funcName, stackTrace := GetDebugInfo(1)

		if !strings.Contains(funcName, "TestGetDebugInfoNestedInline.func1") {
			t.Errorf("Expected function name to contain 'TestGetDebugInfoNestedInline.func1', got %v", funcName)
		}

		// Stack trace should contain some content
		if len(stackTrace) == 0 {
			t.Errorf("Expected stack trace to have content, got empty string")
		}
	}

	nestedFunction()
}
