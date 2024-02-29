package wsc

import (
	"context"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/sortorder"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/remiges-tech/alya/service"
	"github.com/remiges-tech/alya/wscutils"
)

var size = 5

type LogRequest struct {
	App                  string  `json:"app" validate:"required"`
	Who                  *string `json:"who,omitempty"`
	Class                *string `json:"class,omitempty"`
	InstanceID           *string `json:"instance_id,omitempty"`
	Days                 int     `json:"days" validate:"required"`
	SearchAfterTimestamp *string `json:"search_after_timestamp,omitempty"`
	SearchAfterDocID     *string `json:"search_after_doc_id,omitempty"`
	SortID               *int    `json:"sort_id,omitempty"`
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
	var matchFeild []types.Query

	if req.App != "" {
		app := types.Query{
			Match: map[string]types.MatchQuery{
				"app": {Query: req.App},
			},
		}
		matchFeild = append(matchFeild, app)
	}

	if req.Who != nil {
		who := types.Query{
			Match: map[string]types.MatchQuery{
				"who": {Query: *req.Who},
			},
		}
		matchFeild = append(matchFeild, who)
	}

	if req.Class != nil {
		class := types.Query{
			Match: map[string]types.MatchQuery{
				"class": {Query: *req.Class},
			},
		}
		matchFeild = append(matchFeild, class)
	}

	if req.InstanceID != nil {
		instanceId := types.Query{
			Match: map[string]types.MatchQuery{
				"instance": {Query: *req.InstanceID},
			},
		}
		matchFeild = append(matchFeild, instanceId)
	}

	logtype := types.Query{
		Match: map[string]types.MatchQuery{
			"type": {Query: "A"},
		},
	}
	matchFeild = append(matchFeild, logtype)

	query := &types.Query{
		Bool: &types.BoolQuery{
			Must: matchFeild,
		},
	}

	sortByWhen := types.SortOptions{
		SortOptions: map[string]types.FieldSort{
			"when": {Order: &sortorder.Desc},
		},
	}

	// sortById := types.SortOptions{
	// 	SortOptions: map[string]types.FieldSort{
	// 		"_id": {Order: &sortorder.Desc},
	// 	},
	// }

	if req.SortID != nil {
		res, err := es.Search().Index(index).Request(&search.Request{
			Size:        &size,
			Query:       query,
			SearchAfter: []types.FieldValue{req.SortID},
			Sort:        []types.SortCombinations{sortByWhen},
		}).Do(context.Background())
		if err != nil {
			// write log here
			wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(222, err.Error()))
			return
		}
		wscutils.SendSuccessResponse(c, wscutils.NewSuccessResponse(res.Hits))
	} else {
		res, err := es.Search().Index(index).Request(&search.Request{
			Size:  &size,
			Query: query,
			Sort:  []types.SortCombinations{sortByWhen},
		}).Do(context.Background())
		if err != nil {
			wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(222, err.Error()))
			return
		}
		wscutils.SendSuccessResponse(c, wscutils.NewSuccessResponse(res.Hits))
	}

}
