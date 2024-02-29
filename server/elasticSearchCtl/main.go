package main

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

	// "github.com/remiges-tech/logharbour/logharbour"
	"github.com/spf13/cobra"
)

var (
	es       *elasticsearch.Client
	address  string
	username string
	password string
	esCer    string
)

type config struct {
	Username               string
	Password               string
	Addresses              []string
	CertificateFingerprint string
}

var createIndexBody = `{
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
		  "type": "text"
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
		  "type": "keyword"
		},
		"msg": {
		  "type": "keyword"
		},
		"data": {
		  "type": "nested"
		}
	  }
	}
  }
  `

func main() {
	var rootCmd = &cobra.Command{
		Use:   "elasticSearchCtl",
		Short: "elasticSearchCtl is a command-line interface facilitating seamless interaction with Elasticsearch, empowering users to efficiently query, manage, and manipulate data within Elasticsearch data stores.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Initialize Elasticsearch client here
			cfg := elasticsearch.Config{
				Addresses:              []string{address},
				Username:               username,
				Password:               password,
				CertificateFingerprint: esCer,
			}

			client, err := elasticsearch.NewClient(cfg)
			if err != nil {
				return err
			}

			es = client

			fmt.Println("Elasticsearch configured successfully.")
			return nil
		},
	}
	rootCmd.PersistentFlags().StringVarP(&address, "address", "a", "https://localhost:9200", "URL for Elasticsearch")
	rootCmd.PersistentFlags().StringVarP(&username, "username", "u", "elastic", "Username for Elasticsearch")
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "Iu4K4=VsUZEBExLjDu4k", "Password for Elasticsearch")
	rootCmd.PersistentFlags().StringVarP(&esCer, "es-cer", "c", "c0456a9e300eac38c9af6f416c54c55857e2fbc19a2deaa44bb8a582618bcd62", "certificateFingerprint")

	// rootCmd.PersistentFlags().StringVarP(&address, "address", "a", "https://10.10.10.220:9200", "URL for Elasticsearch")
	// rootCmd.PersistentFlags().StringVarP(&username, "username", "u", "elastic", "Username for Elasticsearch")
	// rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "Yga1QzMVaqzw-UnMBt=n", "Password for Elasticsearch")
	// rootCmd.PersistentFlags().StringVarP(&esCer, "es-cer", "c", "2a0808ed5628b523dec26435eb2761b4e27305a0e7c44f295eea7feabf208a22", "certificateFingerprint")

	var insertCmd = &cobra.Command{
		Use:   "add [logFile]",
		Short: "Import data from a file into the Elasticsearch datastore.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if es == nil {
				return fmt.Errorf("elasticsearch client is not configured")
			}
			logFile := args[0]
			indexName := args[1]
			log, err := ReadLogFromFile(logFile)
			if err != nil {
				return fmt.Errorf("error converting data from log file:%v", err)
			}
			if err := InsertLog(es, log, indexName); err != nil {
				return fmt.Errorf("error while inserting data: %v", err)
			}
			fmt.Println("Logs inserted successfully.")
			return nil
		},
	}

	var createIndex = &cobra.Command{
		Use:   "create [index Name]",
		Short: "Create index in elastic search.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if es == nil {
				return fmt.Errorf("elasticsearch client is not configured")
			}
			indexName := args[0]

			indexBody := strings.NewReader(createIndexBody)
			// Create the index request
			req := esapi.IndicesCreateRequest{
				Index: indexName,
				Body:  indexBody,
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
				// var successResponse map[string]interface{}
				// if err := json.NewDecoder(res.Body).Decode(&successResponse); err != nil {
				// 	log.Fatalf("Error parsing the success response body: %s", err)
				// }
				fmt.Println("Index created successfully:") //, successResponse["index"].(map[string]interface{})["created"])
			}
			return nil

		},
	}

	rootCmd.AddCommand(insertCmd)
	rootCmd.AddCommand(createIndex)

	if err := rootCmd.Execute(); err != nil {
		log.Println("Error:", err)
		os.Exit(1)
	}
}

func GetElasticsearch(filepath string) (*elasticsearch.Client, error) {

	bytedata, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	var ESConfig config
	err = json.Unmarshal(bytedata, &ESConfig)
	if err != nil {
		return nil, err
	}

	cfg := elasticsearch.Config{
		Addresses:              ESConfig.Addresses,
		Username:               ESConfig.Username,
		Password:               ESConfig.Password,
		CertificateFingerprint: ESConfig.CertificateFingerprint,
	}

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	fmt.Println("es", es)
	return es, nil

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
