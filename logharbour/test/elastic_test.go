package logharbour_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/remiges-tech/logharbour/logharbour"
	estestutils "github.com/remiges-tech/logharbour/logharbour/test"

	"github.com/stretchr/testify/require"
)

type GetlogsTestCaseStruct struct {
	Name               string
	LogsParam          logharbour.GetLogsParam
	ExpectedLogEntries []logharbour.LogEntry
	ActualLogEntries   []logharbour.LogEntry
	ExpectedRecords    int
	ActualRecords      int
	TestJsonFile       string
	ExpectError        bool
}

type GetSetTestCasesStruct struct {
	Name             string
	SetAttribute     string
	GetSetParam      logharbour.GetSetParam
	ExpectedResponse map[string]int64
	ActualResponse   map[string]int64
	ExpectedError    bool
}
type GetAppsTestCasesStruct struct {
	Name             string
	ExpectedResponse []string
	ActualResponse   []string
	ExpectedError    bool
}

type GetUnusualIPTestCaseStruct struct {
	Name              string
	GetUnusualIPParam logharbour.GetUnusualIPParam
	unusualPercent    float64
	ActualLogEntries  []logharbour.LogEntry
	ExpectedIps       []string
	ActualIps         []string
	ExpectError       bool
}

type GetUnusualIPListTestCaseStruct struct {
	Name              string
	GetUnusualIPParam logharbour.GetUnusualIPParam
	unusualPercent    float64
	ActualLogEntries  []logharbour.LogEntry
	ExpectedIps       []logharbour.IPLocation
	ActualIps         []logharbour.IPLocation
	ExpectError       bool
}

var err error

func TestGetLogs(t *testing.T) {
	testCases := getLogsTestCase()
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {

			if tc.TestJsonFile != "" {
				tc.ExpectedLogEntries, err = estestutils.ReadLogFromFile(tc.TestJsonFile)
				if err != nil {
					fmt.Printf("error converting data from log file:%v\n", err)
				}

			}
			logharbour.Index = "logharbour"
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

func getLogsTestCase() []GetlogsTestCaseStruct {
	app := "amazon"
	typeConst := logharbour.Activity
	who := "Jannie"
	class := "config"
	instance := "3"
	remote_ip := "142.250.67.206"
	pri := logharbour.Info
	nDay := 100
	fromTs := time.Date(2024, 02, 01, 00, 00, 00, 00, time.UTC)
	toTs := time.Date(2024, 05, 30, 00, 00, 00, 00, time.UTC)
	searchAfterTS := "2024-02-25T07:28:00.110813597Z"
	logsTestCase := []GetlogsTestCaseStruct{
		{
			Name: "1st test case ",
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
			ExpectedRecords: 33,
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
			ExpectedRecords: 33,
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

func TestGetSet(t *testing.T) {
	testCases := getSetTestCase()
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			logharbour.Index = "logharbour"
			tc.ActualResponse, err = logharbour.GetSet("", typedClient, tc.SetAttribute, tc.GetSetParam)

			if tc.ExpectedError {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				require.NoError(t, err)

			}
			// Compare the set Values
			if !reflect.DeepEqual(tc.ExpectedResponse, tc.ActualResponse) {
				t.Errorf("logsets are not equal. Expected: %v, Actual: %v", tc.ExpectedResponse, tc.ActualResponse)
			}

		})
	}

}

func getSetTestCase() []GetSetTestCasesStruct {
	app := "amazon"
	typeConst := logharbour.Activity
	who := "Jannie"
	class := "config"
	instance := "3"
	remoteIP := "142.250.67.206"
	priority := logharbour.Info
	days := 100
	fromTs := time.Date(2024, 02, 01, 00, 00, 00, 00, time.UTC)
	toTs := time.Date(2024, 05, 30, 00, 00, 00, 00, time.UTC)
	setAttribute := "app"
	InvalidSetAttribute := "when"

	expectedData := map[string]int64{"amazon": 1}

	expectedDataForApp := map[string]int64{"amazon": 33, "crux": 28, "flipkart": 31, "logharbour": 39, "rigel": 32, "starmf": 38}

	getSetTestCase := []GetSetTestCasesStruct{{
		Name: "SUCCESS : GetSet() with valid method parameters",
		GetSetParam: logharbour.GetSetParam{
			App:      &app,
			Type:     &typeConst,
			Who:      &who,
			Class:    &class,
			Instance: &instance,
			Fromts:   &fromTs,
			Tots:     &toTs,
			Ndays:    &days,
			RemoteIP: &remoteIP,
			Pri:      &priority,
		},
		SetAttribute:     setAttribute,
		ExpectedResponse: expectedData,
		ExpectedError:    false,
	}, {
		Name:             "SUCCESS : GetSet() with valid setAttribute",
		SetAttribute:     "app",
		GetSetParam:      logharbour.GetSetParam{},
		ExpectedResponse: expectedDataForApp,
		ExpectedError:    false,
	}, {
		Name:          "ERROR : GetSet() with Invalid SetAttribute ",
		SetAttribute:  InvalidSetAttribute,
		GetSetParam:   logharbour.GetSetParam{},
		ExpectedError: true,
	}, {
		Name:         "ERROR : Getset() with Invalid Time",
		SetAttribute: setAttribute,
		GetSetParam: logharbour.GetSetParam{
			App:      &app,
			Fromts:   &toTs,
			Tots:     &fromTs,
			Ndays:    new(int),
			RemoteIP: new(string),
		},
		ExpectedError: true,
	},
	}
	return getSetTestCase
}

func TestGetApps(t *testing.T) {
	testCases := getAppTestCase()
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {

			logharbour.Index = "logharbour"
			tc.ActualResponse, err = logharbour.GetApps("", typedClient)
			if tc.ExpectedError {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				require.NoError(t, err)

			}
			// Compare the responses
			if !reflect.DeepEqual(tc.ExpectedResponse, tc.ActualResponse) {
				t.Errorf("response are not equal. Expected: %v, Actual: %v", tc.ExpectedResponse, tc.ActualResponse)
			}
		})
	}

}

func getAppTestCase() []GetAppsTestCasesStruct {
	getAppTestCase := []GetAppsTestCasesStruct{{
		Name:             "SUCCESS : valid response",
		ExpectedResponse: []string{"crux", "flipkart", "rigel", "amazon", "starmf", "logharbour"},
		ExpectedError:    false,
	}}
	return getAppTestCase

}

func TestGetUnusualIP(t *testing.T) {
	testCases := getUnusualIPTestCase()
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {

			logharbour.Index = "logharbour"
			tc.ActualIps, err = logharbour.GetUnusualIP("", typedClient, tc.unusualPercent, tc.GetUnusualIPParam)

			if tc.ExpectError {
				if err == nil {
					t.Errorf("Expected error for input %d, but got nil", tc.GetUnusualIPParam)
				}
			} else {
				require.NoError(t, err)
			}

			// Compare the LogEntries
			if !reflect.DeepEqual(tc.ExpectedIps, tc.ActualIps) {
				t.Errorf("IPs are not equal. Expected: %v, Actual: %v", tc.ExpectedIps, tc.ActualIps)
			}
		})
	}

}

func getUnusualIPTestCase() []GetUnusualIPTestCaseStruct {
	app := "starmf"
	// who := "tushar"
	// class := "wfinstance"
	tasteCase := []GetUnusualIPTestCaseStruct{
		{
			Name:              "1st test for empty param",
			GetUnusualIPParam: logharbour.GetUnusualIPParam{},
			unusualPercent:    0.0,
			ExpectError:       true,
		},
		{
			Name: "2st test for unusualPercent 12",
			GetUnusualIPParam: logharbour.GetUnusualIPParam{
				App: &app,
				// Who:   &who,
				// Class: &class,
			},
			unusualPercent: 1.0,
			ExpectedIps:    []string{"142.250.67.2077"},
			ExpectError:    false,
		},
	}
	return tasteCase
}
