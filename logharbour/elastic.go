package logharbour

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/elastic/go-elasticsearch/v8/esutil"
)

// ElasticsearchWriter defines methods for Elasticsearch writer
type ElasticsearchWriter interface {
	Write(index string, documentID string, body string) error
	BulkWrite(index string, documents []BulkDocument) (*BulkWriteResult, error)
}

// BulkDocument represents a document to be indexed in bulk
type BulkDocument struct {
	ID   string
	Body string
}

// BulkWriteResult contains the results of a bulk write operation
type BulkWriteResult struct {
	Successful int         // Number of documents successfully indexed
	Failed     int         // Number of documents that failed to index
	Errors     []BulkError // Detailed error information for each failed document
}

// BulkError represents an error for a specific document in bulk operation
type BulkError struct {
	DocumentID string // The document ID that failed
	Error      string // Error message (e.g., "mapper_parsing_exception: failed to parse field [priority_id]")
	Status     int    // HTTP status code (e.g., 400 for bad request, 409 for version conflict)
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

func (ec *ElasticsearchClient) CreateIndex(indexName, mapping string) error {
	res, err := ec.client.Indices.Create(indexName, ec.client.Indices.Create.WithBody(strings.NewReader(mapping)))
	if err != nil {
		return err
	}
	if res.IsError() {
		return fmt.Errorf("error creating index: %s", res.String())
	}
	return nil
}

func (ec *ElasticsearchClient) IndexExists(indexName string) (bool, error) {
	res, err := ec.client.Indices.Exists([]string{indexName})
	if err != nil {
		return false, err
	}
	return res.StatusCode == 200, nil
}

// BulkWrite performs bulk indexing of multiple documents to Elasticsearch
// 
// Error Handling Strategy:
// 1. Empty documents - Returns success with zero counts
// 2. Network/connection errors - Returned for retry by caller
// 3. Individual document failures - Tracked in BulkWriteResult.Errors
// 4. Partial failures - Success count > 0, Failed count > 0, errors detailed
// 5. Complete failure - All documents fail, suitable for retry
//
// The caller should check both the error return and the BulkWriteResult.Failed count
// to determine if retry is needed or if partial success is acceptable.
func (ec *ElasticsearchClient) BulkWrite(index string, documents []BulkDocument) (*BulkWriteResult, error) {
	if len(documents) == 0 {
		return &BulkWriteResult{}, nil
	}

	// Create bulk indexer
	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client:     ec.client,
		Index:      index,
		NumWorkers: 1, // Single worker to maintain order
	})
	if err != nil {
		return nil, fmt.Errorf("error creating bulk indexer: %w", err)
	}

	result := &BulkWriteResult{
		Errors: make([]BulkError, 0),
	}

	// Add documents to bulk indexer
	for _, doc := range documents {
		err := bi.Add(
			context.Background(),
			esutil.BulkIndexerItem{
				Action:     "index",
				DocumentID: doc.ID,
				Body:       strings.NewReader(doc.Body),
				OnSuccess: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem) {
					result.Successful++
				},
				OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
					result.Failed++
					// Capture detailed error information for each failed document
					bulkErr := BulkError{
						DocumentID: item.DocumentID,
						Status:     res.Status,
					}
					if err != nil {
						bulkErr.Error = err.Error()
					} else if res.Error.Type != "" {
						// Elasticsearch returned an error (e.g., mapping conflict, validation error)
						bulkErr.Error = fmt.Sprintf("%s: %s", res.Error.Type, res.Error.Reason)
					}
					result.Errors = append(result.Errors, bulkErr)
					log.Printf("Failed to index document %s: %s", item.DocumentID, bulkErr.Error)
				},
			},
		)
		if err != nil {
			return nil, fmt.Errorf("error adding document %s to bulk indexer: %w", doc.ID, err)
		}
	}

	// Close the bulk indexer and wait for all items to be processed
	if err := bi.Close(context.Background()); err != nil {
		return nil, fmt.Errorf("error closing bulk indexer: %w", err)
	}

	// Get stats
	stats := bi.Stats()
	log.Printf("Bulk indexing completed: indexed=%d, failed=%d", stats.NumIndexed, stats.NumFailed)

	return result, nil
}

// Write sends a document to Elasticsearch with retry logic.
// func (ec *ElasticsearchClient) Write(index string, documentID string, body string) error {
// 	var maxAttempts = 5
// 	var initialBackoff = 1 * time.Second

// 	operation := func() error {
// 		req := esapi.IndexRequest{
// 			Index:      index,
// 			DocumentID: documentID,
// 			Body:       strings.NewReader(body),
// 		}

// 		res, err := req.Do(context.Background(), ec.client)
// 		if err != nil {
// 			return err
// 		}
// 		defer res.Body.Close()

// 		if res.IsError() {
// 			log.Printf("Error response from Elasticsearch: %s", res.String())
// 			return errors.New(res.String())
// 		}

// 		return nil
// 	}

// 	for attempt := 1; attempt <= maxAttempts; attempt++ {
// 		err := operation()
// 		if err == nil {
// 			return nil // Success
// 		}

// 		if attempt == maxAttempts {
// 			return fmt.Errorf("after %d attempts, last error: %s", attempt, err)
// 		}

// 		wait := initialBackoff * time.Duration(1<<(attempt-1)) // Exponential backoff
// 		log.Printf("Attempt %d failed, retrying in %v: %v", attempt, wait, err)
// 		time.Sleep(wait)
// 	}

// 	return fmt.Errorf("reached max attempts without success")
// }

// InsecureTransport returns an HTTP transport that skips TLS certificate verification.
func InsecureTransport() *http.Transport {
	return &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
}
