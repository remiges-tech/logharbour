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
func TestShowActivityLog(t *testing.T) {
    testCases := showActivityLogTestcase()
    for _, tc := range testCases {
        t.Run(tc.Name, func(t *testing.T) {
            // Setting up buffer
            payload := bytes.NewBuffer(testUtils.MarshalJson(tc.RequestPayload))
            t.Logf("Running test case: %s", tc.Name)
            t.Logf("Request payload: %s", payload.String())

            res := httptest.NewRecorder()
            logharbour.Index = "logharbour_unit_test1"
            req, err := http.NewRequest(http.MethodPost, "/activitylog", payload)
            if err != nil {
                t.Errorf("Error creating new HTTP request: %v", err)
            }
            require.NoError(t, err)

            r.ServeHTTP(res, req)
            t.Logf("HTTP response code: %d", res.Code)
            t.Logf("HTTP response body: %s", res.Body.String())

            require.Equal(t, tc.ExpectedHttpCode, res.Code)
            if tc.ExpectedResult != nil {
                jsonData := testUtils.MarshalJson(tc.ExpectedResult)
                require.JSONEq(t, string(jsonData), res.Body.String())
            } else {
                jsonData, err := testUtils.ReadJsonFromFile(tc.TestJsonFile)
                if err != nil {
                    t.Errorf("Error reading JSON from file: %v", err)
                }
                require.NoError(t, err)
                require.JSONEq(t, string(jsonData), res.Body.String())
            }
        })
    }
}


func showActivityLogTestcase() []testUtils.TestCasesStruct {
	schemaNewTestCase := []testUtils.TestCasesStruct{
		{
			Name: "err- binding_json_error",
			RequestPayload: wscutils.Request{
				Data: nil,
			},

			ExpectedHttpCode: http.StatusBadRequest,
			ExpectedResult: &wscutils.Response{
				Status: wscutils.ErrorStatus,
				Data:   nil,
				Messages: []wscutils.ErrorMessage{
					{
						MsgID:   0,
						ErrCode: "",
					},
				},
			},
		},
		{
			Name: "err- standard validation",
			RequestPayload: wscutils.Request{
				Data: wsc.LogRequest{},
			},

			ExpectedHttpCode: http.StatusBadRequest,
			TestJsonFile:     "./data/show_activitylog_test/standerd_validation.json",
		},
		// {
		// 	Name: "successful",
		// 	RequestPayload: wscutils.Request{
		// 		Data: wsc.LogRequest{
		// 			App:  "crux",
		// 			Days: 100,
		// 		},
		// 	},
		// 	ExpectedHttpCode: http.StatusOK,
		// 	TestJsonFile:     "./data/show_activitylog_test/successful_test_case.json",
		// },
	}
	return schemaNewTestCase
}
