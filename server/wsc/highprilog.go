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
	Priority = []string{"Debug2", "Debug1", "Debug0", "Info", "Warn", "Err", "Crit", "Sec"}
)

type HighPriReq struct {
	App                  string `json:"app" validate:"required,alpha"`
	Pri                  string `json:"pri" validate:"required,alpha"`
	Days                 int64  `json:"days" validate:"number,required"`
	SearchAfterTimestamp string `json:"search_after_timestamp" validate:"omitempty,datetime=2006-01-02"`
	SearchAfterDocId     string `json:"search_after_doc_id,omitempty"`
}

func GetHighprilog(c *gin.Context, s *service.Service) {
	lh := s.LogHarbour
	lh.Debug0().Log("starting execution of GetHighprilog()")
	var (
		request     HighPriReq
		srchAftr    []types.FieldValue
		pageSize    = 10
		requiredPri []string
	)
	// step 1: json request binding with a struct
	err := wscutils.BindJSON(c, &request)
	if err != nil {
		lh.Err().Error(err).Log("error while binding json request error")
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(404, err.Error()))
		return
	}

	// Validate request
	validationErrors := wscutils.WscValidate(request, func(err validator.FieldError) []string { return []string{} })
	if len(validationErrors) > 0 {
		lh.Debug0().LogDebug("standard validation errors", validationErrors)
		wscutils.SendErrorResponse(c, wscutils.NewResponse(wscutils.ErrorStatus, nil, validationErrors))
		return
	}

	//________________Day calculation___________________________________________________
	afDays := fmt.Sprintf("now-%dd", request.Days)

	//___________________________________________________________________
	priFrom := slices.Index(Priority, request.Pri)
	if priFrom > 0 {
		requiredPri = Priority[priFrom:]
	}

	clnt, ok := s.Dependencies["client"].(*elasticsearch.TypedClient)
	if !ok {
		lh.Debug0().Log("client dependency not found")
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(100, "ErrCode_DatabaseError"))
		return
	}
	index, ok := s.Dependencies["index"].(string)
	if !ok {
		lh.Debug0().Log("index dependency not found")
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(404, "ErrCode_IndexNotFound"))
		return
	}

	if request.SearchAfterDocId != "" || len(request.SearchAfterDocId) > 0 {
		srchAftr = append(srchAftr, request.SearchAfterDocId)
	}

	// WORKING ____________________________________________________________
	sortOrder := types.SortOptions{
		SortOptions: map[string]types.FieldSort{
			"when": {Order: &sortorder.Desc},
			// "status": {Order: &sortorder.Asc},
			// "_id":  {Order: &sortorder.Desc},
		}}

	var qry []types.Query

	for _, v := range requiredPri {
		termQry := types.Query{
			Match: map[string]types.MatchQuery{
				"pri": {Query: v},
			},
		}
		qry = append(qry, termQry)
	}
	// WORKING ____________________________________________________________
	searchQuery, err := clnt.Search().
		Index(index).
		Request(&search.Request{
			Size: &pageSize,
			Query: &types.Query{
				Bool: &types.BoolQuery{
					Filter: []types.Query{
						types.Query{
							Range: map[string]types.RangeQuery{
								"when": types.DateRangeQuery{
									Gte: &afDays,
								},
							},
						},
						types.Query{
							Match: map[string]types.MatchQuery{
								"app": types.MatchQuery{
									Query: request.App,
								},
							},
						},
					},
					Should: qry,
				},
			},
			Sort:        []types.SortCombinations{sortOrder},
			SearchAfter: srchAftr,
		}).Do(context.Background())
	// WORKING ____________________________________________________________

	if err != nil {
		lh.Err().Error(err).Log("error while retriving data from db")
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(222, err.Error()))
		return
	}
	lh.Info().Log("exit from GetHighprilog")
	wscutils.SendSuccessResponse(c, wscutils.NewSuccessResponse(searchQuery.Hits.Hits))
}
