package wsc

import (
	"context"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/sortorder"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/remiges-tech/alya/service"
	"github.com/remiges-tech/alya/wscutils"
)

type LogRequest struct {
	App                  string  `json:"app" validate:"required"`
	Who                  string  `json:"who,omitempty"`
	Class                string  `json:"class,omitempty"`
	InstanceID           string  `json:"instance_id,omitempty"`
	Days                 int     `json:"days" validate:"required"`
	SearchAfterTimestamp *string `json:"search_after_timestamp,omitempty"`
	SearchAfterDocID     *string `json:"search_after_doc_id,omitempty"`
}

func ShowActivitylog(c *gin.Context, s *service.Service) {
	l := s.LogHarbour
	l.Debug0().Log("starting execution of ShowActivitylog()")

	var req LogRequest

	err := wscutils.BindJSON(c, &req)
	if err != nil {
		l.Debug0().Error(err).Log("error unmarshalling request payload to struct")
		return
	}
	fmt.Println("req", req)

	// Validate request
	validationErrors := wscutils.WscValidate(req, func(err validator.FieldError) []string { return []string{} })
	if len(validationErrors) > 0 {
		l.Debug0().LogDebug("standard validation errors", validationErrors)
		wscutils.SendErrorResponse(c, wscutils.NewResponse(wscutils.ErrorStatus, nil, validationErrors))
		return
	}

	es, ok := s.Dependencies["client"].(*elasticsearch.TypedClient)
	if !ok {
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(100, "ErrCode_DatabaseError"))
		return
	}
	index, ok := s.Dependencies["index"].(string)
	if !ok {
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(404, "ErrCode_IndexNotFound"))
		return
	}

	app := types.Query{
		Match: map[string]types.MatchQuery{
			"app_name": {Query: req.App},
		},
	}
	logtype := types.Query{
		Match: map[string]types.MatchQuery{
			"type": {Query: "Activity"},
		},
	}
	who := types.Query{
		Match: map[string]types.MatchQuery{
			"who": {Query: req.Who},
		},
	}

	class := types.Query{
		Match: map[string]types.MatchQuery{
			"class": {Query: req.Class},
		},
	}

	instanceId := types.Query{
		Match: map[string]types.MatchQuery{
			"instance_id": {Query: req.InstanceID},
		},
	}

	// searchAfterTimestamp := types.Query{
	// 	Match: map[string]types.MatchQuery{
	// 		"search_after_timestamp": {Query: req.SearchAfterTimestamp},
	// 	},
	// }

	// searchAfterDocID := types.Query{
	// 	Match: map[string]types.MatchQuery{
	// 		"search_after_doc_id": {Query: req.SearchAfterDocID},
	// 	},
	// }

	query := &types.Query{
		Bool: &types.BoolQuery{
			Must:   []types.Query{logtype, app},
			Should: []types.Query{class, instanceId, who},
		},
	}
	// q := &types.Query{
	// 	Bool: &types.BoolQuery{
	// 		Must: []types.Query{types.Query{
	// 			Match: map[string]types.MatchQuery{
	// 				"type":     {Query: "Activity"},
	// 				"app_name": {Query: req.App},
	// 			},
	// 		}},
	// 		Should: []types.Query{class, instanceId, who},
	// 	},
	// }

	if req.SearchAfterDocID != nil || req.SearchAfterTimestamp != nil {
		from := 1
		size := 5

		sortOpti := types.SortOptions{
			SortOptions: map[string]types.FieldSort{
				"when":     {Order: &sortorder.Desc},
				"app_name": {Order: &sortorder.Desc},
			},
		}
		res, err := es.Search().Index(index).Request(&search.Request{
			From:        &from,
			Size:        &size,
			Query:       query,
			SearchAfter: []types.FieldValue{},
			Sort:        []types.SortCombinations{sortOpti},
		}).Do(context.Background())
		if err != nil {
			// write log here
			wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(222, err.Error()))
			return
		}
		wscutils.SendSuccessResponse(c, wscutils.NewSuccessResponse(res.Hits))

	} else {

		from := 1
		size := 5

		sortOpti := types.SortOptions{
			SortOptions: map[string]types.FieldSort{
				"when": {Order: &sortorder.Asc},
			},
		}
		res, err := es.Search().Index(index).Request(&search.Request{
			From:  &from,
			Size:  &size,
			Query: query,
			Sort:  []types.SortCombinations{sortOpti},
		}).Do(context.Background())
		if err != nil {
			// write log here
			wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(222, err.Error()))
			return
		}
		wscutils.SendSuccessResponse(c, wscutils.NewSuccessResponse(res.Hits))
	}

}
