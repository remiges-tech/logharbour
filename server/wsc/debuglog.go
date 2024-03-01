package wsc

import (
	"context"
	"fmt"
	"slices"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/sortorder"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/remiges-tech/alya/service"
	"github.com/remiges-tech/alya/wscutils"
)

var (
	DebugPriority = []string{"Debug2", "Debug1", "Debug0", "Info", "Warn", "Err", "Crit", "Sec"}
)

// request format
type DebugLogRequest struct {
	App                  string `json:"app" validate:"required,alpha"`
	Module               string `json:"module" validate:"required,alpha"`
	Priority             string `json:"pri" validate:"required,alpha"`
	Days                 int    `json:"days" validate:"required,number"`
	TraceID              string `json:"trace_id" validate:"omitempty"`
	SearchAfterTimestamp string `json:"search_after_timestamp,omitempty" validate:"omitempty"`
	SearchAfterDocId     string `json:"search_after_doc_id,omitempty"`
}

func GetDebugLog(c *gin.Context, s *service.Service) {
	lh := s.LogHarbour.WithClass("logharbour")

	var (
		pageSize       = 10
		request        DebugLogRequest
		searchAfter    []types.FieldValue
		shouldFieldQry []types.Query
	)

	// bind request
	err := wscutils.BindJSON(c, &request)
	if err != nil {
		lh.Err().Error(err).Log("GetDebugLog||error while binding json request")
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(404, err.Error()))
		return
	}

	lh.Debug2().LogDebug("GetDebugLog ||request parameters:", request)

	afDay := fmt.Sprintf("now-%dd/d", request.Days)

	// Validate request
	validationErrors := wscutils.WscValidate(request, func(err validator.FieldError) []string { return []string{} })
	if len(validationErrors) > 0 {
		lh.Debug0().LogDebug("standard validation errors", validationErrors)
		wscutils.SendErrorResponse(c, wscutils.NewResponse(wscutils.ErrorStatus, nil, validationErrors))
		return
	}
	// assert dependencies
	client, ok := s.Dependencies["client"].(*elasticsearch.TypedClient)
	if !ok {
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(100, "ErrCode_DatabaseError"))
		return
	}
	index, ok := s.Dependencies["index"].(string)
	if !ok {
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(404, "ErrCode_IndexNotFound"))
		return
	}

	priFrom := slices.Index(Priority, request.Priority)

	requiredPri := Priority[priFrom:]
	if request.SearchAfterDocId != "" || len(request.SearchAfterDocId) > 0 {
		searchAfter = append(searchAfter, request.SearchAfterDocId)
	}

	// app := types.Query{
	// 	Match: map[string]types.MatchQuery{
	// 		"app": {Query: request.App},
	// 	},
	// }
	// shouldFieldQry = append(shouldFieldQry, app)

	module := types.Query{
		Match: map[string]types.MatchQuery{
			"module": {Query: request.Module},
		},
	}
	shouldFieldQry = append(shouldFieldQry, module)

	for _, v := range requiredPri {
		pri := types.Query{
			Match: map[string]types.MatchQuery{
				"pri": {Query: v},
			},
		}
		shouldFieldQry = append(shouldFieldQry, pri)
	}

	// shouldFieldQry = append(shouldFieldQry, logType)

	sortOrder := types.SortOptions{
		SortOptions: map[string]types.FieldSort{
			"when": {Order: &sortorder.Desc},
		}}

	searchQuery, err := client.Search().
		Index(index).
		Request(&search.Request{
			Size: &pageSize,
			Query: &types.Query{
				Bool: &types.BoolQuery{
					Filter: []types.Query{
						types.Query{
							Range: map[string]types.RangeQuery{
								"when": types.DateRangeQuery{
									Gte: &afDay,
								},
							},
						},
						types.Query{
							Match: map[string]types.MatchQuery{
								"app": types.MatchQuery{
									Query: request.App,
								},
								"module": types.MatchQuery{
									Query: request.Module,
								},
								"type": types.MatchQuery{
									Query: D,
								},
							},
						},
					},
					Should: shouldFieldQry,
				},
			},
			Sort:        []types.SortCombinations{sortOrder},
			SearchAfter: []types.FieldValue{searchAfter},
		}).Do(context.Background())

	if err != nil {
		lh.Err().Error(err).Log("error while retriving data from db")
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(222, err.Error()))
		return
	}

	wscutils.SendSuccessResponse(c, wscutils.NewSuccessResponse(searchQuery.Hits.Hits))
}
