package logharbour_test

import (
	"reflect"
	"sort"
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
				var err error
				tc.ExpectedLogEntries, err = estestutils.ReadLogFromFile(tc.TestJsonFile)
				if err != nil {
					t.Fatalf("error converting data from log file: %v", err)
				}
			}

			logharbour.Index = "logharbour"
			var err error
			tc.ActualLogEntries, tc.ActualRecords, err = logharbour.GetLogs("", typedClient, tc.LogsParam)

			if tc.ExpectError {
				if err == nil {
					t.Errorf("Expected error for input %d, but got nil", tc.ActualRecords)
				}
			} else {
				require.NoError(t, err)
			}

			// Sort the log entries before comparing
			SortLogEntries(tc.ExpectedLogEntries)
			SortLogEntries(tc.ActualLogEntries)

			// Compare the LogEntries
			if !reflect.DeepEqual(tc.ExpectedLogEntries, tc.ActualLogEntries) {
				t.Errorf("LogEntries are not equal. Expected: %v, Actual: %v", tc.ExpectedLogEntries, tc.ActualLogEntries)
			}

			// Compare number of records
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
	//searchAfterTS := "2024-02-25T07:28:00.110813597Z"
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
			Name:               "3rd_tc_without_filterParam",
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

	expectedDataForApp := map[string]int64{"amazon": 33, "crux": 28, "flipkart": 31, "logharbour": 39, "rigel": 32, "starmf": 39}

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
			// Sort the slices for comparison
			sort.Strings(tc.ExpectedResponse)
			sort.Strings(tc.ActualResponse)

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

			// Sort the slices for comparison
			sort.Strings(tc.ExpectedIps)
			sort.Strings(tc.ActualIps)

			// Compare the sorted slices
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
			unusualPercent: 3.0,
			ExpectedIps: []string{
				"23.84.90.201",
				"31.130.239.24",
				"117.217.12.191",
				"163.43.120.140",
				"b31c:acfd:bb0a:5d4c:b0bf:fa69:a88e:dba7",
				"b33c:eac5:25e7:714c:3eeb:5c53:df6c:abad",
				"55.107.177.20",
				"4dfb:94:f7f5:bc9f:f5e1:7f44:4fb8:ce8d",
				"fa4a:afea:f19d:9b91:bdd4:fb43:eb3b:ad81",
				"71.34.12.118",
				"80.252.56.229",
				"107.248.246.217",
				"68c4:aeef:d3b0:ad7b:ceda:bdad:c59a:ec9d",
				"fa4c:ea18:e131:e0db:5ace:c8ed:a6eb:4b37",
				"fdbf:41cb:fa1a:1d8:d9d6:6f01:8e0:8ea7",
				"100.0.115.205",
				"107.234.177.79",
				"203.25.117.108",
				"64.131.117.177",
				"71.155.48.60",
				"218.14.74.156",
				"d80a:ccc2:e938:fce0:3eee:2dfc:bd8f:4afa",
				"f7ec:e265:e67e:cdaf:ce3b:f8e7:f4da:7a6b",
				"96.10.45.253",
				"131.187.187.204",
				"224.250.161.71",
				"237.128.19.53",
				"253.137.0.217",
				"4f2b:91ba:d292:3baf:a1cf:24c0:ba9b:f6aa",
				"93.228.80.75",
				"150.125.249.182",
				"205.102.136.112",
				"cc69:fca:2559:cd94:51f2:d39c:ce8f:eb3d",
				"142.250.67.29",
				"173.101.122.105",
				"3a87:482d:a52d:8ba0:cddc:e16c:abd9:1762",
				"b7bc:e5c1:5fd8:4390:fcce:df73:acca:aedd",
			},

			ExpectError: false,
		},
	}
	return tasteCase
}

// SortLogEntries sorts log entries by their ID.
func SortLogEntries(logs []logharbour.LogEntry) {
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Id < logs[j].Id
	})
}
