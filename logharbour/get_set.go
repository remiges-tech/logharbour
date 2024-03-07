package logharbour

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"slices"
	"strconv"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/some"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/remiges-tech/alya/config"
	appconf "github.com/remiges-tech/logharbour/server/types"
)

type GetSetParam struct {
	QueryToken string       `json:"querytoken" validate:"required"` // client token which authorises the call to the function
	App        *string      `json:"app" validate:"omitempty,alpha,lt=30"`
	Type       *LogType     `json:"type" validate:"omitempty,oneof=1 2 3 4"`
	Who        *string      `json:"who" validate:"omitempty,alpha,lt=20"`
	Class      *string      `json:"class" validate:"omitempty,alpha,lt=30"`
	Instance   *string      `json:"instance" validate:"omitempty,alpha,lt=30"`
	Op         *string      `json:"op" validate:"omitempty,alpha,lt=25"`
	Fromts     *time.Time   `json:"fromts" validate:"omitempty"`
	Tots       *time.Time   `json:"tots" validate:"omitempty"`
	Ndays      *int         `json:"ndays" validate:"omitempty,number,lt=100"`
	RemoteIP   *string      `json:"remoteIP" validate:"omitempty"`
	Pri        *LogPriority `json:"pri" validate:"omitempty,oneof=1 2 3 4 5 6 7 8"`
	SetAttr    string       `json:"setattr" validate:"required,alpha,lte=9"`
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
)

// GetSet gets a set of values for an attribute from the log entries specified.
// This is a faceted search for one attribute.
func GetSet(req GetSetParam) (setvalues map[string]int64, err error) {

	var (
		appConfig *appconf.AppConfig
		query     *types.Query
		dataMap   = make(map[string]int64)
		empty     = struct{}{}
		zero      = 0
	)

	// Elasticsearch client setup
	err = config.LoadConfigFromFile("./config_dev_kanchan.json", &appConfig)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	// Database connection
	url := appConfig.DBHost + ":" + strconv.Itoa(appConfig.DBPort)

	dbConfig := elasticsearch.Config{
		Addresses:              []string{url},
		Username:               appConfig.DBUser,
		Password:               appConfig.DBPassword,
		CertificateFingerprint: appConfig.CertificateFingerprint,
	}

	// Create elasticSearch client with configvalues
	client, err := elasticsearch.NewTypedClient(dbConfig)
	if err != nil {
		log.Fatalf("Failed to create db connection: %v", err)
		return
	}

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), DIALTIMEOUT)
	defer cancel()

	// Validate setattr
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

	// To validate only one of allowedAttributes has been named, and if not, will return an error.
	if _, ok := allowedAttributes[req.SetAttr]; !ok {
		return nil, fmt.Errorf("attribute '%s' is not allowed for set retrieval", req.SetAttr)
	}

	// Call getQuery fuction which will return a query for valid method parameters
	query, err = getQuery(req)
	if err != nil {
		return nil, fmt.Errorf("error while calling getQuery : %v ", err)

	}

	// To Serach data
	// This will return a set of unique values for an attribute based on method parameters
	res, err := client.Search().Index(INDEX).Request(&search.Request{
		Query: query,
		Size:  &zero,
		Aggregations: map[string]types.Aggregations{
			SERVICES: {
				Terms: &types.TermsAggregation{
					Field: some.String(req.SetAttr),
				},
			},
		},
	}).Do(ctx)

	if err != nil {
		return nil, fmt.Errorf("error runnning search query: %s", err)
	}

	// To assert if response contains aggregation response and it must be type of *types.StringTermsAggregate
	servicesAgg, ok := res.Aggregations["services"].(*types.StringTermsAggregate)
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
func getQuery(req GetSetParam) (*types.Query, error) {
	var (
		termQueries = make([]types.Query, 0)
		query       *types.Query
		Priority    = []string{"Debug2", "Debug1", "Debug0", "Info", "Warn", "Err", "Crit", "Sec"}
	)

	// Case 1:If fromts and tots are specified, then tots must be after fromts.
	if req.Fromts != nil && req.Tots != nil {
		fromTs := req.Fromts.Format(LAYOUT)
		toTs := req.Tots.Format(LAYOUT)
		if req.Fromts.Before(*req.Tots) {
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
	} else if req.Fromts != nil && req.Tots == nil {
		fromTs := req.Fromts.Format(LAYOUT)
		days := types.Query{
			Range: map[string]types.RangeQuery{
				WHEN: types.DateRangeQuery{
					Gte: &fromTs,
				},
			},
		}
		termQueries = append(termQueries, days)
		// case 3: If only Tots present
	} else if req.Fromts == nil && req.Tots != nil {
		toTs := req.Tots.Format(LAYOUT)
		days := types.Query{
			Range: map[string]types.RangeQuery{
				WHEN: types.DateRangeQuery{
					Lte: &toTs,
				},
			},
		}
		termQueries = append(termQueries, days)
		// Case 4:If both fromts and tots are not specified only days are present
	} else if req.Ndays != nil && req.Fromts == nil && req.Tots == nil {
		if *req.Ndays > 0 {
			day := fmt.Sprintf("now-%dd/d", *req.Ndays)
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

	if req.App != nil {
		termQueries = append(termQueries, types.Query{
			Term: map[string]types.TermQuery{
				APP: {Value: req.App},
			},
		})
	}

	if req.Type != nil {
		termQueries = append(termQueries, types.Query{
			Term: map[string]types.TermQuery{
				TYPE: {Value: req.Type.String()},
			},
		})
	}

	if req.Who != nil {
		termQueries = append(termQueries, types.Query{
			Term: map[string]types.TermQuery{
				WHO: {Value: *req.Who},
			},
		})
	}

	if req.Class != nil {
		termQueries = append(termQueries, types.Query{
			Term: map[string]types.TermQuery{
				CLASS: {Value: *req.Class},
			},
		})

		// Instance must be considered only if class is specified
		if req.Instance != nil && req.Class != nil {
			termQueries = append(termQueries, types.Query{
				Term: map[string]types.TermQuery{
					INSTANCE: {Value: *req.Instance},
				},
			})
		}
	}

	if req.Op != nil {
		termQueries = append(termQueries, types.Query{
			Term: map[string]types.TermQuery{
				OP: {Value: *req.Op},
			},
		})
	}

	if req.RemoteIP != nil {
		termQueries = append(termQueries, types.Query{
			Term: map[string]types.TermQuery{
				REMOTE_IP: {Value: *req.RemoteIP},
			},
		})
	}

	// pri specifies that only logs of priority equal to or higher than the value given here will be returned.
	// If this parameter is present in the call, then data-change log entries are omitted from the result, because those log entries have no priority.
	if req.Pri != nil && req.Type != nil && LogType(*req.Type) != Change {
		reqPri := req.Pri.String()
		priFrom := slices.Index(Priority, reqPri)
		if priFrom >= 0 {
			requiredPri := Priority[priFrom:]
			if len(requiredPri) > 0 {
				var vals []types.FieldValue
				for _, val := range requiredPri {
					vals = append(vals, val)
				}
				termQueries = append(termQueries, types.Query{
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
