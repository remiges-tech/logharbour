package logharbour

import (
	"os"
	"runtime"
	"strings"
)

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
		length := runtime.Stack(buf, false)
		stackTrace = string(buf[:length])
	}
	return
}
