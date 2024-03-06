package logharbour

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
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
	index                     = "logharbour"
	LOGHARBOUR_GETLOGS_MAXREC = 15
	res                       *search.Response
	err                       error
	Priority                  = []string{"Debug2", "Debug1", "Debug0", "Info", "Warn", "Err", "Crit", "Sec"}
	requiredPri               []string
)

type GetLogsParam struct {
	App              *string
	Type             *LogType
	Who              *string
	Class            *string
	Instance         *string
	Operation        *string
	FromTS           *time.Time
	ToTS             *time.Time
	NDays            *int
	RemoteIP         *string
	Priority         *string
	SearchAfterTS    *string
	SearchAfterDocID *string
}

func GetLogs(querytoken string, client *elasticsearch.TypedClient, logParam GetLogsParam) ([]LogEntry, int, error) {

	var queries []types.Query
	var logEntries []LogEntry

	// appending query if both present fromTs and toTs
	if logParam.FromTS != nil && logParam.ToTS != nil {
		fromTs := logParam.FromTS.Format(layout)
		toTs := logParam.ToTS.Format(layout)
		if logParam.FromTS.Before(*logParam.ToTS) {
			days := types.Query{
				Range: map[string]types.RangeQuery{
					when: types.DateRangeQuery{
						Gte: &fromTs,
						Lte: &toTs,
					},
				},
			}
			queries = append(queries, days)
		} else {
			return nil, 0, fmt.Errorf("tots must be after fromts")
		}

		// appending query if FromTs is present
	} else if logParam.FromTS != nil && logParam.ToTS == nil {
		fromTs := logParam.FromTS.Format(layout)
		days := types.Query{
			Range: map[string]types.RangeQuery{
				when: types.DateRangeQuery{
					Gte: &fromTs,
				},
			},
		}
		queries = append(queries, days)

		// appending query if ToTS is present
	} else if logParam.FromTS == nil && logParam.ToTS != nil {
		toTs := logParam.ToTS.Format(layout)
		days := types.Query{
			Range: map[string]types.RangeQuery{
				when: types.DateRangeQuery{
					Lte: &toTs,
				},
			},
		}
		queries = append(queries, days)

		// appending query for get log for n number of day
	} else if logParam.NDays != nil && logParam.FromTS == nil && logParam.ToTS == nil {
		if *logParam.NDays > 0 {
			day := fmt.Sprintf("now-%dd/d", *logParam.NDays) // now-5d
			days := types.Query{
				Range: map[string]types.RangeQuery{
					when: types.DateRangeQuery{
						Gte: &day,
					},
				},
			}
			queries = append(queries, days)
		}
	}

	if ok, app := termQueryForField(app, logParam.App); ok {
		queries = append(queries, app)
	}
	logTypeStr := logParam.Type.String()
	if ok, logType := termQueryForField(typeConst, &logTypeStr); ok {

		queries = append(queries, logType)
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
		priFrom := slices.Index(Priority, *logParam.Priority)
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
		res, err = client.Search().Index(index).Request(&search.Request{
			Size:        &LOGHARBOUR_GETLOGS_MAXREC,
			Query:       query,
			SearchAfter: []types.FieldValue{logParam.SearchAfterTS, logParam.SearchAfterDocID},
			Sort:        []types.SortCombinations{sortByWhen},
		}).Do(context.Background())
		//  calling search query when SearchAfterTS is given
	} else if logParam.SearchAfterTS != nil && logParam.SearchAfterDocID == nil {
		res, err = client.Search().Index(index).Request(&search.Request{
			Size:        &LOGHARBOUR_GETLOGS_MAXREC,
			Query:       query,
			SearchAfter: []types.FieldValue{logParam.SearchAfterTS},
			Sort:        []types.SortCombinations{sortByWhen},
		}).Do(context.Background())
		//  calling search query when is SearchAfterDocID
	} else if logParam.SearchAfterTS == nil && logParam.SearchAfterDocID != nil {
		res, err = client.Search().Index(index).Request(&search.Request{
			Size:        &LOGHARBOUR_GETLOGS_MAXREC,
			Query:       query,
			SearchAfter: []types.FieldValue{logParam.SearchAfterDocID},
			Sort:        []types.SortCombinations{sortByWhen},
		}).Do(context.Background())
	} else {
		//  calling search query when SearchAfterTS and SearchAfterDocID not given
		res, err = client.Search().Index(index).Request(&search.Request{
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
		matchQuery := types.Query{
			Term: map[string]types.TermQuery{
				field: {Value: value},
			},
		}
		return true, matchQuery
	}
	if values != nil {
		var vals []types.FieldValue
		for _, val := range values {
			vals = append(vals, val)
		}
		matchQuery := types.Query{
			Terms: &types.TermsQuery{
				TermsQuery: map[string]types.TermsQueryField{
					field: vals,
				},
			},
		}
		return true, matchQuery
	}
	return false, types.Query{}
}
