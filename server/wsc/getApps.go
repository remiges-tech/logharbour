package wsc

import (
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-gonic/gin"
	"github.com/remiges-tech/alya/service"
	"github.com/remiges-tech/alya/wscutils"
	"github.com/remiges-tech/logharbour/logharbour"
)

func GetApps(c *gin.Context, s *service.Service) {
	l := s.LogHarbour
	l.Debug0().Log("starting execution of GetApps()")

	es, ok := s.Dependencies["client"].(*elasticsearch.TypedClient)
	if !ok {
		l.Debug0().Log("Error while getting elasticsearch instance from service Dependencies")
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(MsgId_InternalErr, ErrCode_DatabaseError))
		return
	}
	apps, err := logharbour.GetApps(queryToken, es)
	if err != nil {
		errmsg := errorHandler(err)
		l.Debug0().Error(err).Log("error in AppList")
		wscutils.SendErrorResponse(c, wscutils.NewResponse(wscutils.ErrorStatus, nil, []wscutils.ErrorMessage{errmsg}))
		return
	}
	wscutils.SendSuccessResponse(c, wscutils.NewSuccessResponse(apps))
}
