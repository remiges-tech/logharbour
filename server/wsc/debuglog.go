package wsc

import (
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/remiges-tech/alya/service"
	"github.com/remiges-tech/alya/wscutils"
	"github.com/remiges-tech/logharbour/logharbour"
)

var (
	DebugPriority = []string{"Debug2", "Debug1", "Debug0", "Info", "Warn", "Err", "Crit", "Sec"}
)

// request format
type DebugLogRequest struct {
	App                  string                 `json:"app" validate:"required,alpha,lt=25"`
	Module               string                 `json:"module" validate:"required,alpha"`
	Priority             logharbour.LogPriority `json:"pri" validate:"required"`
	Days                 int                    `json:"days" validate:"required,number"`
	TraceID              *string                `json:"trace_id" validate:"omitempty"`
	SearchAfterTimestamp *string                `json:"search_after_timestamp,omitempty"`
	SearchAfterDocId     *string                `json:"search_after_doc_id,omitempty"`
}

func GetDebugLog(c *gin.Context, s *service.Service) {
	lh := s.LogHarbour.WithClass("logharbour")

	var (
		request DebugLogRequest
		LogType = logharbour.Debug
	)

	// bind request
	err := wscutils.BindJSON(c, &request)
	if err != nil {
		lh.Err().Error(err).Log("GetDebugLog||error while binding json request")
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(404, err.Error()))
		return
	}

	lh.Debug2().LogDebug("GetDebugLog ||request parameters:", request)

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
		lh.Debug0().Log("client dependency not found")
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(100, "ErrCode_DatabaseError"))
		return
	}

	// call GetLogs()
	debuglogs, count, err := logharbour.GetLogs("", client, logharbour.GetLogsParam{
		App:              &request.App,
		Module:           &request.Module,
		Type:             &LogType,
		NDays:            &request.Days,
		Priority:         &request.Priority,
		SearchAfterTS:    request.SearchAfterTimestamp,
		SearchAfterDocID: request.SearchAfterDocId,
	})
	if err != nil {
		lh.Err().Error(err).Log("error while retriving data from db")
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(222, err.Error()))
		return
	}

	lh.Debug0().LogActivity("exit from GetDebugLog with recordCount:", count)
	wscutils.SendSuccessResponse(c, wscutils.NewSuccessResponse(debuglogs))
}
