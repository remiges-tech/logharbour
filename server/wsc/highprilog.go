package wsc

import (
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-gonic/gin"
	"github.com/remiges-tech/alya/service"
	"github.com/remiges-tech/alya/wscutils"
	"github.com/remiges-tech/logharbour/server/types"
)

func GetHighprilog(c *gin.Context, s *service.Service) {
	var request types.LogReq
	// step 1: json request binding with a struct
	err := wscutils.BindJSON(c, &request)
	if err != nil {
		// lh.Debug0().LogActivity("error while binding json request error:", err.Error())
		return
	}

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

	res, err := clnt.Search().Index(index).Do(c)

	if err != nil {
		// write log here
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(222, err.Error()))
		return
	}
	// write log here
	wscutils.SendSuccessResponse(c, wscutils.NewSuccessResponse(res.Hits))
}
