package logharbour

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// ElasticsearchWriter defines methods for Elasticsearch writer
type ElasticsearchWriter interface {
	Write(index string, documentID string, body string) error
}

type ElasticsearchClient struct {
	client *elasticsearch.Client
}

// NewElasticsearchClient creates a new Elasticsearch client with the given configuration
func NewElasticsearchClient(cfg elasticsearch.Config) (*ElasticsearchClient, error) {
	esClient, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &ElasticsearchClient{client: esClient}, nil
}

// Write sends a document to Elasticsearch. It implements ElasticsearchWriter.
func (ec *ElasticsearchClient) Write(index string, documentID string, body string) error {
	req := esapi.IndexRequest{
		Index:      index,
		DocumentID: documentID,
		Body:       strings.NewReader(body),
	}

	res, err := req.Do(context.Background(), ec.client)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		log.Printf("Error response from Elasticsearch: %s", res.String())
		return errors.New(res.String())
	}

	return nil
}
