package logharbour_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/remiges-tech/logharbour/logharbour"
	elasticsearchctl "github.com/remiges-tech/logharbour/server/elasticSearchCtl/elasticSearch"
	"github.com/stretchr/testify/require"
)

type TestCasesStruct struct {
	Name               string
	LogsParam          logharbour.GetLogsParam
	ExpectedLogEntries []logharbour.LogEntry
	ActualLogEntries   []logharbour.LogEntry
	ExpectedRecords    int
	ActualRecords      int
	TestJsonFile       string
}

var err error

func TestGetLogs(t *testing.T) {
	testCases := getLogsTestCase()
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			logharbour.Index = "logharbour_unit_test"
			tc.ActualLogEntries, tc.ActualRecords, err = logharbour.GetLogs("", typedClient, tc.LogsParam)
			require.NoError(t, err)

			// Compare the LogEntries
			if !reflect.DeepEqual(tc.ExpectedLogEntries, tc.ActualLogEntries) {
				t.Errorf("LogEntries are not equal. Expected: %v, Actual: %v", tc.ExpectedLogEntries, tc.ActualLogEntries)
			}

			// compare number of records
			if tc.ExpectedRecords != tc.ActualRecords {
				t.Errorf("Expected: %d, Actual: %d", tc.ExpectedRecords, tc.ActualRecords)
			}
		})
	}

}

func getLogsTestCase() []TestCasesStruct {
	app := "crux"
	typeConst := logharbour.Activity
	who := "Tushar"
	// class := "class"
	// instance := "instance"
	// op := "op"
	// remote_ip := "remote_ip"
	// pri := "pri"
	// id := "id" // document id
	logEntries, err := elasticsearchctl.ReadLogFromFile("./testData/getLogs_testdata/1stTestcase.json")
	if err != nil {
		fmt.Printf("error converting data from log file:%v\n", err)
	}
	logsTestCase := []TestCasesStruct{{
		Name: "1st test case",
		LogsParam: logharbour.GetLogsParam{
			App:  &app,
			Type: &typeConst,
			Who:  &who,
			// Class:            new(string),
			// Instance:         new(string),
			// Operation:        new(string),
			// FromTS:           &time.Time{},
			// ToTS:             &time.Time{},
			// NDays:            *,
			// RemoteIP:         new(string),
			// Priority:         new(string),
			// SearchAfterTS:    new(string),
			// SearchAfterDocID: new(string),
		},
		ExpectedLogEntries: logEntries,
		ExpectedRecords:    42,
	}}
	return logsTestCase
}
