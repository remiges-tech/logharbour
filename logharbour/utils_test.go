package logharbour

import (
	"strings"
	"testing"
)

func TestGetDebugInfo(t *testing.T) {
	_, _, _, stackTrace := GetDebugInfo(1)

	if !strings.Contains(stackTrace, "TestGetDebugInfo") {
		t.Errorf("Expected stack trace to contain 'TestGetDebugInfo', got %v", stackTrace)
	}
}

func TestGetDebugInfoNested(t *testing.T) {
	nestedFunction(t)

}

func nestedFunction(t *testing.T) {
	_, _, _, stackTrace := GetDebugInfo(1)

	if !strings.Contains(stackTrace, "nestedFunction") {
		t.Errorf("Expected stack trace to contain 'nestedFunction', got %v", stackTrace)
	}
}

func TestGetDebugInfoNestedInline(t *testing.T) {
	nestedFunction := func() {
		_, _, _, stackTrace := GetDebugInfo(1)

		if !strings.Contains(stackTrace, "TestGetDebugInfoNestedInline.func1") {
			t.Errorf("Expected stack trace to contain 'TestGetDebugInfoNested.func1', got %v", stackTrace)
		}
	}

	nestedFunction()
}
