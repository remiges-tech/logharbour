package wsc_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/remiges-tech/alya/wscutils"
	"github.com/remiges-tech/logharbour/server/wsc"
	"github.com/remiges-tech/logharbour/server/wsc/test/testUtils"
	"github.com/stretchr/testify/require"
)

const (
	TestDataChangeLog_1 = "ERROR_1- slice validation"
	TestDataChangeLog_2 = "SUCCESS_2- get data by valid req"
)

func TestDataChangeLog(t *testing.T) {
	testCases := dataChangeLogTestcase()
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// Setting up buffer
			payload := bytes.NewBuffer(testUtils.MarshalJson(tc.RequestPayload))

			res := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodPost, "/datachange", payload)
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

func dataChangeLogTestcase() []testUtils.TestCasesStruct {
	app := "crux"
	class := "schema"
	instance := "1"
	days := 1000
	schemaNewTestCase := []testUtils.TestCasesStruct{
		// 1st test case
		{
			Name: TestDataChangeLog_1,
			RequestPayload: wscutils.Request{
				Data: wsc.DataChangeReq{},
			},

			ExpectedHttpCode: http.StatusBadRequest,
			ExpectedResult: &wscutils.Response{
				Status: wscutils.ErrorStatus,
				Data:   nil,
				Messages: []wscutils.ErrorMessage{
					{

						MsgID:   101,
						ErrCode: "required",
						Field:   &wsc.APP,
					},
				},
			},
		},
		// 2nd test case
		{
			Name: TestDataChangeLog_2,
			RequestPayload: wscutils.Request{
				Data: wsc.DataChangeReq{
					App:      app,
					Class:    &class,
					Instance: &instance,
					Days:     &days,
				},
			},

			ExpectedHttpCode: http.StatusOK,
			TestJsonFile:     "./data/data_change_response.json",
		},
	}
	return schemaNewTestCase
}
