package logharbour

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/some"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
)

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
}

const (
	INDEX       = "logharbour"
	APP         = "app"
	TYPE        = "type"
	WHO         = "who"
	CLASS       = "class"
	INSTANCE    = "instance"
	OP          = "op"
	REMOTE_IP   = "remote_ip"
	PRI         = "pri"
	WHEN        = "when"
	MODULE      = "module"
	STATUS      = "status"
	SYSTEM      = "system"
	LAYOUT      = "2006-01-02T15:04:05Z"
	DIALTIMEOUT = 10 * time.Second
	SERVICES    = "services"
	ACTIVITY    = "A"
	DEBUG       = "D"
)

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

	// Call getQuery fuction which will return a query for valid method parameters
	query, err = getQuery(setParam)
	if err != nil {
		return nil, fmt.Errorf("error while calling getQuery : %v ", err)

	}

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), DIALTIMEOUT)
	defer cancel()

	// To Serach data
	// This will return a set of unique values for an attribute based on method parameters
	res, err := client.Search().Index(INDEX).Request(&search.Request{
		Query: query,
		Size:  &zero,
		Aggregations: map[string]types.Aggregations{
			SERVICES: {
				Terms: &types.TermsAggregation{
					Field: some.String(setAttr),
				},
			},
		},
	}).Do(ctx)

	if err != nil {
		return nil, fmt.Errorf("error runnning search query: %s", err)
	}

	// To assert if response contains aggregation response and it must be type of *types.StringTermsAggregate
	servicesAgg, ok := res.Aggregations[SERVICES].(*types.StringTermsAggregate)
	if !ok || servicesAgg == nil {
		return nil, fmt.Errorf("services aggregation is not present or not of type *types.StringTermsAggregate")
	}
	// To extract bucket_key and bucket_count from Buckets which must be type of types.BucketsStringTermsBucket and append it to dataMap
	bucketsSlice, ok := servicesAgg.Buckets.([]types.StringTermsBucket)
	if ok {
		for _, bucket := range bucketsSlice {
			// Ensuring bucket key is a string.
			// It proceeds to extract the key as a string and assign it to bucketKeyString.
			if bucketKeyString, ok := bucket.Key.(string); ok {
				dataMap[bucketKeyString] = bucket.DocCount
			} else {
				return nil, fmt.Errorf("The bucket key is not a string")
			}
		}
	} else {
		return nil, fmt.Errorf("services aggregation Buckets field has Unknown type: %v , valid type is :%v", reflect.TypeOf(servicesAgg.Buckets), "[]types.StringTermsBucket")
	}

	return dataMap, nil
}

// To form a query based on valid method parameters
func getQuery(param GetSetParam) (*types.Query, error) {
	var (
		termQueries = make([]types.Query, 0)
		typeQueries = make([]types.Query, 0)
		query       *types.Query
		Priority    = []string{"Debug2", "Debug1", "Debug0", "Info", "Warn", "Err", "Crit", "Sec"}
	)

	// Case 1:If fromts and tots are specified, then tots must be after fromts.
	if param.Fromts != nil && param.Tots != nil {
		fromTs := param.Fromts.Format(LAYOUT)
		toTs := param.Tots.Format(LAYOUT)
		if param.Fromts.Before(*param.Tots) {
			days := types.Query{
				Range: map[string]types.RangeQuery{
					WHEN: types.DateRangeQuery{
						Gte: &fromTs,
						Lte: &toTs,
					},
				},
			}
			termQueries = append(termQueries, days)
		} else {
			return nil, fmt.Errorf("tots must be after fromts")
		}
		// Case 2:If only Fromts present
	} else if param.Fromts != nil && param.Tots == nil {
		fromTs := param.Fromts.Format(LAYOUT)
		days := types.Query{
			Range: map[string]types.RangeQuery{
				WHEN: types.DateRangeQuery{
					Gte: &fromTs,
				},
			},
		}
		termQueries = append(termQueries, days)
		// case 3: If only Tots present
	} else if param.Fromts == nil && param.Tots != nil {
		toTs := param.Tots.Format(LAYOUT)
		days := types.Query{
			Range: map[string]types.RangeQuery{
				WHEN: types.DateRangeQuery{
					Lte: &toTs,
				},
			},
		}
		termQueries = append(termQueries, days)
		// Case 4:If both fromts and tots are not specified only days are present
	} else if param.Ndays != nil && param.Fromts == nil && param.Tots == nil {
		if *param.Ndays > 0 {
			day := fmt.Sprintf("now-%dd/d", *param.Ndays)
			days := types.Query{
				Range: map[string]types.RangeQuery{
					WHEN: types.DateRangeQuery{
						Gte: &day,
					},
				},
			}
			termQueries = append(termQueries, days)
		}
	}

	if param.App != nil {
		termQueries = append(termQueries, types.Query{
			Term: map[string]types.TermQuery{
				APP: {Value: param.App},
			},
		})
	}

	if param.Type != nil {
		termQueries = append(termQueries, types.Query{
			Term: map[string]types.TermQuery{
				TYPE: {Value: param.Type.String()},
			},
		})
	}

	// typequeris contains only debug and activity logs
	typeQueries = append(typeQueries, types.Query{
		Term: map[string]types.TermQuery{
			TYPE: {Value: ACTIVITY},
		},
	})

	typeQueries = append(typeQueries, types.Query{
		Term: map[string]types.TermQuery{
			TYPE: {Value: DEBUG},
		},
	})

	// whenever type parameter is nil it considers all three types i.e Activity,Debug and Data change
	// If pri parameter is present then we cannot consider Data change type logs because it has no priority
	// so, In this case filter is applied for getting only Activity and Debug type i.e. tyqueries
	if param.Type == nil && param.Pri != nil {
		termQueries = append(termQueries, types.Query{
			Bool: &types.BoolQuery{
				Filter: typeQueries,
			},
		})

	}
	if param.Who != nil {
		termQueries = append(termQueries, types.Query{
			Term: map[string]types.TermQuery{
				WHO: {Value: *param.Who},
			},
		})
	}

	if param.Class != nil {
		termQueries = append(termQueries, types.Query{
			Term: map[string]types.TermQuery{
				CLASS: {Value: *param.Class},
			},
		})

		// Instance must be considered only if class is specified
		if param.Instance != nil && param.Class != nil {
			termQueries = append(termQueries, types.Query{
				Term: map[string]types.TermQuery{
					INSTANCE: {Value: *param.Instance},
				},
			})
		}
	}

	if param.Op != nil {
		termQueries = append(termQueries, types.Query{
			Term: map[string]types.TermQuery{
				OP: {Value: *param.Op},
			},
		})
	}

	if param.RemoteIP != nil {
		termQueries = append(termQueries, types.Query{
			Term: map[string]types.TermQuery{
				REMOTE_IP: {Value: *param.RemoteIP},
			},
		})
	}

	// pri specifies that only logs of priority equal to or higher than the value given here will be returned.
	if param.Pri != nil && param.Type != nil && LogType(*param.Type) != Change {
		pri := param.Pri.String()
		priFrom := slices.Index(Priority, pri)
		if priFrom >= 0 {
			priParams := Priority[priFrom:]
			if len(priParams) > 0 {
				var vals []types.FieldValue
				for _, val := range priParams {
					vals = append(vals, val)
				}
				termQueries = append(termQueries, types.Query{
					// add filter
					Terms: &types.TermsQuery{

						TermsQuery: map[string]types.TermsQueryField{
							PRI: vals,
						},
					},
				})
			}
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
		APP:       empty,
		TYPE:      empty,
		OP:        empty,
		INSTANCE:  empty,
		MODULE:    empty,
		PRI:       empty,
		STATUS:    empty,
		REMOTE_IP: empty,
		SYSTEM:    empty,
		WHO:       empty,
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
	setvalues, err := GetSet(querytoken, client, APP, GetSetParam{})

	if err != nil {
		return apps, fmt.Errorf("error at calling getset() : %w", err)
	}

	// Extracting keys from setValues
	for app := range setvalues {
		apps = append(apps, app)
	}
	return apps, nil
}
