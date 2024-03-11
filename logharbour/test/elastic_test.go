package logharbour_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

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
	ExpectError        bool
}

var err error

func TestGetLogs(t *testing.T) {
	testCases := getLogsTestCase()
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {

			if tc.TestJsonFile != "" {
				tc.ExpectedLogEntries, err = elasticsearchctl.ReadLogFromFile(tc.TestJsonFile)
				if err != nil {
					fmt.Printf("error converting data from log file:%v\n", err)
				}

			}
			logharbour.Index = "logharbour_unit_test"
			tc.ActualLogEntries, tc.ActualRecords, err = logharbour.GetLogs("", typedClient, tc.LogsParam)

			if tc.ExpectError {
				if err == nil {
					t.Errorf("Expected error for input %d, but got nil", tc.ActualRecords)
				}
			} else {
				require.NoError(t, err)
			}

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
	class := "wfinstance"
	instance := "2"
	remote_ip := "192.168.1.100"
	pri := logharbour.Info
	nDay := 100
	fromTs := time.Date(2024, 02, 01, 00, 00, 00, 00, time.UTC)
	toTs := time.Date(2024, 03, 01, 00, 00, 00, 00, time.UTC)
	searchAfterTS := "2024-02-25T07:28:00.110813597Z"
	logsTestCase := []TestCasesStruct{{
		Name: "1st_tc_GetLogs_with_all_filter_param",
		LogsParam: logharbour.GetLogsParam{
			App:      &app,
			Type:     &typeConst,
			Who:      &who,
			Class:    &class,
			Instance: &instance,
			NDays:    &nDay,
			RemoteIP: &remote_ip,
			Priority: &pri,
		},
		ExpectedRecords: 1,
		TestJsonFile:    "./testData/getLogs_testdata/1st_tc_GetLogs_with_all_filter_param.json",
		ExpectError:     false,
	},
		{
			Name: "2nd_tc_GetLogs_within_ts",
			LogsParam: logharbour.GetLogsParam{
				App:    &app,
				FromTS: &fromTs,
				ToTS:   &toTs,
			},
			ExpectedRecords: 42,
			TestJsonFile:    "./testData/getLogs_testdata/2nd_tc_GetLogs_within_ts.json",
			ExpectError:     false,
		},
		{
			Name: "3rd_test_case_GetLogs_with_SearchAfterTS",
			LogsParam: logharbour.GetLogsParam{
				App:           &app,
				FromTS:        &fromTs,
				ToTS:          &toTs,
				SearchAfterTS: &searchAfterTS,
			},
			ExpectedRecords: 42,
			TestJsonFile:    "./testData/getLogs_testdata/3rd_test_case_GetLogs_with_SearchAfterTS.json",
			ExpectError:     false,
		},
		{
			Name:               "4th_tc_without_filterParam",
			LogsParam:          logharbour.GetLogsParam{},
			ExpectedLogEntries: nil,
			ExpectedRecords:    0,
			ExpectError:        true,
		},
	}
	return logsTestCase
}
