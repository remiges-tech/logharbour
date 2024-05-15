package wsc

import (
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/remiges-tech/alya/service"
	"github.com/remiges-tech/alya/wscutils"
	"github.com/remiges-tech/logharbour/logharbour"
)

// DataChangeReq: is for request of ShowDataChange()
type DataChangeReq struct {
	App                  string  `json:"app" validate:"required,alpha,lt=15"`
	Who                  *string `json:"who"`
	Class                *string `json:"class"`
	Instance             *string `json:"instance"`
	Field                *string `json:"field"`
	Days                 *int    `json:"days" validate:"omitempty,gt=0,lt=1003"`
	SearchAfterTimestamp *string `json:"search_after_timestamp" validate:"omitempty,datetime=2006-01-02T15:04:05Z"`
	SearchAfterDocId     *string `json:"search_after_doc_id,omitempty"`
}

// ShowDataChange : handler for POST: "/datachangelog" API
func ShowDataChange(c *gin.Context, s *service.Service) {
	lh := s.LogHarbour
	lh.Debug0().Log("starting execution of ShowDataChange()")
	var (
		request     DataChangeReq
		recordCount int
		searchQuery []logharbour.LogEntry
	)

	// step 1: json request binding with a struct
	err := wscutils.BindJSON(c, &request)
	if err != nil {
		lh.Err().Error(err).Log("error while binding json request error")
		return
	}

	// Validate request
	validationErrors := wscutils.WscValidate(request, func(err validator.FieldError) []string { return []string{} })
	if len(validationErrors) > 0 {
		lh.Debug0().LogDebug("standard validation errors", validationErrors)
		wscutils.SendErrorResponse(c, wscutils.NewResponse(wscutils.ErrorStatus, nil, validationErrors))
		return
	}

	esClient, ok := s.Dependencies["client"].(*elasticsearch.TypedClient)
	if !ok {
		lh.Debug0().Log("client dependency not found")
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(100, "ErrCode_DatabaseError"))
		return
	}

	// response := logharbour.Change
	searchQuery, recordCount, err = logharbour.GetChanges("", esClient, logharbour.GetLogsParam{
		App:              &request.App,
		Who:              request.Who,
		Class:            request.Class,
		Instance:         request.Instance,
		Field:            request.Field,
		NDays:            request.Days,
		SearchAfterTS:    request.SearchAfterTimestamp,
		SearchAfterDocID: request.SearchAfterDocId,
	})

	if err != nil {
		lh.Err().Error(err).Log("error while retriving data from db")
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(222, err.Error()))
		return
	}

	// var jsonMap map[string]interface{}
	// var actualResult []logharbour.LogEntry
	// for _, v := range searchQuery {
	// 	if len(fmt.Sprint(v.Data)) > 0 {
	// 		err := json.Unmarshal([]byte(fmt.Sprint(v.Data)), &jsonMap)
	// 		if err == nil {
	// 			if v.Data {

	// 			}
	// 		}
	// 	}
	// }

	lh.Info().LogActivity("exit from GetHighprilog with recordCount:", recordCount)
	wscutils.SendSuccessResponse(c, wscutils.NewSuccessResponse(map[string]any{"logs": searchQuery}))
	// wscutils.SendSuccessResponse(c, wscutils.NewSuccessResponse(map[string]any{"logs": searchQuery}))
}
