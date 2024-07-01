package wsc_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/remiges-tech/alya/wscutils"
	"github.com/remiges-tech/logharbour/server/wsc"
	"github.com/remiges-tech/logharbour/server/wsc/test/testUtils"
	"github.com/stretchr/testify/assert"
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

			// Assuming `r` is your router or handler
			r.ServeHTTP(res, req)

			require.Equal(t, tc.ExpectedHttpCode, res.Code)

			var actualResult wscutils.Response
			err = json.Unmarshal(res.Body.Bytes(), &actualResult)
			require.NoError(t, err)

			expectedData, ok := tc.ExpectedResult.Data.([]interface{})
			if ok {
				actualData, ok := actualResult.Data.([]interface{})
				if ok {
					assert.ElementsMatch(t, expectedData, actualData)
				} else {
					t.Errorf("Expected Data to be []interface{}, got %T", actualResult.Data)
				}
			} else {
				assert.Equal(t, tc.ExpectedResult.Data, actualResult.Data)
			}

			assert.Equal(t, tc.ExpectedResult.Status, actualResult.Status)
			assert.ElementsMatch(t, tc.ExpectedResult.Messages, actualResult.Messages)
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
					{
						MsgID:   101,
						ErrCode: "required",
						Field:   str("SetAttr"),
					},
				},
			},
		},
		{
			Name: "successful",
			RequestPayload: wscutils.Request{
				Data: wsc.GetSetReq{
					App:     "crux",
					SetAttr: "class",
				},
			},
			ExpectedHttpCode: http.StatusOK,
			ExpectedResult: &wscutils.Response{
				Status: wscutils.SuccessStatus,
				Data:   []interface{}{"ruleset", "app", "config", "schema"},
				Messages: nil,
			},
		},
	}
	return schemaNewTestCase
}

func str(s string) *string {
	return &s
}
