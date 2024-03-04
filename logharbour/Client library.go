package logharbour

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/sortorder"
)

var layout = "2006-01-02T15:04:05"
var index = "logharbour"
var LOGHARBOUR_GETLOGS_MAXREC = 5
var res *search.Response
var err error

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
	Priority         *LogPriority
	SearchAfterTS    *string
	SearchAfterDocID *string
}

func GetLogs(querytoken string, client *elasticsearch.TypedClient, logParam GetLogsParam) ([]LogEntry, int, error) {

	var matchField []types.Query
	var logentry []LogEntry

	if logParam.FromTS != nil && logParam.ToTS != nil {
		fromTs := logParam.FromTS.Format(layout)
		toTs := logParam.ToTS.Format(layout)
		if logParam.FromTS.Before(*logParam.ToTS) {
			days := types.Query{
				Range: map[string]types.RangeQuery{
					"when": types.DateRangeQuery{
						Gte: &fromTs,
						Lte: &toTs,
					},
				},
			}
			matchField = append(matchField, days)
		} else {
			return nil, 0, fmt.Errorf("tots must be after fromts")
		}
	} else if logParam.FromTS != nil && logParam.ToTS == nil {
		fromTs := logParam.FromTS.Format(layout)
		days := types.Query{
			Range: map[string]types.RangeQuery{
				"when": types.DateRangeQuery{
					Gte: &fromTs,
				},
			},
		}
		matchField = append(matchField, days)

	} else if logParam.FromTS == nil && logParam.ToTS != nil {
		toTs := logParam.ToTS.Format(layout)
		days := types.Query{
			Range: map[string]types.RangeQuery{
				"when": types.DateRangeQuery{
					Lte: &toTs,
				},
			},
		}
		matchField = append(matchField, days)

	} else if logParam.NDays != nil && logParam.FromTS == nil && logParam.ToTS == nil {
		if *logParam.NDays > 0 {
			day := fmt.Sprintf("now-%dd/d", *logParam.NDays)
			days := types.Query{
				Range: map[string]types.RangeQuery{
					"when": types.DateRangeQuery{
						Gte: &day,
					},
				},
			}
			matchField = append(matchField, days)
		}
	}

	if logParam.App != nil {
		app := types.Query{
			Match: map[string]types.MatchQuery{
				"app": {Query: *logParam.App},
			},
		}
		matchField = append(matchField, app)
	}

	if logParam.Type != nil {
		types := types.Query{
			Match: map[string]types.MatchQuery{
				"type": {Query: logParam.Type.String()},
			},
		}
		matchField = append(matchField, types)
	}

	if logParam.Who != nil {
		who := types.Query{
			Match: map[string]types.MatchQuery{
				"who": {Query: *logParam.Who},
			},
		}
		matchField = append(matchField, who)
	}

	if logParam.Class != nil {
		class := types.Query{
			Match: map[string]types.MatchQuery{
				"class": {Query: *logParam.Class},
			},
		}
		matchField = append(matchField, class)
	}

	if logParam.Instance != nil {
		instance := types.Query{
			Match: map[string]types.MatchQuery{
				"instance": {Query: *logParam.Instance},
			},
		}
		matchField = append(matchField, instance)
	}

	if logParam.Operation != nil {
		op := types.Query{
			Match: map[string]types.MatchQuery{
				"op": {Query: *logParam.Operation},
			},
		}
		matchField = append(matchField, op)
	}

	if logParam.RemoteIP != nil {
		remoteIp := types.Query{
			Match: map[string]types.MatchQuery{
				"remote_ip": {Query: *logParam.RemoteIP},
			},
		}
		matchField = append(matchField, remoteIp)
	}

	if logParam.Priority != nil {
		pri := types.Query{
			Match: map[string]types.MatchQuery{
				"app": {Query: logParam.Priority.String()},
			},
		}
		matchField = append(matchField, pri)
	}

	query := &types.Query{
		Bool: &types.BoolQuery{
			Must: matchField,
		},
	}

	sortByWhen := types.SortOptions{
		SortOptions: map[string]types.FieldSort{
			"when": {Order: &sortorder.Desc},
		},
	}

	// sortById := types.SortOptions{
	// 	SortOptions: map[string]types.FieldSort{
	// 		"_id": {Order: &sortorder.Desc},
	// 	},
	// }

	if logParam.SearchAfterTS != nil && logParam.SearchAfterDocID != nil {
		res, err = client.Search().Index(index).Request(&search.Request{
			Size:        &LOGHARBOUR_GETLOGS_MAXREC,
			Query:       query,
			SearchAfter: []types.FieldValue{logParam.SearchAfterTS, logParam.SearchAfterDocID},
			Sort:        []types.SortCombinations{sortByWhen},
		}).Do(context.Background())
	} else if logParam.SearchAfterTS != nil && logParam.SearchAfterDocID == nil {
		res, err = client.Search().Index(index).Request(&search.Request{
			Size:        &LOGHARBOUR_GETLOGS_MAXREC,
			Query:       query,
			SearchAfter: []types.FieldValue{logParam.SearchAfterTS},
			Sort:        []types.SortCombinations{sortByWhen},
		}).Do(context.Background())
	} else if logParam.SearchAfterTS == nil && logParam.SearchAfterDocID != nil {
		res, err = client.Search().Index(index).Request(&search.Request{
			Size:        &LOGHARBOUR_GETLOGS_MAXREC,
			Query:       query,
			SearchAfter: []types.FieldValue{logParam.SearchAfterDocID},
			Sort:        []types.SortCombinations{sortByWhen},
		}).Do(context.Background())
	} else {
		res, err = client.Search().Index(index).Request(&search.Request{
			Size:  &LOGHARBOUR_GETLOGS_MAXREC,
			Query: query,
			Sort:  []types.SortCombinations{sortByWhen},
		}).Do(context.Background())
	}
	if err != nil {
		return nil, 0, fmt.Errorf("Error while searching document in es:%v", err)
	}
	var logg LogEntry

	if res != nil {
		for _, hit := range res.Hits.Hits {
			if err := json.Unmarshal([]byte(hit.Source_), &logg); err != nil {
				return nil, 0, fmt.Errorf("error while unmarshalling response:%v", err)
			}
			logentry = append(logentry, logg)
		}
	}
	return logentry, int(res.Hits.Total.Value), nil
}

// func match(field string, query *string) types.Query {
// 	var matchQuery types.Query
// 	if query != nil {
// 		matchQuery = types.Query{
// 			Match: map[string]types.MatchQuery{
// 				field: {Query: *query},
// 			},
// 		}
// 	}
// 	return matchQuery
// }
