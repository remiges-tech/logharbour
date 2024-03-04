package logharbour

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/sortorder"
	"github.com/remiges-tech/alya/config"
	appconf "github.com/remiges-tech/logharbour/server/types"
)

type GetSetAttributesRequest struct {
	Client   string       `json:"client"` // client token which authorises the call to the function
	App      *string      `json:"app"`
	Type     *string      `json:"type"`
	Who      *string      `json:"who"`
	Class    *string      `json:"class"`
	Instance *string      `json:"instance"`
	Op       *string      `json:"op"`
	Fromts   *time.Time   `json:"fromts"`
	Tots     *time.Time   `json:"tots"`
	Ndays    *int         `json:"ndays"`
	RemoteIP *string      `json:"remoteIP"`
	Pri      *LogPriority `json:"pri"`
	SetAttr  string       `json:"setattr"`
}

const (
	INDEX     = "logharbour"
	APP       = "app"
	TYPE      = "type"
	WHO       = "who"
	CLASS     = "class"
	INSTANCE  = "instance"
	OP        = "op"
	REMOTE_IP = "remote_ip"
	PRI       = "pri"
	WHEN      = "when"
	MODULE    = "module"
	STATUS    = "status"
	SYSTEM    = "system"
	LAYOUT    = "2006-01-02T15:04:05"
)

func GetSet(req GetSetAttributesRequest) (setvalues map[string]int, err error) {
	var (
		appConfig   *appconf.AppConfig
		query       *types.Query
		dataMap     = make(map[string]int)
		placeholder = struct{}{}
	)

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

	client, err := elasticsearch.NewTypedClient(dbConfig)
	if err != nil {
		log.Fatalf("Failed to create db connection: %v", err)
		return
	}

	// Perform sanity check on setattr
	allowedAttributes := map[string]struct{}{
		APP:       placeholder,
		TYPE:      placeholder,
		OP:        placeholder,
		INSTANCE:  placeholder,
		MODULE:    placeholder,
		PRI:       placeholder,
		STATUS:    placeholder,
		REMOTE_IP: placeholder,
		SYSTEM:    placeholder,
		WHO:       placeholder,
	}

	if _, ok := allowedAttributes[req.SetAttr]; !ok {
		return nil, fmt.Errorf("attribute '%s' is not allowed for set retrieval", req.SetAttr)
	}

	// To get query based on request parameters
	query, err = getQuery(req)
	if err != nil {
		return nil, fmt.Errorf("invalid parameters : %v ", err)

	}

	sortByWhen := types.SortOptions{
		SortOptions: map[string]types.FieldSort{
			WHEN: {Order: &sortorder.Desc},
		},
	}

	res, err := client.Search().Index(INDEX).Request(&search.Request{
		Query: query,
		Sort:  []types.SortCombinations{sortByWhen},
	}).Do(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to search to elasticsearch: %w", err)
	}

	// Iterate over the hits in the response
	for _, hit := range res.Hits.Hits {
		// Convert the hit source to a map[string]interface{}
		var source map[string]interface{}
		err := json.Unmarshal(hit.Source_, &source)
		if err != nil {
			fmt.Errorf("Error unmarshalling hit source: %w", err)
			continue
		}

		fieldValue, ok := source[req.SetAttr]
		if !ok {
			// Handle the case where the field doesn't exist in the hit's source
			fmt.Errorf("Field %s not found in hit\n", req.SetAttr)
			continue
		}

		fieldValueString, ok := fieldValue.(string)
		if !ok {

			fmt.Errorf("Field %s is not a string in hit\n", req.SetAttr)
			continue
		}

		dataMap[fieldValueString]++
	}
	return dataMap, nil
}

// To form a query based on method parameters
func getQuery(req GetSetAttributesRequest) (*types.Query, error) {
	var (
		matchField []types.Query
		query      *types.Query
	)

	// time
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
			matchField = append(matchField, days)
		} else {
			return nil, fmt.Errorf("tots must be after fromts")
		}
	} else if req.Fromts != nil && req.Tots == nil {
		fromTs := req.Fromts.Format(LAYOUT)
		days := types.Query{
			Range: map[string]types.RangeQuery{
				WHEN: types.DateRangeQuery{
					Gte: &fromTs,
				},
			},
		}
		matchField = append(matchField, days)

	} else if req.Fromts == nil && req.Tots != nil {
		toTs := req.Tots.Format(LAYOUT)
		days := types.Query{
			Range: map[string]types.RangeQuery{
				WHEN: types.DateRangeQuery{
					Lte: &toTs,
				},
			},
		}
		matchField = append(matchField, days)
	}

	if req.App != nil {
		app := types.Query{
			Match: map[string]types.MatchQuery{
				APP: {Query: *req.App},
			},
		}
		matchField = append(matchField, app)
	}

	if req.Type != nil {
		app := types.Query{
			Match: map[string]types.MatchQuery{
				TYPE: {Query: *req.Type},
			},
		}
		matchField = append(matchField, app)
	}

	if req.Who != nil {
		who := types.Query{
			Match: map[string]types.MatchQuery{
				WHO: {Query: *req.Who},
			},
		}
		matchField = append(matchField, who)
	}

	if req.Class != nil {
		class := types.Query{
			Match: map[string]types.MatchQuery{
				CLASS: {Query: *req.Class},
			},
		}
		matchField = append(matchField, class)
	}

	if req.Instance != nil && req.Class != nil {
		instance := types.Query{
			Match: map[string]types.MatchQuery{
				INSTANCE: {Query: *req.Instance},
			},
		}
		matchField = append(matchField, instance)
	}

	if req.Op != nil {
		op := types.Query{
			Match: map[string]types.MatchQuery{
				OP: {Query: *req.Op},
			},
		}
		matchField = append(matchField, op)
	}

	if req.RemoteIP != nil {
		remoteIP := types.Query{
			Match: map[string]types.MatchQuery{
				REMOTE_IP: {Query: *req.RemoteIP},
			},
		}
		matchField = append(matchField, remoteIP)
	}
	if req.Pri != nil {
		pri := types.Query{
			Match: map[string]types.MatchQuery{
				PRI: {Query: req.Pri.String()},
			},
		}
		matchField = append(matchField, pri)
	}

	query = &types.Query{
		Bool: &types.BoolQuery{
			Must: matchField,
		},
	}

	return query, nil
}
