package wsc_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/remiges-tech/alya/wscutils"
	"github.com/remiges-tech/logharbour/logharbour"
	"github.com/remiges-tech/logharbour/server/wsc"
	"github.com/remiges-tech/logharbour/server/wsc/test/testUtils"
	"github.com/stretchr/testify/require"
)

func TestShowDebugLogs(t *testing.T) {
	testCases := showDebugLogTestCase()
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// Setting up buffer
			payload := bytes.NewBuffer(testUtils.MarshalJson(tc.RequestPayload))

			res := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodPost, "/debuglog", payload)
			require.NoError(t, err)

			r.ServeHTTP(res, req)

			require.Equal(t, tc.ExpectedHttpCode, res.Code)
			if tc.ExpectedResult != nil {
				jsonData := testUtils.MarshalJson(tc.ExpectedResult)
				require.JSONEq(t, string(jsonData), res.Body.String())
			} else {
				jsonData, err := testUtils.ReadJsonFromFile(tc.TestJsonFile)
				require.NoError(t, err)
				require.JSONEq(t, string(jsonData), res.Body.String())
			}
		})
	}
}

func showDebugLogTestCase() []testUtils.TestCasesStruct {
	app := "crux"
	module := "login"
	days := 20
	invalidApp := "cr12@"
	debugTestCases := []testUtils.TestCasesStruct{{
		Name: "SUCCESS : with valid parameters ",
		RequestPayload: wscutils.Request{
			Data: wsc.DebugLogRequest{
				App:                  app,
				Module:               module,
				Priority:             logharbour.Debug0,
				Days:                 days,
				SearchAfterTimestamp: nil,
				SearchAfterDocId:     nil,
			},
		},
		ExpectedHttpCode: http.StatusOK,
		TestJsonFile:     "./data/debuglog_valid_response.json",
	}, {
		Name:             "ERROR : with invalid request",
		RequestPayload:   wscutils.Request{Data: wsc.DebugLogRequest{App: invalidApp, Module: module, Priority: logharbour.Debug0, Days: days}},
		ExpectedHttpCode: http.StatusBadRequest,
		TestJsonFile:     "./data/debuglog_invalid_request_res.json",
	},
	}
	return debugTestCases
}
