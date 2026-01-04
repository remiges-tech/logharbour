package wsc_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	es "github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-gonic/gin"
	"github.com/oschwald/geoip2-golang"
	"github.com/remiges-tech/alya/service"
	"github.com/remiges-tech/alya/wscutils"
	"github.com/remiges-tech/logharbour/logharbour"
	estestutils "github.com/remiges-tech/logharbour/logharbour/test"
	"github.com/remiges-tech/logharbour/server/wsc"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	indexBody        = logharbour.ESLogsMapping
	typedClient      *es.TypedClient
	r                *gin.Engine
	seedDataFilePath = "../../../logharbour/test/testData/testData.json"
	indexName        = "logharbour"
	timeout          = 1000 * time.Second
)

func TestMain(m *testing.M) {

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Define container request for Elasticsearch.
	// we are using bitnami elasticsearch for testing purpose so authentication details are not required
	req := testcontainers.ContainerRequest{
		Image:        "bitnami/elasticsearch:latest",
		ExposedPorts: []string{"9200/tcp"},
		WaitingFor:   wait.ForHTTP("/").WithPort("9200").WithStartupTimeout(timeout),
	}

	// Create and start the Elasticsearch container
	elasticsearchContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		log.Fatalf("failed to start container: %s", err)
	}
	defer func() {
		err := elasticsearchContainer.Terminate(ctx)
		if err != nil {
			log.Fatalf("failed to terminate container: %s", err)
		}
	}()
	// Get the host and port of the running container
	host, err := elasticsearchContainer.Host(ctx)
	if err != nil {
		log.Fatalf("failed to get container host: %s", err)
	}

	port, err := elasticsearchContainer.MappedPort(ctx, "9200")
	if err != nil {
		log.Fatalf("failed to get mapped port: %s", err)
	}

	cfg := es.Config{
		Addresses: []string{
			fmt.Sprintf("http://%s:%s", host, port.Port()),
		},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // Disable SSL verification
			},
		},
	}

	// NewTypedClient create a new elasticsearch client with the configuration from cfg
	typedClient, err = es.NewTypedClient(cfg)
	if err != nil {
		log.Fatalf("error while elastic search client config: %v", err)
	}

	// NewClient creates a new elasticsearch client with configuration from cfg.
	esClient, err := es.NewClient(cfg)
	if err != nil {
		log.Fatalf("error creating the client: %s", err)
	}

	if err := fillElasticWithData(esClient, indexName, indexBody, seedDataFilePath); err != nil {
		log.Fatalf("elasticdata error while creating elastic search index: %v", err)
	}

	// Register routes.
	r, err = registerRoutes(typedClient)
	if err != nil {
		log.Fatalf("Could not start resource: %v", err)
	}
	fmt.Println("Register routes")

	exitVal := m.Run()
	os.Exit(exitVal)
}

func fillElasticWithData(esClient *es.Client, indexName, indexBody, filepath string) error {

	if err := estestutils.CreateElasticIndex(esClient, indexName, indexBody); err != nil {
		return fmt.Errorf(" error while creating elastic search index: %v", err)
	}

	logEntries, err := estestutils.ReadLogFromFile(filepath)
	if err != nil {
		return fmt.Errorf("error converting data from log file:%v", err)
	}

	if err := estestutils.InsertLog(esClient, logEntries, indexName); err != nil {
		return fmt.Errorf("error while inserting data in elastic search: %v", err.Error())
	}

	return nil

}

// registerRoutes register and runs.
func registerRoutes(typedClient *es.TypedClient) (*gin.Engine, error) {

	geoLiteDbPath := "../../../logharbour/GeoLite2-City.mmdb" // path of  mmdb file
	// GeoLite2-City database
	geoLiteCityDb, err := geoip2.Open(geoLiteDbPath)
	if err != nil {
		log.Printf("Warning: Failed to create GeoLite2-City db connection: %v. IP geolocation tests will be skipped.", err)
		geoLiteCityDb = nil
	} else {
		defer geoLiteCityDb.Close()
	}

	// router
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	// logger setup
	fallbackWriter := logharbour.NewFallbackWriter(os.Stdout, os.Stdout)
	lctx := logharbour.NewLoggerContext(logharbour.Info)
	l := logharbour.NewLogger(lctx, "crux", fallbackWriter)

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

	// schema services
	s := service.NewService(r).
		WithLogHarbour(l).
		WithDependency("client", typedClient)

	s.RegisterRoute(http.MethodPost, "/highprilog", wsc.GetHighprilog)
	s.RegisterRoute(http.MethodPost, "/activitylog", wsc.ShowActivityLog)
	s.RegisterRoute(http.MethodPost, "/debuglog", wsc.GetDebugLog)
	s.RegisterRoute(http.MethodPost, "/datachange", wsc.ShowDataChange)
	s.RegisterRoute(http.MethodGet, "/getapps", wsc.GetApps)
	s.RegisterRoute(http.MethodPost, "/getset", wsc.GetSet)
	// creating a seprate service for getting list of unusualIPS with geoLiteCityDb dependency
	unusualIPServ := service.NewService(r).
		WithLogHarbour(l).
		WithDependency("client", typedClient).
		WithDependency("index", "logharbour").WithDependency("geoLiteCityDb", geoLiteCityDb)

	unusualIPServ.RegisterRoute(http.MethodPost, "/getunusualips", wsc.GetUnusualIPs)
	return r, nil
}
