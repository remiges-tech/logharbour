package wsc_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/remiges-tech/alya/wscutils"
	"github.com/remiges-tech/logharbour/server/wsc/test/testUtils"
	"github.com/stretchr/testify/require"
)

func TestAppList(t *testing.T) {
	testCases := AppListTestcase()
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			res := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, "/applist", nil)
			require.NoError(t, err)

			r.ServeHTTP(res, req)

			require.Equal(t, tc.ExpectedHttpCode, res.Code)
			jsonData := testUtils.MarshalJson(tc.ExpectedResult)
			require.JSONEq(t, string(jsonData), res.Body.String())

		})
	}
}

func AppListTestcase() []testUtils.TestCasesStruct {
	schemaNewTestCase := []testUtils.TestCasesStruct{
		{
			Name:             "successful",
			ExpectedHttpCode: http.StatusOK,
			ExpectedResult:   wscutils.NewSuccessResponse([]string{"idshield", "crux"}),
		},
	}
	return schemaNewTestCase
}
