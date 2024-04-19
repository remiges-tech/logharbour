package wsc

import (
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/remiges-tech/alya/service"
	"github.com/remiges-tech/alya/wscutils"
	"github.com/remiges-tech/logharbour/logharbour"
)

type GetSetReq struct {
	App         string                 `json:"app" validate:"required,alpha,lt=50"`
	SetAttr     string                 `json:"setAttr" validate:"required,alpha,lt=50"`
	GetSetParam logharbour.GetSetParam `json:"GetSetParam,omitempty"`
}

func GetSet(c *gin.Context, s *service.Service) {
	l := s.LogHarbour
	l.Debug0().Log("starting execution of GetList()")
	var getSetReq GetSetReq

	err := wscutils.BindJSON(c, &getSetReq)
	if err != nil {
		l.Debug0().Error(err).Log("error unmarshalling request payload to struct")
		return
	}

	// Validate request
	validationErrors := wscutils.WscValidate(getSetReq, func(err validator.FieldError) []string { return []string{} })
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
	getSetReq.GetSetParam.App = &getSetReq.App

	if getSetReq.SetAttr == "field" {
		getSetReq.SetAttr = "data.changes.field"
	}
	res, err := logharbour.GetSet(queryToken, es, getSetReq.SetAttr, getSetReq.GetSetParam)
	if err != nil {
		errmsg := errorHandler(err)
		l.Debug0().Error(err).Log("error in GetLogs")
		wscutils.SendErrorResponse(c, wscutils.NewResponse(wscutils.ErrorStatus, nil, []wscutils.ErrorMessage{errmsg}))
		return
	}

	var list []string
	for l := range res {
		list = append(list, l)
	}
	wscutils.SendSuccessResponse(c, wscutils.NewSuccessResponse(list))
}
