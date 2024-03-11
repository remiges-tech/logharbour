package logharbour

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/sortorder"
)

const (
	when      = "when"
	app       = "app"
	typeConst = "type"
	who       = "who"
	class     = "class"
	instance  = "instance"
	op        = "op"
	remote_ip = "remote_ip"
	pri       = "pri"
	id        = "id" // document id
	layout    = "2006-01-02T15:04:05Z"
)

var (
	Index                     = "logharbour"
	LOGHARBOUR_GETLOGS_MAXREC = 5
	res                       *search.Response
	Priority                  = []string{"Debug2", "Debug1", "Debug0", "Info", "Warn", "Err", "Crit", "Sec"}
	requiredPri               []string
)

type GetLogsParam struct {
	App              *string
	Type             *LogType
	Module           *string
	Who              *string
	Class            *string
	Instance         *string
	Operation        *string
	FromTS           *time.Time
	ToTS             *time.Time
	NDays            *int
	RemoteIP         *string
	Priority         *LogPriority
	SearchAfterTS    *string
	SearchAfterDocID *string
}

type GetUnusualIPParam struct {
	App       *string
	Who       *string
	Class     *string
	Operation *string
	NDays     *int
}

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

// GetLogs retrieves an slice of logEntry from Elasticsearch based on the fields provided in logParam.
func GetLogs(querytoken string, client *elasticsearch.TypedClient, logParam GetLogsParam) ([]LogEntry, int, error) {

	var queries []types.Query
	var logEntries []LogEntry

	ok, ranges, err := rangeQueryForTimestamp(logParam.FromTS, logParam.ToTS, logParam.NDays)
	if ok {
		queries = append(queries, ranges)
	}
	if err != nil {
		return nil, 0, err
	}

	if ok, app := termQueryForField(app, logParam.App); ok {
		queries = append(queries, app)
	}

	if logParam.Type != nil {
		logTypeStr := logParam.Type.String()
		if ok, logType := termQueryForField(typeConst, &logTypeStr); ok {
			queries = append(queries, logType)
		}
	}

	if ok, who := termQueryForField(WHO, logParam.Who); ok {

		queries = append(queries, who)
	}
	if ok, class := termQueryForField(class, logParam.Class); ok {

		queries = append(queries, class)
	}
	if ok, instance := termQueryForField(instance, logParam.Instance); ok {

		queries = append(queries, instance)
	}
	if ok, op := termQueryForField(op, logParam.Operation); ok {

		queries = append(queries, op)
	}
	if ok, remoteIp := termQueryForField(remote_ip, logParam.RemoteIP); ok {

		queries = append(queries, remoteIp)
	}

	if logParam.Priority != nil {
		priStr := logParam.Priority.String()
		priFrom := slices.Index(Priority, priStr)
		if priFrom > 0 {
			requiredPri = Priority[priFrom:]
		}
		if ok, pri := termQueryForField(pri, nil, requiredPri...); ok {
			queries = append(queries, pri)
		}
	}

	// creating elastic search bool query
	query := &types.Query{
		Bool: &types.BoolQuery{
			Filter: queries,
		},
	}

	if len(queries) == 0 {
		return nil, 0, fmt.Errorf("No Filter param")
	}

	// sorting record on base of when
	sortByWhen := types.SortOptions{
		SortOptions: map[string]types.FieldSort{
			when: {Order: &sortorder.Desc},
		},
	}

	// sortById := types.SortOptions{
	// 	SortOptions: map[string]types.FieldSort{
	// 		id : {Order: &sortorder.Desc},
	// 	},
	// }

	//  calling search query when SearchAfterTS and SearchAfterDocID given
	if logParam.SearchAfterTS != nil && logParam.SearchAfterDocID != nil {
		res, err = client.Search().Index(Index).Request(&search.Request{
			Size:        &LOGHARBOUR_GETLOGS_MAXREC,
			Query:       query,
			SearchAfter: []types.FieldValue{logParam.SearchAfterTS, logParam.SearchAfterDocID},
			Sort:        []types.SortCombinations{sortByWhen},
		}).Do(context.Background())
		//  calling search query when SearchAfterTS is given
	} else if logParam.SearchAfterTS != nil && logParam.SearchAfterDocID == nil {
		res, err = client.Search().Index(Index).Request(&search.Request{
			Size:        &LOGHARBOUR_GETLOGS_MAXREC,
			Query:       query,
			SearchAfter: []types.FieldValue{logParam.SearchAfterTS},
			Sort:        []types.SortCombinations{sortByWhen},
		}).Do(context.Background())
		//  calling search query when is SearchAfterDocID
	} else if logParam.SearchAfterTS == nil && logParam.SearchAfterDocID != nil {
		res, err = client.Search().Index(Index).Request(&search.Request{
			Size:        &LOGHARBOUR_GETLOGS_MAXREC,
			Query:       query,
			SearchAfter: []types.FieldValue{logParam.SearchAfterDocID},
			Sort:        []types.SortCombinations{sortByWhen},
		}).Do(context.Background())
	} else {
		//  calling search query when SearchAfterTS and SearchAfterDocID not given
		res, err = client.Search().Index(Index).Request(&search.Request{
			Size:  &LOGHARBOUR_GETLOGS_MAXREC,
			Query: query,
			Sort:  []types.SortCombinations{sortByWhen},
		}).Do(context.Background())
	}
	if err != nil {
		return nil, 0, fmt.Errorf("Error while searching document in es:%v", err)
	}
	var logEnter LogEntry

	// Unmarshalling hit.source into LogEntry
	if res != nil {
		for _, hit := range res.Hits.Hits {
			if err := json.Unmarshal([]byte(hit.Source_), &logEnter); err != nil {
				return nil, 0, fmt.Errorf("error while unmarshalling response:%v", err)
			}
			logEntries = append(logEntries, logEnter)
		}
	}
	return logEntries, int(res.Hits.Total.Value), nil
}

// termQueryForField constructs and returns a term query for a specified field and its corresponding value.
func termQueryForField(field string, value *string, values ...string) (bool, types.Query) {
	if value != nil {
		query := types.Query{
			Term: map[string]types.TermQuery{
				field: {Value: value},
			},
		}
		return true, query
	}
	if values != nil {
		var vals []types.FieldValue
		for _, val := range values {
			vals = append(vals, val)
		}
		query := types.Query{
			Terms: &types.TermsQuery{
				TermsQuery: map[string]types.TermsQueryField{
					field: vals,
				},
			},
		}
		return true, query
	}
	return false, types.Query{}
}

// rangeQueryForTimestamp generates a range query for Elasticsearch based on the provided timestamps and number of days.
func rangeQueryForTimestamp(fromTS, toTS *time.Time, nDays *int) (bool, types.Query, error) {

	// return query if both present fromTs and toTs
	if fromTS != nil && toTS != nil {
		fromTs := fromTS.Format(layout)
		toTs := toTS.Format(layout)
		if fromTS.Before(*toTS) {
			query := types.Query{
				Range: map[string]types.RangeQuery{
					when: types.DateRangeQuery{
						Gte: &fromTs,
						Lte: &toTs,
					},
				},
			}
			return true, query, nil
		} else {
			return false, types.Query{}, fmt.Errorf("tots must be after fromts")
		}

		// appending query if FromTs is present
	} else if fromTS != nil && toTS == nil {
		fromTs := fromTS.Format(layout)
		query := types.Query{
			Range: map[string]types.RangeQuery{
				when: types.DateRangeQuery{
					Gte: &fromTs,
				},
			},
		}
		return true, query, nil

		// appending query if ToTS is present
	} else if fromTS == nil && toTS != nil {
		toTs := toTS.Format(layout)
		query := types.Query{
			Range: map[string]types.RangeQuery{
				when: types.DateRangeQuery{
					Lte: &toTs,
				},
			},
		}
		return true, query, nil

		// appending query for get log for n number of day
	} else if nDays != nil && fromTS == nil && toTS == nil {
		if *nDays > 0 {
			day := fmt.Sprintf("now-%dd/d", *nDays) // now-5d
			query := types.Query{
				Range: map[string]types.RangeQuery{
					when: types.DateRangeQuery{
						Gte: &day,
					},
				},
			}
			return true, query, nil
		}
	}
	return false, types.Query{}, nil
}

// GetUnusualIP will go through the logs of the last ndays days which match the search criteria, and pull out all the
// remote IP addresses which account for a low enough percentage of the total to be treated as unusual or suspicious.
func GetUnusualIP(queryToken string, client *elasticsearch.TypedClient, logParam GetUnusualIPParam, unusualPercent float64) ([]string, error) {
	unusualIPs := []string{}

	if unusualPercent < 0.5 || unusualPercent > 50 {
		return nil, fmt.Errorf("unusualPercent is not between 0.5 to 50")
	}

	aggregatedIPs, err := GetSet(queryToken, client, remote_ip, GetSetParam{
		App:   logParam.App,
		Who:   logParam.Who,
		Class: logParam.Class,
		Op:    logParam.Operation,
		Ndays: logParam.NDays,
	})
	if err != nil {
		return nil, err
	}
	//   Calculate total number of logs to find what 1% represents
	var count int64 = 0
	for _, v := range aggregatedIPs {
		count += v
	}
	percentThreshold := float64(count) * unusualPercent / 100

	if percentThreshold > 1 {
		for ip, count := range aggregatedIPs {
			fmt.Printf("IP: %s, Count: %d\n", ip, count)
			if count <= int64(percentThreshold) {
				if ip != "local" {
					unusualIPs = append(unusualIPs, ip)
				}
			}
		}
	}
	return unusualIPs, nil

}
