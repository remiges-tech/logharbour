package main

import (
	"fmt"
	"log"

	// "main/elasticSearchCtl"
	"os"

	"github.com/elastic/elastic-transport-go/v8/elastictransport"
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
		"id": {
		  "type": "keyword"
		},
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
		  "type": "object",
		  "properties": {
			"entity": {
			  "type": "keyword"
			},
			"op": {
			  "type": "keyword"
			},
			"activity_data": {
			  "type": "text"
			},
			"debug_data": {
			  "type": "object"
			},
			"change_data": {
			  "type": "object",
			  "properties": {
				"entity": {
				  "type": "keyword"
				},
				"op": {
				  "type": "keyword"
				},
				"changes": {
				  "type": "object",
				  "properties": {
					"field": {
					  "type": "keyword"
					},
					"new_value": {
					  "type": "text"
					},
					"old_value": {
					  "type": "text"
					}
				  }
				}
			  }
			}
		  }
		}
	  }
	}
  }
  `

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
				Logger:                 &elastictransport.TextLogger{Output: log.Writer(), EnableRequestBody: true},
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
	rootCmd.PersistentFlags().StringVarP(&address, "address", "a", "https://127.0.0.1:9200", "URL for Elasticsearch")
	rootCmd.PersistentFlags().StringVarP(&username, "username", "u", "elastic", "Username for Elasticsearch")
	// TUSHAR DB DETAILS
	// rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "Iu4K4=VsUZEBExLjDu4k", "Password for Elasticsearch")
	// rootCmd.PersistentFlags().StringVarP(&esCer, "es-cer", "c", "c0456a9e300eac38c9af6f416c54c55857e2fbc19a2deaa44bb8a582618bcd62", "certificateFingerprint")

	// // ANIKET DB DETAILS
	// rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "XSTwxC*giO71PGZm5urS", "Password for Elasticsearch")
	// rootCmd.PersistentFlags().StringVarP(&esCer, "es-cer", "c", "4b41377142441840d1099bcde5d294d25b8e7b39daf0a879343e5b552bc17f2c", "certificateFingerprint")

	// KANCHAN DB DETAILS
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "9jjWQryjHca-9flzDcKU", "Password for Elasticsearch")
	rootCmd.PersistentFlags().StringVarP(&esCer, "es-cer", "c", "3395adb7832f24e043ea7101b7f821bd97786fc808c335ba439a3681585119fc", "certificateFingerprint")

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
