package logHarbour

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"
)

func setupEmptyFile() {
	filename := getRigelLogFileName()
	fmt.Println("removing file ", filename)
	err := os.Truncate(filename, 0)
	if err != nil {
		fmt.Println("file not found:", err)
	}
}

type errorLogMsg struct {
	Time        string `json:"time"`
	Level       string `json:"level"`
	Msg         string `json:"msg"`
	Handle      string `json:"handle"`
	App         string `json:"app"`
	Module      string `json:"module"`
	System      string `json:"system"`
	CurrentTime string `json:"currentTime"`
	When        string `json:"when"`
	LogMsgStr   logMsg `json:"LOG_MSG"`
	Found       string `json:"found"`
	IPAddrMsg   string `json:"needed in RemoteIp"`
	Required    string `json:"required"`
}

var lm logMsg = logMsg{
	Ll:             "INFO",
	SpanId:         "spanid",
	CorrelationId:  "correlationid",
	Who:            "bhavya",
	RemoteIp:       "127.0.0.1",
	Op:             "newLog",
	WhatClass:      "valueBeingUpdated",
	WhatInstanceId: "id2",
	Status:         1,
	Msg:            "This is an activity logger error message",
	When:           time.Now(),
}

func TestLogWrite(t *testing.T) {

	setupEmptyFile()
	loggers := LogInit("app1", "module1", "system1")

	//Check non zero fields are not allowed for any fields of LogWrite method
	checkNonZeroField(t, loggers, "Msg")
	checkNonZeroField(t, loggers, "SpanId")
	checkNonZeroField(t, loggers, "CorrelationId")
	checkNonZeroField(t, loggers, "Who")
	checkNonZeroField(t, loggers, "RemoteIp")
	checkNonZeroField(t, loggers, "Op")
	checkNonZeroField(t, loggers, "WhatClass")
	checkNonZeroField(t, loggers, "WhatInstanceId")
	checkNonZeroField(t, loggers, "Status")

	//Only valid IP address are allowed
	checkInvalidIpAddress(t, loggers, "abcd")
	checkInvalidIpAddress(t, loggers, "900.900.900.90")

	//The field 'when' cannot be in future
	checkWhenCannotBeInFuture(t, loggers)

	//App indentifier fields should be present in all types of Loggers
	checkFieldPresentInLog(t, loggers.ActivityLogger, LevelInfo, "app")
	checkFieldPresentInLog(t, loggers.DataChangeLogger, LevelError, "module")
	checkFieldPresentInLog(t, loggers.DebugLogger, LevelDebug0, "system")
	checkFieldPresentInLog(t, loggers.ActivityLogger, LevelInfo, "handle")

	//Check activityLogger does not have DataChangeObj
	checkFieldNotPresentInLog(t, loggers.ActivityLogger, LevelInfo, "oldVal")
	//Check DebugLogger does not have DataChangeObj
	checkFieldNotPresentInLog(t, loggers.DebugLogger, LevelDebug0, "oldVal")
	//Check DataChangeLogger does have DataChangeObj
	checkFieldPresentInLog(t, loggers.DataChangeLogger, LevelInfo, "oldVal")

	//Loghandles should be correct for respective type of logs
	checkLogHandleField(t, loggers.ActivityLogger, "A", "Handle of ActivityLogger should be of type "+ACTIVITY_LOGGER)
	checkLogHandleField(t, loggers.DebugLogger, "D", "Handle of DebugLogger should be of type "+DEBUG_LOGGER)
	checkLogHandleField(t, loggers.DataChangeLogger, "C", "Handle of DataChangeLogger should be of type "+DATACHANGE_LOGGER)

	//Debug fields should be present in DebugLoggers
	checkFieldPresentInLog(t, loggers.DebugLogger, LevelDebug2, "pid")
	checkFieldPresentInLog(t, loggers.DebugLogger, LevelDebug1, "runtime")
	checkFieldPresentInLog(t, loggers.DebugLogger, LevelDebug0, "callTrace")
	checkFieldPresentInLog(t, loggers.DebugLogger, LevelDebug2, "source")

	//Debug fields should not be present in Activity and DatachangeLogger
	checkFieldNotPresentInLog(t, loggers.DataChangeLogger, LevelInfo, "runtime")
	checkFieldNotPresentInLog(t, loggers.DataChangeLogger, LevelDebug0, "pid")
	checkFieldNotPresentInLog(t, loggers.ActivityLogger, LevelInfo, "callTrace")
	checkFieldNotPresentInLog(t, loggers.ActivityLogger, LevelError, "source")
}

func checkInvalidIpAddress(t *testing.T, loggers LogHandles, ipAddr string) {
	t.Run("Valid IP Address is required", func(t *testing.T) {
		setupEmptyFile()
		LogWrite(loggers.ActivityLogger, LevelError, "spanid2", "correlationid2", time.Now(), "bhavya", ipAddr,
			"newLog", "valueBeingUpdated", "id2", 1, "This is an activity logger error message", "somekey", "somevalue",
			DataChg("amt", "100", "200"), DataChg("qty", "1", "2"))

		readFile, err := os.Open(getRigelLogFileName())

		if err != nil {
			fmt.Println(err)
		}
		fileScanner := bufio.NewScanner(readFile)

		fileScanner.Split(bufio.ScanLines)

		var errorLog errorLogMsg
		for fileScanner.Scan() {
			//fmt.Println("line read::::::::", fileScanner.Text())

			json.Unmarshal(fileScanner.Bytes(), &errorLog)

			if (errorLog.LogMsgStr != logMsg{}) {
				errorLog = errorLogMsg{}
				continue
			} else {
				if errorLog.Level != "ERROR" || errorLog.IPAddrMsg != "ip_addr" {
					t.Error("Log message is test failed")
				} else {
					t.Log("Log message is test passed")
				}
			}
		}
		readFile.Close()
	})
}

func checkNonZeroField(t *testing.T, loggers LogHandles, field string) {
	t.Run("Non Zero values not allowed:"+field, func(t *testing.T) {
		setupEmptyFile()
		lmTest := lm
		switch field {
		case "SpanId":
			lmTest.SpanId = ""
		case "CorrelationId":
			lmTest.CorrelationId = ""
		case "Who":
			lmTest.Who = ""
		case "When":
			lmTest.When = time.Time{}
		case "RemoteIp":
			lmTest.RemoteIp = ""
		case "Op":
			lmTest.Op = ""
		case "WhatClass":
			lmTest.WhatClass = ""
		case "WhatInstanceId":
			lmTest.WhatInstanceId = ""
		case "Status":
			lmTest.Status = 0
		case "Msg":
			lmTest.Msg = ""
		}

		LogWrite(loggers.ActivityLogger, LevelError, lmTest.SpanId, lmTest.CorrelationId, lmTest.When, lmTest.Who,
			lmTest.RemoteIp, lmTest.Op, lmTest.WhatClass, lmTest.WhatInstanceId, lmTest.Status, lmTest.Msg,
			"somekey", "somevalue", DataChg("amt", "100", "200"), DataChg("qty", "1", "2"))

		readFile, err := os.Open(getRigelLogFileName())

		if err != nil {
			fmt.Println(err)
		}
		fileScanner := bufio.NewScanner(readFile)

		fileScanner.Split(bufio.ScanLines)

		var errorLog errorLogMsg
		for fileScanner.Scan() {
			//fmt.Printf("[field:%v]line read::::::::%v\n", field, fileScanner.Text())

			json.Unmarshal(fileScanner.Bytes(), &errorLog)

			if (errorLog.LogMsgStr != logMsg{}) {
				errorLog = errorLogMsg{}
				continue
			} else {
				if errorLog.Level != "ERROR" || errorLog.Required != field {
					t.Error("Log message is test failed")
				} else {
					t.Log("Log message is test passed")
				}
			}
		}
		readFile.Close()
	})
}

func checkFieldPresentInLog(t *testing.T, loggers *slog.Logger, level slog.Level, field string) {
	checkFieldInLog(t, loggers, level, field, true)
}

func checkFieldNotPresentInLog(t *testing.T, loggers *slog.Logger, level slog.Level, field string) {
	checkFieldInLog(t, loggers, level, field, false)
}

func checkFieldInLog(t *testing.T, loggers *slog.Logger, level slog.Level, field string, present bool) {
	var testMsg string
	if present {
		testMsg = "Log message should have field:[" + field + "]"
	} else {
		testMsg = "Log message should not have field:[" + field + "]"
	}
	t.Run(testMsg, func(t *testing.T) {
		setupEmptyFile()
		lmTest := lm
		LogWrite(loggers, level, lmTest.SpanId, lmTest.CorrelationId, lmTest.When, lmTest.Who,
			lmTest.RemoteIp, lmTest.Op, lmTest.WhatClass, lmTest.WhatInstanceId, lmTest.Status, lmTest.Msg,
			"somekey", "somevalue", DataChg("amt", "100", "200"), DataChg("qty", "1", "2"))

		readFile, err := os.Open(getRigelLogFileName())
		if err != nil {
			fmt.Println(err)
		}
		fileScanner := bufio.NewScanner(readFile)
		fileScanner.Split(bufio.ScanLines)
		var fieldIsPresent = false
		for fileScanner.Scan() {
			//fmt.Printf("[field:%v]line read::::::::%v\n", field, fileScanner.Text())
			s := "\"" + field + "\""
			if strings.Contains(fileScanner.Text(), s) {
				fieldIsPresent = true
				//t.Log("Log message is test passed")
			} else {
				fieldIsPresent = false
				//t.Errorf("Log message is test failed. field [%v] is not present\n", field)
			}
		}
		if fieldIsPresent == present {
			t.Log("Log message is test passed")
		} else {
			t.Errorf("Log message is test failed. field [%v] is not present\n", field)
		}
		readFile.Close()
	})
}

func checkLogHandleField(t *testing.T, loggers *slog.Logger, handle string, testName string) {
	t.Run(testName, func(t *testing.T) {
		setupEmptyFile()
		lmTest := lm
		LogWrite(loggers, LevelInfo, lmTest.SpanId, lmTest.CorrelationId, lmTest.When, lmTest.Who,
			lmTest.RemoteIp, lmTest.Op, lmTest.WhatClass, lmTest.WhatInstanceId, lmTest.Status, lmTest.Msg,
			"somekey", "somevalue", DataChg("amt", "100", "200"), DataChg("qty", "1", "2"))

		readFile, err := os.Open(getRigelLogFileName())
		if err != nil {
			fmt.Println(err)
		}
		fileScanner := bufio.NewScanner(readFile)
		fileScanner.Split(bufio.ScanLines)
		var logLine errorLogMsg
		for fileScanner.Scan() {
			json.Unmarshal(fileScanner.Bytes(), &logLine)
			if logLine.Handle != handle {
				t.Error("Log message is test failed")
			} else {
				t.Log("Log message is test passed")
			}
		}
		readFile.Close()
	})
}

func checkWhenCannotBeInFuture(t *testing.T, loggers LogHandles) {
	t.Run("When Cannot be in Future", func(t *testing.T) {
		setupEmptyFile()

		// Test case 1: when is in future, DEFAULT Logger should write error
		LogWrite(loggers.ActivityLogger, LevelInfo, "spanid1", "correlationid1", time.Now().AddDate(1, 0, 0), "bhavya", "127.0.0.1",
			"newLog", "valueBeingUpdated", "id1", 1, "This is an activity logger info message", "somekey", "somevalue", "key2", "value2")
		readFile, err := os.Open(getRigelLogFileName())
		if err != nil {
			fmt.Println(err)
		}
		fileScanner := bufio.NewScanner(readFile)
		fileScanner.Split(bufio.ScanLines)
		var errorLog errorLogMsg
		for fileScanner.Scan() {
			json.Unmarshal(fileScanner.Bytes(), &errorLog)
			if (errorLog.LogMsgStr != logMsg{}) {
				errorLog = errorLogMsg{}
				continue
			} else {
				if errorLog.Level != "ERROR" || errorLog.Msg != "LOG_MSG_ERR: 'when' cannot be after system current time." {
					t.Error("Log message is test failed")
				} else {
					t.Log("Log message is test passed")
				}
			}
		}
		readFile.Close()
	})
}
