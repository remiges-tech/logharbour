# LogHarbour

[![GoDoc][doc-img]][doc] 

> [!WARNING]  
> LogHarbour is currently under active development: bugs and breaking changes can happen. 
> We highly appreciate any feedback and contributions.

LogHarbour is a powerful and flexible logging library for Go applications. It provides a simple and intuitive API for logging various types of information, including activities, data changes, and debug information.
Features

- Flexible Logging Levels: LogHarbour supports multiple logging levels, allowing you to control the verbosity of your logs.
- Contextual Logging: You can attach additional context to your logs, such as the module name, user information, and more.
- Fallback Mechanism: LogHarbour provides a fallback mechanism to ensure that your logs are always captured, even if the primary logging destination fails.

Usage

Here is a basic example of how to use LogHarbour:


```Go
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

	// Initialize the logger.
	logger := NewLogger("MyApp", fallbackWriter)

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
	logger.ChangePriority(Debug2)

	// Log a debug entry.
	logger.LogDebug("Debugging user session", DebugInfo{
		Variables: map[string]any{"sessionID": "12345"},
	})

	// Output:
	// {
	//     "app_name": "MyApp",
	//     "module": "Module1",
	//     "priority": "Info",
	//     "who": "John Doe",
	//     "status": "Success",
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
	//     "status": "Success",
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
```

[doc-img]: https://pkg.go.dev/badge/go.uber.org/nilaway.svg
[doc]: https://pkg.go.dev/github.com/remiges-tech/logharbour/logharbour
