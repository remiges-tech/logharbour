package logharbour_test

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
	"github.com/remiges-tech/logharbour/logharbour"
	estestutils "github.com/remiges-tech/logharbour/logharbour/test"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	indexBody   = logharbour.ESLogsMapping
	typedClient *es.TypedClient
	filepath  = "../test/testData/testData.json"
	indexName = "logharbour"
	timeout   = 500 * time.Second
)

func TestMain(m *testing.M) {

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Define container request for Elasticsearch.
	// Using official Elasticsearch image with security disabled for testing
	req := testcontainers.ContainerRequest{
		Image:        "docker.elastic.co/elasticsearch/elasticsearch:8.12.0",
		ExposedPorts: []string{"9200/tcp"},
		Env: map[string]string{
			"discovery.type":         "single-node",
			"xpack.security.enabled": "false",
			"ES_JAVA_OPTS":           "-Xms512m -Xmx512m",
		},
		WaitingFor: wait.ForHTTP("/").WithPort("9200").WithStartupTimeout(timeout),
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

	if err := fillElasticWithData(esClient, indexName, indexBody, filepath); err != nil {
		log.Fatalf("error while creating elastic search index: %v", err)
	}

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
