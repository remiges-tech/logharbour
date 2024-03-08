package main

import (
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

	err := config.LoadConfigFromFile("./config_dev_kanchan.json", &appConfig)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// logger setup
	fallbackWriter := logharbour.NewFallbackWriter(os.Stdout, os.Stdout)
	lctx := logharbour.NewLoggerContext(logharbour.Debug0)
	l := logharbour.NewLogger(lctx, "logharbour", fallbackWriter)

	// rigelAppName := flag.String("appName", "logharbour", "The name of the application")
	// rigelModuleName := flag.String("moduleName", "WSC", "The name of the module")
	// rigelVersionNumber := flag.Int("versionNumber", 1, "The number of the version")
	// rigelConfigName := flag.String("configName", "devConfig", "The name of the configuration")
	// etcdEndpoints := flag.String("etcdEndpoints", "localhost:2379", "Comma-separated list of etcd endpoints")

	// flag.Parse()
	// // Create a new EtcdStorage instance
	// etcdStorage, err := etcd.NewEtcdStorage([]string{*etcdEndpoints})
	// if err != nil {
	// 	l.LogActivity("Error while Creating new instance of EtcdStorage", err)
	// 	log.Fatalf("Failed to create EtcdStorage: %v", err)
	// }
	// l.LogActivity("Creates a new instance of EtcdStorage with endpoints", "localhost:2379")

	// // Create a new Rigel instance
	// rigel := rigel.New(etcdStorage, *rigelAppName, *rigelModuleName, *rigelVersionNumber, *rigelConfigName)
	// l.LogActivity("Creates a new instance of rigel", rigel)

	// // Create a context with a timeout
	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()

	// dbHost, err := rigel.GetString(ctx, "db_host")
	// if err != nil {
	// 	l.LogActivity("Error while getting db_host from rigel", err)
	// 	log.Fatalf("Failed to get db_host from rigel: %v", err)
	// }

	// dbPort, err := rigel.GetInt(ctx, "db_port")
	// if err != nil {
	// 	l.LogActivity("Error while getting db_port from rigel", err)
	// 	log.Fatalf("Failed to get db_port from rigel: %v", err)
	// }

	// dbUser, err := rigel.GetString(ctx, "db_user")
	// if err != nil {
	// 	l.LogActivity("Error while getting db_user from rigel", err)
	// 	log.Fatalf("Failed to get db_user from rigel: %v", err)
	// }

	// dbPassword, err := rigel.GetString(ctx, "db_password")
	// if err != nil {
	// 	l.LogActivity("Error while getting db_password from rigel", err)
	// 	log.Fatalf("Failed to get db_password from rigel: %v", err)
	// }

	// certificateFingerprint, err := rigel.GetString(ctx, "certificate_fingerprint")
	// if err != nil {
	// 	l.LogActivity("Error while getting certificate_fingerprint from rigel", err)
	// 	log.Fatalf("Failed to get certificate_fingerprint from rigel: %v", err)
	// }

	// appServerPort, err := rigel.GetString(ctx, "app_server_port")
	// if err != nil {
	// 	l.LogActivity("Error while getting app_server_port from rigel", err)
	// 	log.Fatalf("Failed to get app_server_port from rigel: %v", err)
	// }

	// Load error code and msg's
	errorCodeSetup()
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
		WithDependency("index", "logharbour")

	apiV1Group := r.Group("/api/v1/")

	s.RegisterRouteWithGroup(apiV1Group, http.MethodPost, "/highprilog", wsc.GetHighprilog)
	s.RegisterRouteWithGroup(apiV1Group, http.MethodPost, "/showActivitylog", wsc.ShowActivitylog)
	s.RegisterRouteWithGroup(apiV1Group, http.MethodPost, "/show_debuglog", wsc.GetDebugLog)

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
