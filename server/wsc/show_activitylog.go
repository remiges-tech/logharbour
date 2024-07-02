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
var queryToken string

// var fromTs = time.Date(2024, 02, 01, 00, 00, 00, 00, time.UTC)
// var toTs = time.Date(2024, 03, 01, 00, 00, 00, 00, time.UTC)

type LogRequest struct {
	App        string                  `json:"app" validate:"required,alpha,lt=15"`
	Who        *string                 `json:"who" validate:"omitempty,alpha,lt=15"`
	Class      *string                 `json:"class" validate:"omitempty,alpha,lt=15"`
	InstanceID *string                 `json:"instance_id" validate:"omitempty,alphanum,lt=15"`
	Op         *string                 `json:"op" validate:"omitempty,alpha,lt=15"`
	Priority   *logharbour.LogPriority `json:"priority" validate:"omitempty,lt=15"`
	Days       int                     `json:"days" validate:"required,number,lt=500"`
	// FromTS               *string                 `json:"fromTs" validate:"omitempty,alpha,lt=15"`
	// ToTS                 *string                 `json:"toTs" validate:"omitempty,alpha,lt=15"`
	SearchAfterTimestamp *string `json:"search_after_timestamp" validate:"omitempty"`
	SearchAfterDocID     *string `json:"search_after_doc_id" validate:"omitempty"`
}

type LogResponse struct {
	LogEntery []logharbour.LogEntry
	Nrec      int
}

func ShowActivityLog(c *gin.Context, s *service.Service) {
	l := s.LogHarbour
	l.Debug0().Log("starting execution of ShowActivityLog()")

	var req LogRequest
	var res LogResponse
	var pri logharbour.LogPriority

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
		l.Debug0().Log("Error while getting elasticsearch instance from service Dependencies")
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(MsgId_InternalErr, ErrCode_DatabaseError))
		return
	}

	res.LogEntery, res.Nrec, err = logharbour.GetLogs(queryToken, es, logharbour.GetLogsParam{
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
		Priority:         &pri,
		SearchAfterTS:    req.SearchAfterTimestamp,
		SearchAfterDocID: req.SearchAfterDocID,
	})
	if err != nil {
		fmt.Println("error>>>>>>>>>>>>>>>>>>",err)
		errmsg := errorHandler(err)
		l.Debug0().Error(err).Log("error in GetLogs")
		wscutils.SendErrorResponse(c, wscutils.NewResponse(wscutils.ErrorStatus, nil, []wscutils.ErrorMessage{errmsg}))
		return
	}
	fmt.Println("res.nrec", len(res.LogEntery))
	wscutils.SendSuccessResponse(c, wscutils.NewSuccessResponse(res))
}

func errorHandler(err error) wscutils.ErrorMessage {
	switch err.Error() {
	case "tots must be after fromts":
		return wscutils.BuildErrorMessage(MsgId_Invalid_Request, ErrCode_InvalidRequest, nil)
	case "No Filter param":
		return wscutils.BuildErrorMessage(MsgId_Invalid_Request, ErrCode_InvalidRequest, nil)
	}
	return wscutils.BuildErrorMessage(MsgId_InternalErr, ErrCode_DatabaseError, nil)

}
