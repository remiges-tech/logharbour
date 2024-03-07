package elasticsearchctl

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/remiges-tech/logharbour/logharbour"
)

func CreateElasticIndex(es *elasticsearch.Client, indexName string, indexBody string) error {

	// Create the index request
	req := esapi.IndicesCreateRequest{
		Index: indexName,
		Body:  strings.NewReader(indexBody),
	}

	// Perform the request
	res, err := req.Do(context.Background(), es)
	if err != nil {
		return fmt.Errorf("error creating the index: %s", err)
	}

	defer res.Body.Close()

	// Print the response status and body
	fmt.Println("Response status:", res.Status())
	if res.IsError() {
		var errorResponse map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&errorResponse); err != nil {
			return fmt.Errorf("error parsing the error response body: %s", err)
		}
		log.Fatalf("Error creating the index: %s", errorResponse["error"].(map[string]interface{})["reason"])
	} else {

		fmt.Println("Index created successfully:")
	}
	return nil

}

func InsertLog(es *elasticsearch.Client, logs []logharbour.LogEntry, indexName string) error {

	for i, log := range logs {
		dataJson, err := json.Marshal(log)
		if err != nil {
			return fmt.Errorf("error while unmarshaling log: %v", err)
		}

		js := string(dataJson)

		req := esapi.IndexRequest{
			Index:      indexName,
			DocumentID: strconv.Itoa(i + 1),
			Body:       strings.NewReader(js),
			Refresh:    "true",
		}

		res, err := req.Do(context.Background(), es)
		if err != nil {
			return fmt.Errorf("error while adding data in es :%v", err)
		}
		defer res.Body.Close()
		if res.IsError() {
			return fmt.Errorf("error indexing document ID=%s", res)
		}
	}
	return nil
}

func ReadLogFromFile(filepath string) ([]logharbour.LogEntry, error) {

	byteValue, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var LogEntries []logharbour.LogEntry

	err = json.Unmarshal(byteValue, &LogEntries)
	if err != nil {
		return nil, err
	}
	return LogEntries, nil
}

// func GetElasticsearch(filepath string) (*elasticsearch.Client, error) {

// 	bytedata, err := os.ReadFile(filepath)
// 	if err != nil {
// 		return nil, err
// 	}
// 	var ESConfig config
// 	err = json.Unmarshal(bytedata, &ESConfig)
// 	if err != nil {
// 		return nil, err
// 	}

// 	cfg := elasticsearch.Config{
// 		Addresses:              ESConfig.Addresses,
// 		Username:               ESConfig.Username,
// 		Password:               ESConfig.Password,
// 		CertificateFingerprint: ESConfig.CertificateFingerprint,
// 	}

// 	es, err := elasticsearch.NewClient(cfg)
// 	if err != nil {
// 		return nil, err
// 	}
// 	fmt.Println("es", es)
// 	return es, nil

// }
