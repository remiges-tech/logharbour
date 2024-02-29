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
	"github.com/remiges-tech/alya/service"
	"github.com/remiges-tech/alya/wscutils"
	// "github.com/remiges-tech/logharbour/server/types"
)

var (
	Priority = []string{"Debug2", "Debug1", "Debug0", "Info", "Warn", "Err", "Crit", "Sec"}
)

type HighPriReq struct {
	App                  string `json:"app" validation:"alpha"`
	Pri                  string `json:"pri"`
	Days                 int64  `json:"days"`
	SearchAfterTimestamp string `json:"search_after_timestamp"`
	SearchAfterDocId     string `json:"search_after_doc_id,omitempty"`
}

func GetHighprilog(c *gin.Context, s *service.Service) {
	var (
		request  HighPriReq
		srchAftr []types.FieldValue
		pageSize = 10
	)
	// step 1: json request binding with a struct
	err := wscutils.BindJSON(c, &request)
	if err != nil {
		// lh.Debug0().LogActivity("error while binding json request error:", err.Error())
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(404, err.Error()))
		return
	}

	priFrom := slices.Index(Priority, request.Pri)

	requiredPri := Priority[priFrom:]

	clnt, ok := s.Dependencies["client"].(*elasticsearch.TypedClient)
	if !ok {
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(100, "ErrCode_DatabaseError"))
		return
	}
	index, ok := s.Dependencies["index"].(string)
	if !ok {
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(404, "ErrCode_IndexNotFound"))
		return
	}

	if request.SearchAfterDocId != "" || len(request.SearchAfterDocId) > 0 {
		srchAftr = append(srchAftr, request.SearchAfterDocId)
	}

	// WORKING ____________________________________________________________
	fmt.Println("index:", index, " from:", request.SearchAfterDocId, " pageSize:", pageSize)
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

	searchQuery, err := clnt.Search().
		Index(index).
		Request(&search.Request{
			Size: &pageSize,
			Query: &types.Query{
				Bool: &types.BoolQuery{
					Should: qry,
				},
			},
			Sort:        []types.SortCombinations{sortOrder},
			SearchAfter: srchAftr,
		}).Do(context.Background())
	// WORKING ____________________________________________________________

	if err != nil {
		// write log here
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(222, err.Error()))
		return
	}
	// write log here
	wscutils.SendSuccessResponse(c, wscutils.NewSuccessResponse(searchQuery.Hits.Hits))
}
