package logharbour_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	es "github.com/elastic/go-elasticsearch/v8"
	elasticsearchctl "github.com/remiges-tech/logharbour/server/elasticSearchCtl/elasticSearch"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/elasticsearch"
)

var (
	indexBody = `{
		"settings": {
		  "number_of_shards": 1,
		  "number_of_replicas": 0
		},
		"mappings": {
		  "properties": {
			"app": {
			  "type": "keyword"
			},
			"system": {
			  "type": "keyword"
			},
			"module": {
			  "type": "keyword"
			},
			"type": {
			  "type": "keyword"
			},
			"pri": {
			  "type": "keyword"
			},
			"when": {
			  "type": "date"
			},
			"who": {
			  "type": "keyword"
			},
			"op": {
			  "type": "keyword"
			},
			"class": {
			  "type": "keyword"
			},
			"instance": {
			  "type": "keyword"
			},
			"status": {
			  "type": "integer"
			},
			"error": {
			  "type": "keyword"
			},
			"remote_ip": {
			  "type": "ip"
			},
			"msg": {
			  "type": "keyword"
			},
			"data": {
			  "type": "text"
			}
		  }
		}
	  }`
	typedClient *es.TypedClient
	filepath    = "../test/testData/testData.json"
	indexName   = "logharbour"
	timeout     = 500 * time.Second
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

	if err := fillElasticWithData(esClient, indexName, indexBody, filepath); err != nil {
		log.Fatalf("error while creating elastic search index: %v", err)
	}

	exitVal := m.Run()
	os.Exit(exitVal)
}

func fillElasticWithData(esClient *es.Client, indexName, indexBody, filepath string) error {

	if err := elasticsearchctl.CreateElasticIndex(esClient, indexName, indexBody); err != nil {
		return fmt.Errorf("error while creating elastic search index: %v", err)
	}

	logEntries, err := elasticsearchctl.ReadLogFromFile(filepath)
	if err != nil {
		return fmt.Errorf("error converting data from log file:%v", err)
	}

	if err := elasticsearchctl.InsertLog(esClient, logEntries, indexName); err != nil {
		return fmt.Errorf("error while inserting data in elastic search: %v", err.Error())
	}

	return nil

}
