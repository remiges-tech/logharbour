package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-gonic/gin"
	"github.com/remiges-tech/alya/config"
	"github.com/remiges-tech/alya/service"
	"github.com/remiges-tech/alya/wscutils"
	"github.com/remiges-tech/logharbour/logharbour"
	"github.com/remiges-tech/logharbour/server/types"
	"github.com/remiges-tech/logharbour/server/wsc"
)

/********************************************************************************
* NOTE : We have used our *colleague's pc as DB over the same network to
* 		 establish connection please modify it accordingly in your json file and
*		 make sure "db_host" starts with prefix "https://"
*********************************************************************************/
func main() {
	var appConfig *types.AppConfig

	// Load error code and msg's
	errorCodeSetup()

	err := config.LoadConfigFromFile("./config_dev_aniket.json", &appConfig)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// logger setup
	fallbackWriter := logharbour.NewFallbackWriter(os.Stdout, os.Stdout)
	lctx := logharbour.NewLoggerContext(logharbour.Debug0)
	l := logharbour.NewLogger(lctx, "logharbour", fallbackWriter)

	flag.Parse()

	// Database connection
	url := appConfig.DBHost + ":" + strconv.Itoa(appConfig.DBPort)

	dbConfig := elasticsearch.Config{
		Addresses:              []string{url},
		Username:               appConfig.DBUser,
		Password:               appConfig.DBPassword,
		CertificateFingerprint: appConfig.CertificateFingerprint,
	}
	client, err := elasticsearch.NewTypedClient(dbConfig)
	if err != nil {
		log.Fatalf("Failed to create db connection: %v", err)
		wscutils.NewErrorResponse(502, "error establishing a database connection")
		return
	}

	// router
	r := gin.Default()

	// r.Use(corsMiddleware())

	// services
	s := service.NewService(r).
		WithLogHarbour(l).
		WithDependency("client", client).
		WithDependency("index", appConfig.IndexName)

	apiV1Group := r.Group("/api/v1/")

	s.RegisterRouteWithGroup(apiV1Group, http.MethodGet, "/highprilog", wsc.GetHighprilog)

	r.Run(":" + appConfig.AppServerPort)
	if err != nil {
		l.LogActivity("Failed to start server", err)
		log.Fatalf("Failed to start server: %v", err)
	}

}

func errorCodeSetup() {
	// Define a custom validation tag-to-message ID map
	customValidationMap := map[string]int{
		"required":  101,
		"gt":        102,
		"alpha":     103,
		"lowercase": 104,
	}
	// Custom validation tag-to-error code map
	customErrCodeMap := map[string]string{
		"required":  "required",
		"gt":        "greater",
		"alpha":     "alphabet",
		"lowercase": "lowercase",
	}
	// Register the custom map with wscutils
	wscutils.SetValidationTagToMsgIDMap(customValidationMap)
	wscutils.SetValidationTagToErrCodeMap(customErrCodeMap)
	// Set default message ID and error code if needed
	wscutils.SetDefaultMsgID(100)
	wscutils.SetDefaultErrCode("validation_error")
}
