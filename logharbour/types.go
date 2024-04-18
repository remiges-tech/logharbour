// Package logharbour is a comprehensive logging system.
// It supports different log levels, log types, and can encode log entries in JSON.
// It also provides a fallback mechanism in case the primary log writer fails.
package logharbour

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

// logPriority defines the severity level of a log message.
type LogPriority int

const (
	Debug2 LogPriority = iota + 1 // Debug2 represents extremely verbose debugging information.
	Debug1                        // Debug1 represents detailed debugging information.
	Debug0                        // Debug0 represents high-level debugging information.
	Info                          // Info represents informational messages.
	Warn                          // Warn represents warning messages.
	Err                           // Err represents error messages where operations failed to complete.
	Crit                          // Crit represents critical failure messages.
	Sec                           // Sec represents security alert messages.
)

const (
	LogPriorityDebug2  = "Debug2"
	LogPriorityDebug1  = "Debug1"
	LogPriorityDebug0  = "Debug0"
	LogPriorityInfo    = "Info"
	LogPriorityWarn    = "Warn"
	LogPriorityErr     = "Err"
	LogPriorityCrit    = "Crit"
	LogPrioritySec     = "Sec"
	LogPriorityUnknown = "Unknown"
)

// String returns the string representation of the logPriority.
func (lp LogPriority) String() string {
	switch lp {
	case Debug2:
		return LogPriorityDebug2
	case Debug1:
		return LogPriorityDebug1
	case Debug0:
		return LogPriorityDebug0
	case Info:
		return LogPriorityInfo
	case Warn:
		return LogPriorityWarn
	case Err:
		return LogPriorityErr
	case Crit:
		return LogPriorityCrit
	case Sec:
		return LogPrioritySec
	default:
		return LogPriorityUnknown
	}
}

// MarshalJSON is required by the encoding/json package.
// It converts the logPriority to its string representation and returns it as a JSON-encoded value.
func (lp LogPriority) MarshalJSON() ([]byte, error) {
	return json.Marshal(lp.String())
}

func (lp *LogPriority) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	value, ok := map[string]LogPriority{
		"Debug2": Debug2,
		"Debug1": Debug1,
		"Debug0": Debug0,
		"Info":   Info,
		"Warn":   Warn,
		"Err":    Err,
		"Crit":   Crit,
		// Add other LogPriority values here
	}[s]

	if !ok {
		return fmt.Errorf("invalid LogPriority %q", s)
	}

	*lp = value
	return nil
}

// LogType defines the category of a log message.
type LogType int

const (
	// Change represents a log entry for data changes.
	Change LogType = iota + 1
	// Activity represents a log entry for activities such as web service calls.
	Activity
	// Debug represents a log entry for debug information.
	Debug
	// Unknown represents an unknown log type.
	Unknown
)

const (
	LogTypeChange   = "C"
	LogTypeActivity = "A"
	LogTypeDebug    = "D"
	LogTypeUnknown  = "U"
)

// String returns the string representation of the LogType.
func (lt LogType) String() string {
	switch lt {
	case Change:
		return LogTypeChange
	case Activity:
		return LogTypeActivity
	case Debug:
		return LogTypeDebug
	default:
		return LogTypeUnknown
	}
}

// MarshalJSON is required by the encoding/json package.
// It converts the LogType to its string representation and returns it as a JSON-encoded value.
func (lt LogType) MarshalJSON() ([]byte, error) {
	return json.Marshal(lt.String())
}

func (lt *LogType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	value, ok := map[string]LogType{
		"U": Unknown,
		"A": Activity,
		"C": Change,
		"D": Debug,
		// Add other LogType values here
	}[s]

	if !ok {
		return fmt.Errorf("invalid LogType %q", s)
	}

	*lt = value
	return nil
}

type Status int

const (
	Success Status = iota
	Failure
)

// LogEntry encapsulates all the relevant information for a log message.
type LogEntry struct {
	App        string      `json:"app"`             // Name of the application.
	System     string      `json:"system"`          // System where the application is running.
	Module     string      `json:"module"`          // The module or subsystem within the application
	Type       LogType     `json:"type"`            // Type of the log entry.
	Pri        LogPriority `json:"pri"`             // Severity level of the log entry.
	When       time.Time   `json:"when"`            // Time at which the log entry was created.
	Who        string      `json:"who"`             // User or service performing the operation.
	Op         string      `json:"op"`              // Operation being performed
	Class      string      `json:"class"`           // Unique ID, name of the object instance on which the operation was being attempted
	InstanceId string      `json:"instance"`        // Unique ID, name, or other "primary key" information of the object instance on which the operation was being attempted
	Status     Status      `json:"status"`          // 0 or 1, indicating success (1) or failure (0), or some other binary representation
	Error      string      `json:"error,omitempty"` // Error message or error chain related to the log entry, if any.
	RemoteIP   string      `json:"remote_ip"`       // IP address of the caller from where the operation is being performed.
	Msg        string      `json:"msg"`             // A descriptive message for the log entry.
	Data       any         `json:"data"`            // The payload of the log entry, can be any type.
}

type ChangeDetail struct {
	Field  string `json:"field"`
	OldVal any    `json:"old_value"`
	NewVal any    `json:"new_value"`
}

// ChangeInfo holds information about data changes such as creations, updates, or deletions.
//
// Example usage of ChangeInfo and ChangeDetail to log changes to an entity.
//
//	func LogEntityChange() {
//		 // Create a new ChangeInfo for the "User" entity undergoing an "Update" operation and add changes.
//		 changeInfo := NewChangeInfo("User", "Update").
//			 AddChange("email", "oldEmail@example.com", "newEmail@example.com").
//			 AddChange("username", "oldUsername", "newUsername")
//		 logger.LogDataChange("User details updated", *changeInfo)
//	}
type ChangeInfo struct {
	Entity  string         `json:"entity"`
	Op      string         `json:"op"`
	Changes []ChangeDetail `json:"changes"`
}

// ActivityInfo holds information about system activities like web service calls or function executions.
type ActivityInfo any

// DebugInfo holds debugging information that can help in software diagnostics.
type DebugInfo struct {
	Pid          int    `json:"pid"`
	Runtime      string `json:"runtime"`
	FileName     string `json:"file"`
	LineNumber   int    `json:"line"`
	FunctionName string `json:"func"`
	StackTrace   string `json:"stackTrace"`
	Data         any    `json:"data"`
}

// FallbackWriter provides an io.Writer that automatically falls back to a secondary writer if the primary writer fails.
// It is also used if logentry is not valid so that we can still log erroneous entries without writing them to the primary writer.
type FallbackWriter struct {
	primary  io.Writer // The main writer to which log entries will be written.
	fallback io.Writer // The fallback writer used if the primary writer fails.
	mu       sync.Mutex
}

// NewFallbackWriter creates a new FallbackWriter with a specified primary and fallback writer.
func NewFallbackWriter(primary, fallback io.Writer) *FallbackWriter {
	return &FallbackWriter{
		primary:  primary,
		fallback: fallback,
	}
}

// Write attempts to write the byte slice to the primary writer, falling back to the secondary writer on error.
// It returns the number of bytes written and any error encountered that caused the write to stop early.
func (fw *FallbackWriter) Write(p []byte) (n int, err error) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	n, err = fw.primary.Write(p)
	if err != nil {
		// Primary writer failed; attempt to write to the fallback writer.
		n, err = fw.fallback.Write(p)
	}
	return n, err // Return the result of the write operation.
}
