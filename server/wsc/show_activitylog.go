package wsc

import (
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/remiges-tech/alya/service"
	"github.com/remiges-tech/alya/wscutils"
	"github.com/remiges-tech/logharbour/logharbour"
)

var LogType = logharbour.Activity

// var fromTs = time.Date(2024, 02, 01, 00, 00, 00, 00, time.UTC)
// var toTs = time.Date(2024, 03, 01, 00, 00, 00, 00, time.UTC)

type LogRequest struct {
	App        string  `json:"app" validate:"required"`
	Who        *string `json:"who,omitempty"`
	Class      *string `json:"class,omitempty"`
	InstanceID *string `json:"instance_id,omitempty"`
	Op         *string `json:"op,omitempty"`
	Priority   *string `json:"priority,omitempty"`
	Days       int     `json:"days"`
	// FromTS               *string `json:"fromTs,omitempty"`
	// ToTS                 *string `json:"toTs,omitempty"`
	SearchAfterTimestamp *string `json:"search_after_timestamp,omitempty"`
	SearchAfterDocID     *string `json:"search_after_doc_id,omitempty"`
	SortID               *int    `json:"sort_id,omitempty"`
}

type LogResponse struct {
	LogEntery []logharbour.LogEntry
	Nrec      int
}

func ShowActivitylog(c *gin.Context, s *service.Service) {
	l := s.LogHarbour
	l.Debug0().Log("starting execution of ShowActivitylog()")

	var req LogRequest
	var res LogResponse

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
	// index, ok := s.Dependencies["index"].(string)
	// if !ok {
	// 	wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(404, "ErrCode_IndexNotFound"))
	// 	return
	// }
	// fromTs := req.FromTS.Format(layout)
	// toTs := req.ToTS.Format(layout)
	res.LogEntery, res.Nrec, err = logharbour.GetLogs("", es, logharbour.GetLogsParam{
		App:       &req.App,
		Type:      &LogType,
		Who:       req.Who,
		Class:     req.Class,
		Instance:  req.InstanceID,
		Operation: req.Op,
		// FromTS:    &fromTs,
		// ToTS:      &toTs,
		NDays: &req.Days,
		// RemoteIP:         req,
		Priority:         req.Priority,
		SearchAfterTS:    req.SearchAfterTimestamp,
		SearchAfterDocID: req.SearchAfterDocID,
	})
	if err != nil {
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(222, err.Error()))
		return
	}
	fmt.Println("res.nrec", len(res.LogEntery))
	wscutils.SendSuccessResponse(c, wscutils.NewSuccessResponse(res))
}
