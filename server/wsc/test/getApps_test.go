package wsc_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/remiges-tech/alya/wscutils"
	"github.com/remiges-tech/logharbour/server/wsc/test/testUtils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppList(t *testing.T) {
	testCases := AppListTestcase()
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			res := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, "/getapps", nil)
			require.NoError(t, err)

			// Assuming `r` is your router or handler
			r.ServeHTTP(res, req)

			require.Equal(t, tc.ExpectedHttpCode, res.Code)

			var actualResult wscutils.Response
			err = json.Unmarshal(res.Body.Bytes(), &actualResult)
			require.NoError(t, err)

			expectedData := tc.ExpectedResult.Data.([]interface{})
			actualData := actualResult.Data.([]interface{})

			assert.ElementsMatch(t, expectedData, actualData)
			assert.Equal(t, tc.ExpectedResult.Status, actualResult.Status)
			assert.Equal(t, tc.ExpectedResult.Messages, actualResult.Messages)
		})
	}
}

func AppListTestcase() []testUtils.TestCasesStruct {
	schemaNewTestCase := []testUtils.TestCasesStruct{
		{
			Name:             "successful",
			ExpectedHttpCode: http.StatusOK,
			ExpectedResult: &wscutils.Response{
				Status:   "success",
				Data:     []interface{}{"rigel", "amazon", "starmf", "logharbour", "crux", "flipkart"},
				Messages: nil,
			},
		},
	}
	return schemaNewTestCase
}
