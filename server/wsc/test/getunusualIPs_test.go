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

func TestGetUnusualIPs(t *testing.T) {
	testCases := GetUnusualIPsTestcase()
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// Setting up buffer
			payload := bytes.NewBuffer(testUtils.MarshalJson(tc.RequestPayload))

			res := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodPost, "/getunusualips", payload)
			require.NoError(t, err)

			r.ServeHTTP(res, req)

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

func GetUnusualIPsTestcase() []testUtils.TestCasesStruct {
	GetUnusualIPsTestcase := []testUtils.TestCasesStruct{

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
			Name: "ERROR:  standard validation",
			RequestPayload: wscutils.Request{
				Data: wsc.GetUnusualIpParam{},
			},
			ExpectedHttpCode: http.StatusBadRequest,
			TestJsonFile:     "./data/getunusualip_standard_err.json",
		},
		{
			Name: "SUCCESS: Valid response",
			RequestPayload: wscutils.Request{
				Data: wsc.GetUnusualIpParam{
					App:            "crux",
					Days:           100,
					UnusualPercent: 10,
				},
			},
			ExpectedHttpCode: http.StatusOK,
			TestJsonFile:     "./data/getunusualip_valid_reponse.json",
		},
	}
	return GetUnusualIPsTestcase
}
