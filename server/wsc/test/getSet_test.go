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

func TestGetSet(t *testing.T) {
	testCases := GetSetTestcase()
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// Setting up buffer
			payload := bytes.NewBuffer(testUtils.MarshalJson(tc.RequestPayload))

			res := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodPost, "/getset", payload)
			require.NoError(t, err)

			r.ServeHTTP(res, req)

			require.Equal(t, tc.ExpectedHttpCode, res.Code)
			jsonData := testUtils.MarshalJson(tc.ExpectedResult)
			require.JSONEq(t, string(jsonData), res.Body.String())
		})
	}
}

func GetSetTestcase() []testUtils.TestCasesStruct {
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
				Data: wsc.GetSetReq{
					App: "",
				},
			},
			ExpectedHttpCode: http.StatusBadRequest,
			ExpectedResult: &wscutils.Response{
				Status: wscutils.ErrorStatus,
				Data:   nil,
				Messages: []wscutils.ErrorMessage{
					{
						MsgID:   101,
						ErrCode: "required",
						Field:   str("App"),
					},
				},
			},
		},
		{
			Name: "successful",
			RequestPayload: wscutils.Request{
				Data: wsc.GetSetReq{
					App:         "crux",
					SetAttr:     "class",
					GetSetParam: logharbour.GetSetParam{},
				},
			},
			ExpectedHttpCode: http.StatusOK,
			ExpectedResult:   wscutils.NewSuccessResponse([]string{"abc"}),
		},
	}
	return schemaNewTestCase
}

func str(str string) *string {
	return &str
}
