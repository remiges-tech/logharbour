package wsc

import (
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/oschwald/geoip2-golang"
	"github.com/remiges-tech/alya/service"
	"github.com/remiges-tech/alya/wscutils"
	"github.com/remiges-tech/logharbour/logharbour"
)

//var unusualPercent = 10.0

type GetUnusualIpParam struct {
	App            string  `json:"app" validate:"required,alpha,lt=15"`
	Days           int     `json:"days" validate:"required,number,lt=500"`
	UnusualPercent float64 `json:"unusualPercent"`
}
type GetUnusualIPResponse struct {
	UnusualIPs []logharbour.IPLocation `json:"unusualIPs" validate:"required"`
}

func GetUnusualIPs(c *gin.Context, s *service.Service) {
	l := s.LogHarbour
	l.Debug0().Log("starting execution of GetUnusualIPs()")

	var (
		req        GetUnusualIpParam
		unusualIPs []logharbour.IPLocation
	)

	// binding json request
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

	geoLiteDb, ok := s.Dependencies["geoLiteCityDb"].(*geoip2.Reader)
	if !ok {
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(100, "ErrCode_DatabaseError"))
		return
	}

	unusualIPs, err = logharbour.ListUnusualIPs("", es, geoLiteDb, req.UnusualPercent, logharbour.GetUnusualIPParam{
		App:   &req.App,
		NDays: &req.Days,
	})

	if err != nil {
		l.Debug0().Error(err).Log("error in GetUnusualIPs")
		wscutils.SendErrorResponse(c, wscutils.NewErrorResponse(222, err.Error()))
		return
	}
	if len(unusualIPs) != 0 {

		response := GetUnusualIPResponse{
			UnusualIPs: unusualIPs,
		}

		l.Debug0().Log("finished execution of GetUnusualIPs()")
		wscutils.SendSuccessResponse(c, wscutils.NewSuccessResponse(response))
		return
	}
	l.Debug0().Log("finished execution of GetUnusualIPs()")
	wscutils.SendSuccessResponse(c, wscutils.NewSuccessResponse("No unusual IP"))

}
