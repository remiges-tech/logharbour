package logharbour_test

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	elasticsearchctl "github.com/remiges-tech/logharbour/server/elasticSearchCtl/elasticSearch"
)

var filepath = "../test/testData/log.json"
var createIndexBody = `{
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

var indexName = "logharbour_unit_test"
var typedClient *elasticsearch.TypedClient

func TestMain(m *testing.M) {
	// Initialize Docker pool to insure it close at the end
	pool, err := dockertest.NewPool("")
	if err != nil {
		fmt.Printf("Could not connect to Docker: %v\n", err)
	}
	fmt.Println("Initialize Docker pool")

	// Deploy the ElasticSearch container.
	resource, elasticAddress, err := deployElasticSearch(pool)
	if err != nil {
		fmt.Printf("Could not start resource: %v\n", err)
	}
	fmt.Println("Deploy the ElasticSearch container")

	// fillElasticWithData
	if err := fillElasticWithData(elasticAddress, indexName, createIndexBody); err != nil {
		fmt.Printf("Error while inserting data in elastic search: %v\n", err)
	}

	// Run the tests.
	exitCode := m.Run()

	// Exit with the appropriate code.
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}
	fmt.Println("Exit with the appropriate code")

	os.Exit(exitCode)

}

func deployElasticSearch(pool *dockertest.Pool) (*dockertest.Resource, string, error) {
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "elasticsearch",
		Tag:        "8.12.2",
		Env:        []string{"discovery.type=single-node"},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start elasticsearch: %s", err)
		return resource, "", err
	}

	elasticAddress := fmt.Sprintf("http://localhost:%s", resource.GetPort("9200/tcp"))

	log.Println("Connecting to elasticAddress on url: ", elasticAddress)

	resource.Expire(20) // Tell docker to hard kill the container in 120 seconds
	pool.MaxWait = 20 * time.Second

	// Ensure the elastic search container is ready to accept connections.
	// if err = pool.Retry(func() error {
	// Ping the Elasticsearch cluster
	// _, err = typedClient.Ping().Do(context.Background())
	// if err != nil {
	// 	log.Fatalf("Error pinging the Elasticsearch cluster: %s", err)
	// }

	// Check the response status
	// if res.IsError() {
	// 	log.Fatalf("Elasticsearch cluster returned an error: %s", res.String())
	// }

	// Print the response status
	// fmt.Println("Elasticsearch cluster is up and running!")
	// }); err != nil {
	// 	log.Fatalf("Could not connect to docker: %s", err)
	// }

	return resource, elasticAddress, nil
}

func fillElasticWithData(elasticAddress, indexName, indexBody string) error {

	cfg := elasticsearch.Config{
		Addresses: []string{elasticAddress},
		// Username:               username,
		// Password:               password,
		// CertificateFingerprint: esCer,
	}

	typedClient, err := elasticsearch.NewTypedClient(cfg)
	if err != nil {
		return fmt.Errorf("error while elastic search client config: %v", err)
	}
	fmt.Println("typedClient", typedClient)

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("error while elastic search client config: %v", err)
	}

	if err := elasticsearchctl.CreateElasticIndex(client, indexName, indexBody); err != nil {
		return fmt.Errorf("error while creating elastic search index: %v", err)
	}

	logEntries, err := elasticsearchctl.ReadLogFromFile(filepath)
	if err != nil {
		return fmt.Errorf("error converting data from log file:%v", err)
	}

	if err := elasticsearchctl.InsertLog(client, logEntries, indexName); err != nil {
		return fmt.Errorf("error while inserting data in elastic search: %v", err.Error())
	}

	return nil

}
