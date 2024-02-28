package wsc

import (
	"context"
	"fmt"
	"slices"
	"strconv"

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
	SearchAfterDocId     string `json:"search_after_doc_id"`
}

func GetHighprilog(c *gin.Context, s *service.Service) {
	fmt.Println("<<<<<<<<<<<<<<<<<< inside GetHighprilog")
	var (
		from    int
		request HighPriReq
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
	fmt.Println(">>>>>>", requiredPri)

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

	pageSize := 10
	page := 1
	fmt.Println("==============", request.SearchAfterDocId)
	// Calculate the `From` parameter based on the page and page size
	if request.SearchAfterDocId == "" || &request.SearchAfterDocId == nil {
		fmt.Println("==============inside nil")
		from = (page - 1) * pageSize
	} else {
		from, err = strconv.Atoi(request.SearchAfterDocId)
		if err != nil {
			wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(404, "ErrCode_NotANumber"))
			return
		}
	}

	// WORKING ____________________________________________________________
	fmt.Println("from:", from, " pageSize:", pageSize)
	sortOrder := types.SortOptions{
		SortOptions: map[string]types.FieldSort{
			"when": {Order: &sortorder.Asc},
		}}

	searchQuery, err := clnt.Search().
		Index(index).From(from).Size(pageSize).
		Request(&search.Request{
			Query: &types.Query{Match: map[string]types.MatchQuery{
				"priority": {Query: request.Pri},
			}},
			Sort:        []types.SortCombinations{sortOrder},
			SearchAfter: []types.FieldValue{from},
		}).Do(context.Background())
	// WORKING ____________________________________________________________

	if err != nil {
		// write log here
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(222, err.Error()))
		return
	}
	// write log here
	wscutils.SendSuccessResponse(c, wscutils.NewSuccessResponse(searchQuery.Hits.Hits))
	// wscutils.SendSuccessResponse(c, wscutils.NewSuccessResponse(res.Hits))
}
