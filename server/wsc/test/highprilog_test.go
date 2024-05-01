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

const (
	TestHighpriLog_1 = "ERROR_1- slice validation"
	TestHighpriLog_2 = "SUCCESS_2- get data by valid req"
)

func TestHighpriLog(t *testing.T) {
	testCases := highpriLogTestcase()
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// Setting up buffer
			payload := bytes.NewBuffer(testUtils.MarshalJson(tc.RequestPayload))

			res := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodPost, "/highprilog", payload)
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

func highpriLogTestcase() []testUtils.TestCasesStruct {
	schemaNewTestCase := []testUtils.TestCasesStruct{
		// 1st test case
		{
			Name: TestHighpriLog_1,
			RequestPayload: wscutils.Request{
				Data: wsc.HighPriReq{},
			},

			ExpectedHttpCode: http.StatusBadRequest,
			ExpectedResult: &wscutils.Response{
				Status: wscutils.ErrorStatus,
				Data:   nil,
				Messages: []wscutils.ErrorMessage{
					{
						MsgID:   1001,
						ErrCode: "invalid_json",
					},
				},
			},
		},
		// 2nd test case
		{
			Name: TestHighpriLog_2,
			RequestPayload: wscutils.Request{
				Data: wsc.HighPriReq{
					App:  "crux",
					Pri:  logharbour.Info,
					Days: 1000,
				},
			},

			ExpectedHttpCode: http.StatusOK,
			TestJsonFile:     "./data/high_pri_response.json",
		},
	}
	return schemaNewTestCase
}
