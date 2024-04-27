package logharbour

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
)

// convertToString attempts to convert any given value to its JSON string representation.
// This function is used to ensure that all values stored in ChangeDetail structs are in string format,
// which is required for its storage in logharbour storage.
// If the conversion fails, the error is written to os.Stderr to notify of the failure,
// and a placeholder error message is returned. This approach was chosen to avoid complicating the API
// with error handling for what is expected to be a rare event. It allows the calling code to proceed,
// potentially logging the conversion error alongside the intended log message.
func convertToString(value any) string {
	// If the value is simple string no need to marshal it
	// Marshalling string would result in double encoding
	// where simple string like "hello" becomes "\"hello\""
	// Also avoiding marshalling would save unnecessary computation
	if str, ok := value.(string); ok {
		return str
	}
	bytes, err := json.Marshal(value)
	if err != nil {
		// Write the error to os.Stderr
		fmt.Fprintf(os.Stderr, "Error converting value to string: %v\n", err)
		return fmt.Sprintf("strconv error: %v", err)
	}
	return string(bytes)
}

// GetSystemName returns the host name of the system.
func getSystemName() string {
	host, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return host
}

// GetDebugInfo returns debug information including file name, line number, function name and stack trace.
// The 'skip' parameter determines how many stack frames to ascend, with 0 identifying the caller of GetDebugInfo.
func GetDebugInfo(skip int) (fileName string, lineNumber int, functionName string, stackTrace string) {
	pc, file, line, ok := runtime.Caller(skip)
	if ok {
		fileName = file
		lineNumber = line

		// Get the function name
		funcName := runtime.FuncForPC(pc).Name()
		// Trim the package name
		funcName = funcName[strings.LastIndex(funcName, ".")+1:]
		functionName = funcName

		// Get the stack trace
		buf := make([]byte, 1024)
		runtime.Stack(buf, false)
		stackTrace = formatStackTrace(string(buf))
	}
	return
}

// formatStackTrace simplifies the stack trace by removing unnecessary details and formatting the remaining information.
func formatStackTrace(stackTraceRaw string) string {
	stackTraceLines := strings.Split(stackTraceRaw, "\n")
	for i, line := range stackTraceLines {
		if strings.Contains(line, "runtime.") {
			continue
		}
		parts := strings.Split(line, " ")
		if len(parts) > 1 {
			funcName := parts[1]
			funcName = funcName[strings.LastIndex(funcName, "/")+1:]
			funcNameParts := strings.Split(funcName, ".")
			if len(funcNameParts) > 1 {
				funcName = funcNameParts[1]
			}
			stackTraceLines[i] = fmt.Sprintf("%s %s", funcName, parts[0])
		}
	}
	return strings.Join(stackTraceLines, "\n")
}
