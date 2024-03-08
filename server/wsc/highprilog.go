package wsc

import (
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/remiges-tech/alya/service"
	"github.com/remiges-tech/alya/wscutils"
	"github.com/remiges-tech/logharbour/logharbour"
)

// var (
// 	Priority = []string{"Debug2", "Debug1", "Debug0", "Info", "Warn", "Err", "Crit", "Sec"}
// )

type HighPriReq struct {
	App string                 `json:"app" validate:"required,alpha"`
	Pri logharbour.LogPriority `json:"pri" validate:"required"`
	// Pri  logharbour.LogPriority `json:"pri" validate:"required, oneof=Info Debug2 Debug1 Debug0 Warn Err Crit Sec"`
	Days                 int    `json:"days" validate:"number,required"`
	SearchAfterTimestamp string `json:"search_after_timestamp" validate:"omitempty,datetime=2006-01-02T15:04:05Z"`
	SearchAfterDocId     string `json:"search_after_doc_id,omitempty"`
}

// GetHighprilog : handler for POST: "/highprilog" API
func GetHighprilog(c *gin.Context, s *service.Service) {
	lh := s.LogHarbour
	lh.Debug0().Log("starting execution of GetHighprilog()")
	var (
		request  HighPriReq
		srchAftr []types.FieldValue
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

	if request.SearchAfterDocId != "" || len(request.SearchAfterDocId) > 0 {
		srchAftr = append(srchAftr, request.SearchAfterDocId)
	}

	searchQuery, recordCount, err := logharbour.GetLogs("", esClient, logharbour.GetLogsParam{
		App:      &request.App,
		Priority: &request.Pri,
		NDays:    &request.Days,
		// FromTS:    &fromTs,
		// ToTS:      &toTs,
		// RemoteIP:         req,
		// SearchAfterTS: &request.SearchAfterTimestamp,
	})
	if err != nil {
		lh.Err().Error(err).Log("error while retriving data from db")
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(222, err.Error()))
		return
	}

	lh.Info().Log("exit from GetHighprilog")
	wscutils.SendSuccessResponse(c, wscutils.NewSuccessResponse(map[string]any{"logs": searchQuery, "count": recordCount}))
}
