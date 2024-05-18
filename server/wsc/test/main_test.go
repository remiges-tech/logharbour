package wsc_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	es "github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-gonic/gin"
	"github.com/remiges-tech/alya/service"
	"github.com/remiges-tech/alya/wscutils"
	"github.com/remiges-tech/logharbour/logharbour"
	estestutils "github.com/remiges-tech/logharbour/logharbour/test"
	"github.com/remiges-tech/logharbour/server/wsc"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/elasticsearch"
)

var (
	indexBody = logharbour.ESLogsMapping
	typedClient      *es.TypedClient
	r                *gin.Engine
	seedDataFilePath = "../test/seed_data.json"
	indexName        = "logharbour"
	timeout          = 100 * time.Second
)

func TestMain(m *testing.M) {

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// creates an instance of the Elasticsearch container
	elasticsearchContainer, err := elasticsearch.RunContainer(
		ctx,
		testcontainers.WithImage("docker.elastic.co/elasticsearch/elasticsearch:8.9.0"),
		elasticsearch.WithPassword("elastic"),
	)
	if err != nil {
		log.Fatalf("failed to start container: %s", err)
	}
	defer func() {
		err := elasticsearchContainer.Terminate(ctx)
		if err != nil {
			log.Fatalf("failed to terminate container: %s", err)
		}
	}()

	cfg := es.Config{
		Addresses: []string{
			elasticsearchContainer.Settings.Address,
		},
		Username: "elastic",
		Password: elasticsearchContainer.Settings.Password,
		CACert:   elasticsearchContainer.Settings.CACert,
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
		log.Fatalf("error while creating elastic search index: %v", err)
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
		return fmt.Errorf("error while creating elastic search index: %v", err)
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
	s.RegisterRoute(http.MethodPost, "/getUnusualIP", wsc.GetUnusualIPs)
	s.RegisterRoute(http.MethodPost, "/datachange", wsc.ShowDataChange)
	s.RegisterRoute(http.MethodGet, "/getapps", wsc.GetApps)
	s.RegisterRoute(http.MethodPost, "/getset", wsc.GetSet)

	return r, nil

}
