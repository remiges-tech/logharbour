package logharbour

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/segmentio/ksuid"
)

const DefaultPriority = Info

// LoggerContext provides a shared context (state) for instances of Logger.
// It contains a minLogPriority field that determines the minimum log priority level
// that should be logged by any Logger using this context.
// The mu Mutex ensures that all operations on the minLogPriority field are mutually exclusive,
// regardless of which goroutine they are performed in.
type LoggerContext struct {
	minLogPriority LogPriority
	debugMode      int32 // int32 to represent the boolean flag atomically
	mu             sync.Mutex
}

// NewLoggerContext creates a new LoggerContext with the specified minimum log priority.
// The returned LoggerContext can be used to create Logger instances with a shared context for all the
// Logger instances.
func NewLoggerContext(minLogPriority LogPriority) *LoggerContext {
	return &LoggerContext{
		minLogPriority: minLogPriority,
	}
}

// Logger provides a structured interface for logging.
// It's designed for each goroutine to have its own instance.
// Logger is safe for concurrent use. However, it's not recommended
// to share a Logger instance across multiple goroutines.
//
// If the writer is a FallbackWriter and validation of a log entry fails,
// the Logger will automatically write the invalid entry to the FallbackWriter's fallback writer.
// If writing to fallback writer also fails then it writes to STDERR.
//
// The 'With' prefixed methods in the Logger are used to create a new Logger instance
// with a specific field set to a new value. These methods  create a copy of the current Logger,
// then set the desired field to the new value, and finally return the new Logger.
// This approach provides a flexible way to create a new Logger with specific settings,
// without having to provide all settings at once or change the settings of an existing Logger.
type Logger struct {
	context    *LoggerContext      // Context for the logger. It is shared by all clones of the logger.
	app        string              // Name of the application.
	system     string              // System where the application is running.
	module     string              // Module or subsystem within the application.
	pri        LogPriority         // Priority level of the log messages.
	who        string              // User or service performing the operation.
	op         string              // Operation being performed.
	class      string              // Class of the object instance involved.
	instanceId string              // Unique ID of the object instance.
	status     Status              // Status of the operation.
	err        string              // Error associated with the operation.
	remoteIP   string              // IP address of the remote endpoint.
	writer     io.Writer           // Writer interface for log entries.
	validator  *validator.Validate // Validator for log entries.
	mu         sync.Mutex          // Mutex for thread-safe operations.
}

// clone creates and returns a new Logger with the same values as the original.
func (l *Logger) clone() *Logger {
	return &Logger{
		context:    l.context,
		app:        l.app,
		system:     l.system,
		module:     l.module,
		pri:        l.pri,
		who:        l.who,
		op:         l.op,
		class:      l.class,
		instanceId: l.instanceId,
		status:     l.status,
		err:        l.err,
		remoteIP:   l.remoteIP,
		writer:     l.writer,
		validator:  l.validator,
	}
}

// NewLogger creates a new Logger with the specified application name and writer.
// We recommend using NewLoggerWithFallback instead of this method.
func NewLogger(context *LoggerContext, appName string, writer io.Writer) *Logger {
	return &Logger{
		context:   context,
		app:       appName,
		system:    getSystemName(),
		writer:    writer,
		validator: validator.New(),
		pri:       DefaultPriority,
	}
}

// NewLoggerWithFallback creates a new Logger with a fallback writer.
// The fallback writer is used if the primary writer fails or if validation of a log entry fails.
func NewLoggerWithFallback(context *LoggerContext, appName string, fallbackWriter *FallbackWriter) *Logger {
	return &Logger{
		context:   context,
		app:       appName,
		system:    getSystemName(),
		writer:    fallbackWriter,
		validator: validator.New(),
		pri:       DefaultPriority,
	}
}

// WithWho returns a new Logger with the 'who' field set to the specified value.
func (l *Logger) WithWho(who string) *Logger {
	newLogger := l.clone() // Create a copy of the logger
	newLogger.who = who    // Change the 'who' field
	return newLogger       // Return the new logger
}

// WithModule returns a new Logger with the 'module' field set to the specified value.
func (l *Logger) WithModule(module string) *Logger {
	newLogger := l.clone()
	newLogger.module = module
	return newLogger
}

// WithOp returns a new Logger with the 'op' field set to the specified value.
func (l *Logger) WithOp(op string) *Logger {
	newLogger := l.clone()
	newLogger.op = op
	return newLogger
}

// WithClass returns a new Logger with the 'whatClass' field set to the specified value.
func (l *Logger) WithClass(whatClass string) *Logger {
	newLogger := l.clone()
	newLogger.class = whatClass
	return newLogger
}

// WithInstanceId returns a new Logger with the 'whatInstanceId' field set to the specified value.
func (l *Logger) WithInstanceId(whatInstanceId string) *Logger {
	newLogger := l.clone()
	newLogger.instanceId = whatInstanceId
	return newLogger
}

// WithStatus returns a new Logger with the 'status' field set to the specified value.
func (l *Logger) WithStatus(status Status) *Logger {
	newLogger := l.clone()
	newLogger.status = status
	return newLogger
}

// Err sets the "error" field for the logger.
func (l *Logger) Error(err error) *Logger {
	newLogger := l.clone()
	newLogger.err = err.Error()
	return newLogger
}

// WithPriority returns a new Logger with the 'priority' field set to the specified value.
//
// There are shortcut functions like Info(), Warn(), etc. provided as a convenient way to
// set the priority level for a single call.
// Each of these function creates a new Logger instance with the specified priority and returns it.
// The original Logger instance remains unchanged.
// For example, instead of writing
//
//	logger.WithPriority(logharbour.Info).LogChange(...),
//
// you can simply write
//
//	logger.Info().LogChange(...)
func (l *Logger) WithPriority(priority LogPriority) *Logger {
	newLogger := l.clone()
	newLogger.pri = priority
	return newLogger
}

// WithRemoteIP returns a new Logger with the 'remoteIP' field set to the specified value.
func (l *Logger) WithRemoteIP(remoteIP string) *Logger {
	newLogger := l.clone()
	newLogger.remoteIP = remoteIP
	return newLogger
}

// log writes a log entry. It locks the Logger's mutex to prevent concurrent write operations.
// If there's a problem with writing the log entry or if the log entry is invalid,
// it attempts to write the error and the log entry to the fallback writer (if available).
// If writing to the fallback writer fails or if the fallback writer is not available,
// it writes the error and the log entry to stderr.
func (l *Logger) log(entry LogEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry.App = l.app
	if !l.shouldLog(entry.Pri) {
		return
	}

	entry.Id = ksuid.New().String()

	if err := l.validator.Struct(entry); err != nil {
		// Check if the writer is a FallbackWriter
		if fw, ok := l.writer.(*FallbackWriter); ok {
			// Write to the fallback writer if validation fails
			if err := formatAndWriteEntry(fw.fallback, entry); err != nil {
				// If writing to the fallback writer fails, write to stderr
				fmt.Fprintf(os.Stderr, "Error: %v, LogEntry: %+v\n", err, entry)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v, LogEntry: %+v\n", err, entry)
		}
		return
	}
	if err := formatAndWriteEntry(l.writer, entry); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v, LogEntry: %+v\n", err, entry)
	}
}

// shouldLog determines whether a log entry should be written based on its priority.
func (l *Logger) shouldLog(p LogPriority) bool {
	l.context.mu.Lock()
	defer l.context.mu.Unlock()
	return p >= l.context.minLogPriority
}

// formatAndWriteEntry formats a log entry as JSON and writes it to the Logger's writer.
func formatAndWriteEntry(writer io.Writer, entry LogEntry) error {
	formattedEntry, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	formattedEntry = append(formattedEntry, '\n')
	_, writeErr := writer.Write(formattedEntry)
	return writeErr
}

// newLogEntry creates a new log entry with the specified message and data.
func (l *Logger) newLogEntry(message string, data *LogData) LogEntry {
	return LogEntry{
		App:        l.app,
		System:     l.system,
		Module:     l.module,
		Pri:        l.pri,
		Who:        l.who,
		Op:         l.op,
		When:       time.Now().UTC(),
		Class:      l.class,
		InstanceId: l.instanceId,
		Status:     l.status,
		Error:      l.err,
		RemoteIP:   l.remoteIP,
		Msg:        message,
		Data:       data,
	}
}

// LogDataChange logs a data change event.
func (l *Logger) LogDataChange(message string, data ChangeInfo) {
	for i := range data.Changes {
		data.Changes[i].OldVal = convertToString(data.Changes[i].OldVal)
		data.Changes[i].NewVal = convertToString(data.Changes[i].NewVal)
	}

	logData := LogData{
		ChangeData: &data,
	}
	entry := l.newLogEntry(message, &logData)
	entry.Type = Change
	l.log(entry)
}

// LogActivity logs an activity event.
func (l *Logger) LogActivity(message string, data ActivityInfo) {
	var logData LogData
	var entry LogEntry
	if data != nil {
		activityData := convertToString(data)
		logData = LogData{
			ActivityData: activityData,
		}
		entry = l.newLogEntry(message, &logData)
	} else {
		entry = l.newLogEntry(message, nil)
	}
	entry.Type = Activity
	l.log(entry)
}

// LogDebug logs a debug event.
func (l *Logger) LogDebug(message string, data any) {
	if !l.context.IsDebugModeSet() {
		return // Skip logging if debugMode is not enabled
	}
	debugInfo := DebugInfo{
		Pid:          os.Getpid(),
		Runtime:      runtime.Version(),
		FileName:     "",
		LineNumber:   0,
		FunctionName: "",
		StackTrace:   "",
		Data:         convertToString(data), // Convert the entire data to a JSON string
	}

	// Populate file name, line number, function name, and stack trace
	// skip = 0 means GetDebugInfo itself
	// skip = 1 means the caller of GetDebugInfo i.e. LogDebug
	// skip = 2 means the caller of LogDebug i.e. the function that called LogDebug which we will add to DebugInfo
	debugInfo.FileName, debugInfo.LineNumber, debugInfo.FunctionName, debugInfo.StackTrace = GetDebugInfo(2)

	logData := LogData{
		DebugData: &debugInfo,
	}
	entry := l.newLogEntry(message, &logData)
	entry.Type = Debug
	l.log(entry)
}

// Log logs a generic message as an activity event.
func (l *Logger) Log(message string) {
	l.LogActivity(message, nil)
}

// SetDebugMode sets the debug mode for all loggers sharing this context.
// Passing true enables debug logging, while false disables it.
func (lc *LoggerContext) SetDebugMode(enable bool) {
	var val int32
	if enable {
		val = 1
	}
	atomic.StoreInt32(&lc.debugMode, val) // Atomically update debugMode
}

// IsDebugModeSet checks if debug mode is enabled in a thread-safe manner.
func (lc *LoggerContext) IsDebugModeSet() bool {
	return atomic.LoadInt32(&lc.debugMode) == 1 // Atomically read debugMode
}

// ChangePriority changes the priority level of the Logger.
func (lc *LoggerContext) ChangeMinLogPriority(minLogPriority LogPriority) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.minLogPriority = minLogPriority
}

// Debug2 returns a new Logger with the 'priority' field set to Debug2.
func (l *Logger) Debug2() *Logger {
	return l.WithPriority(Debug2)
}

// Debug1 returns a new Logger with the 'priority' field set to Debug1.
func (l *Logger) Debug1() *Logger {
	return l.WithPriority(Debug1)
}

// Debug0 returns a new Logger with the 'priority' field set to Debug0.
func (l *Logger) Debug0() *Logger {
	return l.WithPriority(Debug0)
}

// Info returns a new Logger with the 'priority' field set to Info.
func (l *Logger) Info() *Logger {
	return l.WithPriority(Info)
}

// Warn returns a new Logger with the 'priority' field set to Warn.
func (l *Logger) Warn() *Logger {
	return l.WithPriority(Warn)
}

// Err returns a new Logger with the 'priority' field set to Err.
func (l *Logger) Err() *Logger {
	return l.WithPriority(Err)
}

// Crit returns a new Logger with the 'priority' field set to Crit.
func (l *Logger) Crit() *Logger {
	return l.WithPriority(Crit)
}

// Sec returns a new Logger with the 'priority' field set to Sec.
func (l *Logger) Sec() *Logger {
	return l.WithPriority(Sec)
}

// NewChangeDetail creates a new ChangeDetail instance from given field, oldValue, and newValue.
// The oldValue and newValue are any type, and internally converted to their string representations
// using convertToString() in utils.go. This design allows for flexibility in logging changes without enforcing
// a strict type constraint on the values being logged. It ensures that regardless of the original value type,
// the change details are stored as strings, which is required for storing it in logharbour storage.
func NewChangeDetail(field string, oldValue, newValue any) ChangeDetail {
	return ChangeDetail{
		Field:  field,
		OldVal: convertToString(oldValue),
		NewVal: convertToString(newValue),
	}
}

// AddChange adds a new change to the ChangeInfo struct. It accepts a field name and old/new values of any type.
// Internally, it uses NewChangeDetail to create a ChangeDetail struct, which converts the old/new values to strings.
// This method simplifies the process of adding changes to a log entry, allowing developers to pass values of any type
// without worrying about their string conversion. The use of convertToString() ensures that all values are consistently
// logged as strings, which is required for storing them in logharbour storage.
func (ci ChangeInfo) AddChange(field string, oldValue, newValue any) ChangeInfo {
	change := NewChangeDetail(field, oldValue, newValue)
	ci.Changes = append(ci.Changes, change)
	return ci
}

// NewChangeInfo creates a new ChangeInfo instance.
func NewChangeInfo(entity, operation string) ChangeInfo {
	return ChangeInfo{
		Entity:  entity,
		Op:      operation,
		Changes: []ChangeDetail{},
	}
}

// Logf is a variant of Log that takes a formatted message
func (l *Logger) Logf(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	l.Log(message)
}

// LogActivityf is a variant of LogActivity that takes a formatted message
func (l *Logger) LogActivityf(data ActivityInfo, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	l.LogActivity(message, data)
}

// LogDataChangef is a variant of LogDataChange that takes a formatted message
func (l *Logger) LogDataChangef(data ChangeInfo, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	l.LogDataChange(message, data)
}

// LogDebugf is a variant of LogDebug that takes a formatted message
func (l *Logger) LogDebugf(data any, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	l.LogDebug(message, data)
}
