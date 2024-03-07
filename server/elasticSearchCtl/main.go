package main

import (
	"fmt"
	"log"

	"os"

	"github.com/elastic/go-elasticsearch/v8"
	elasticsearchctl "github.com/remiges-tech/logharbour/server/elasticSearchCtl/elasticSearch"

	"github.com/spf13/cobra"
)

var (
	es       *elasticsearch.Client
	address  string
	username string
	password string
	esCer    string
)

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

//   {
// 	"type": "object",
// 	"properties": {
// 	  "changes": {
// 		"properties": {
// 		  "field": {
// 			"type": "keyword"
// 		  },
// 		  "new_value": {
// 			"type": "text"
// 		  },
// 		  "old_value": {
// 			"type": "text"
// 		  }
// 		}
// 	  }
// 	}
//   }

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
			logEntries, err := elasticsearchctl.ReadLogFromFile(logFile)
			if err != nil {
				return fmt.Errorf("error converting data from log file:%v", err)
			}
			if err := elasticsearchctl.InsertLog(es, logEntries, indexName); err != nil {
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

			if err := elasticsearchctl.CreateElasticIndex(es, indexName, createIndexBody); err != nil {
				return fmt.Errorf("error while creating index: %v", err)
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
