package logharbour

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/some"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/sortorder"
)

const (
	when        = "when"
	app         = "app"
	module      = "module"
	typeConst   = "type"
	who         = "who"
	status      = "status"
	system      = "system"
	class       = "class"
	instance    = "instance"
	op          = "op"
	remote_ip   = "remote_ip"
	pri         = "pri"
	id          = "id" // document id
	layout      = "2006-01-02T15:04:05Z"
	logSet      = "logset"
	DIALTIMEOUT = 500 * time.Second
	ACTIVITY    = "A"
	DEBUG       = "D"
	field       = "data.changes.field"
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
	Field            *string
}

type GetUnusualIPParam struct {
	App       *string
	Who       *string
	Class     *string
	Operation *string
	NDays     *int
}
type GetSetParam struct {
	App      *string      `json:"app" validate:"omitempty,alpha,lt=30"`
	Type     *LogType     `json:"type" validate:"omitempty,oneof=1 2 3 4"`
	Who      *string      `json:"who" validate:"omitempty,alpha,lt=20"`
	Class    *string      `json:"class" validate:"omitempty,alpha,lt=30"`
	Instance *string      `json:"instance" validate:"omitempty,alpha,lt=30"`
	Op       *string      `json:"op" validate:"omitempty,alpha,lt=25"`
	Fromts   *time.Time   `json:"fromts" validate:"omitempty"`
	Tots     *time.Time   `json:"tots" validate:"omitempty"`
	Ndays    *int         `json:"ndays" validate:"omitempty,number,lt=100"`
	RemoteIP *string      `json:"remoteIP" validate:"omitempty"`
	Pri      *LogPriority `json:"pri" validate:"omitempty,oneof=1 2 3 4 5 6 7 8"`
	setAttr  string       `json:"setAttr"`
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

	if ok, who := termQueryForField(who, logParam.Who); ok {

		queries = append(queries, who)
	}

	if ok, module := termQueryForField(module, logParam.Module); ok {

		queries = append(queries, module)
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
			// "_doc": {Order: &sortorder.Desc},
		},
	}

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

// GetUnusualIP will go through the logs of the last ndays days which match the search criteria, and pull out all the
// remote IP addresses which account for a low enough percentage of the total to be treated as unusual or suspicious.
func GetUnusualIP(queryToken string, client *elasticsearch.TypedClient, unusualPercent float64, logParam GetUnusualIPParam) ([]string, error) {
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

	localIp, err := GetLocalIPAddress()
	if err != nil {
		println("Error:", err)
		return nil, err
	}
	println("Local IP address:", localIp)

	if percentThreshold > 1 {
		for ip, count := range aggregatedIPs {
			fmt.Printf("IP: %s, Count: %d\n", ip, count)
			if count <= int64(percentThreshold) {
				if ip != localIp {
					unusualIPs = append(unusualIPs, ip)
				}
			}
		}
	}
	return unusualIPs, nil

}

// GetLocalIPAddress returns the local IPv4 address of the system.
func GetLocalIPAddress() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	return "", nil
}

// GetSet gets a set of values for an attribute from the log entries specified.
// This is a faceted search for one attribute.
func GetSet(queryToken string, client *elasticsearch.TypedClient, setAttr string, setParam GetSetParam) (map[string]int64, error) {

	var (
		query   *types.Query
		zero    = 0
		dataMap = make(map[string]int64)
	)

	// Validate setAttr
	_, err := isValidSetAttribute(setAttr)
	if err != nil {
		return nil, err
	}

	// Call getQuery function which will return a query for valid method parameters
	query, err = getQuery(setParam)
	if err != nil {
		return nil, fmt.Errorf("error while calling getQuery : %v ", err)

	}

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), DIALTIMEOUT)
	defer cancel()

	// To Serach data
	// This will return a set of unique values for an attribute based on method parameters
	res, err := client.Search().Index(Index).Request(&search.Request{
		Query: query,
		Size:  &zero,
		Aggregations: map[string]types.Aggregations{
			logSet: {
				Terms: &types.TermsAggregation{
					Field: some.String(setAttr),
					Order: map[string]sortorder.SortOrder{"_count": sortorder.SortOrder{"asc"}},
				},
			},
		},
	}).Do(ctx)

	if err != nil {
		return nil, fmt.Errorf("error runnning search query: %s", err)
	}

	// To assert if response contains aggregation response and it must be type of *types.StringTermsAggregate
	logSetAgg, ok := res.Aggregations[logSet].(*types.StringTermsAggregate)
	if !ok || logSetAgg == nil {
		return nil, fmt.Errorf("services aggregation is not present or not of type *types.StringTermsAggregate")
	}
	// To extract bucket_key and bucket_count from Buckets which must be type of types.BucketsStringTermsBucket and append it to dataMap
	bucketsSlice, ok := logSetAgg.Buckets.([]types.StringTermsBucket)
	if ok {
		for _, bucket := range bucketsSlice {
			// Ensuring bucket key is a string.
			// It proceeds to extract the key as a string and assign it to bucketKeyString.
			if bucketKeyString, ok := bucket.Key.(string); ok {
				dataMap[bucketKeyString] = bucket.DocCount
			} else {
				return nil, fmt.Errorf("the bucket key is not a string")
			}
		}
	} else {
		return nil, fmt.Errorf("services aggregation Buckets field has Unknown type: %v , valid type is :%v", reflect.TypeOf(logSetAgg.Buckets), "[]types.StringTermsBucket")
	}

	return dataMap, nil
}

// To form a query based on valid method parameters
func getQuery(param GetSetParam) (*types.Query, error) {

	var (
		termQueries = make([]types.Query, 0)
		typeQueries = make([]types.Query, 0)
		activity    = "A"
		debug       = "D"
		query       *types.Query
		Priority    = []string{"Debug2", "Debug1", "Debug0", "Info", "Warn", "Err", "Crit", "Sec"}
	)

	ok, ranges, err := rangeQueryForTimestamp(param.Fromts, param.Tots, param.Ndays)
	if ok {
		termQueries = append(termQueries, ranges)
	}
	if err != nil {
		return nil, err
	}
	if ok, app := termQueryForField(app, param.App); ok {
		termQueries = append(termQueries, app)
	}

	// typequeris contains only debug and activity logs
	if ok, typeConst := termQueryForField(typeConst, &activity); ok {
		typeQueries = append(typeQueries, typeConst)
	}

	if ok, typeConst := termQueryForField(typeConst, &debug); ok {
		typeQueries = append(typeQueries, typeConst)
	}

	// whenever type parameter is nil it considers all three types i.e Activity,Debug and Data change
	// If pri parameter is present then we cannot consider Data change type logs because it has no priority
	// so, In this case filter is applied for getting only Activity and Debug type i.e. tyqueries
	if param.Type == nil {
		if param.Pri != nil {
			termQueries = append(termQueries, types.Query{
				Bool: &types.BoolQuery{
					Filter: typeQueries,
				},
			})
		}
	} else {
		if *param.Type == Change {
			termQueries = append(termQueries, types.Query{
				Bool: &types.BoolQuery{
					Filter: typeQueries,
				},
			})
		} else {
			logTypeStr := param.Type.String()
			if ok, logType := termQueryForField(typeConst, &logTypeStr); ok {
				termQueries = append(termQueries, logType)
			}

		}

	}

	if ok, who := termQueryForField(who, param.Who); ok {

		termQueries = append(termQueries, who)
	}
	if ok, class := termQueryForField(class, param.Class); ok {

		termQueries = append(termQueries, class)
	}
	// Instance must be considered only if the class is specified
	if param.Class != nil {
		if ok, instance := termQueryForField(instance, param.Instance); ok {

			termQueries = append(termQueries, instance)
		}
	}
	if ok, op := termQueryForField(op, param.Op); ok {

		termQueries = append(termQueries, op)
	}
	if ok, remote_ip := termQueryForField(remote_ip, param.RemoteIP); ok {

		termQueries = append(termQueries, remote_ip)
	}

	// pri specifies that only logs of priority equal to or higher than the value given here will be returned.
	if param.Pri != nil {
		priStr := param.Pri.String()
		priFrom := slices.Index(Priority, priStr)
		if priFrom > 0 {
			requiredPri = Priority[priFrom:]
		}
		if ok, pri := termQueryForField(pri, nil, requiredPri...); ok {
			termQueries = append(termQueries, pri)
		}
	}

	query = &types.Query{
		Bool: &types.BoolQuery{
			Filter: termQueries,
		},
	}
	return query, nil
}

func isValidSetAttribute(setAttr string) (bool, error) {

	var (
		empty   = struct{}{}
		pattern = "^[a-z]{1,9}$"
	)
	// Validate setAttr
	regex := regexp.MustCompile(pattern)
	if setAttr == "" && !regex.MatchString(setAttr) {
		return false, fmt.Errorf("attribute %s must not contain numbers or special characters, and must not be empty, with length not exceeding 9", setAttr)
	}

	// The attribute named can only be one of those which have finite discrete values, i.e. they are
	// conceptually enumerated types.
	allowedAttributes := map[string]struct{}{
		app:       empty,
		typeConst: empty,
		op:        empty,
		instance:  empty,
		class:     empty,
		module:    empty,
		pri:       empty,
		status:    empty,
		remote_ip: empty,
		system:    empty,
		who:       empty,
		field:     empty,
	}

	// To validate  setAttr only one of allowedAttributes has been named, and if not, will return an error.
	if _, ok := allowedAttributes[setAttr]; !ok {
		return false, fmt.Errorf("attribute '%s' is not allowed for set retrieval", setAttr)
	}
	return true, nil
}

// GetApps is used  to retrieve the list of apps
func GetApps(querytoken string, client *elasticsearch.TypedClient) (apps []string, err error) {

	// Calling GetSet() for getting  all unique values for apps
	setvalues, err := GetSet(querytoken, client, app, GetSetParam{})

	if err != nil {
		return apps, fmt.Errorf("error at calling GetSet() : %w", err)
	}

	// Extracting keys from setValues
	for app := range setvalues {
		apps = append(apps, app)
	}
	return apps, nil
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

// GetChanges retrieves an slice of logEntry from Elasticsearch based on the fields provided in logParam.
func GetChanges(querytoken string, client *elasticsearch.TypedClient, logParam GetLogsParam) ([]LogEntry, int, error) {

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

	// if logParam.Type != nil {
	logTypeStr := LogTypeChange
	if ok, logType := termQueryForField(typeConst, &logTypeStr); ok {
		queries = append(queries, logType)
	}
	// }

	if logParam.Field != nil {

		if ok, field := termQueryForField("data.changes.field", logParam.Field); ok {
			queries = append(queries, field)
		}
	}

	if ok, who := termQueryForField(who, logParam.Who); ok {

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
			// "_doc": {Order: &sortorder.Desc},
		},
	}

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
