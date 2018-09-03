package main

import (
	"golang.org/x/net/context"
	"github.com/grafana/grafana_plugin_model/go/datasource"
	"github.com/hashicorp/go-plugin"
	"log"
	"time"
	"fmt"
	"github.com/hashicorp/consul/api"
	"encoding/json"
	"strconv"
	"strings"
	"regexp"
	"path"
)

type ConsulDatasource struct {
	plugin.NetRPCUnsupportedPlugin
}

func (t *ConsulDatasource) Query(ctx context.Context, req *datasource.DatasourceRequest) (*datasource.DatasourceResponse, error) {
	log.Printf("called consul plugin with: \n%v", req)

	consul, consulToken, err := NewConsulFromReq(req)
	if err != nil {
		return generateErrorResponse(err), nil
	}

	queries, err := parseQueries(req)
	if err != nil {
		return nil, fmt.Errorf("error parsing queries: %v", err)
	}

	return handleQueries(consul, consulToken, queries)
}

func handleQueries(consul *api.Client, consulToken string, queries []query) (*datasource.DatasourceResponse, error) {
	if len(queries) == 0 {
		return generateErrorResponse(fmt.Errorf("no queries found in request")), nil
	}
	if len(queries) == 1 && queries[0].Type == "test" {
		return handleTest(consul, consulToken)
	}

	switch queries[0].Format {
	case "timeseries":
		return handleTimeseries(consul, queries)
	case "table":
		return handleTable(consul, queries)
	}
	return generateErrorResponse(fmt.Errorf("unknown format, nothing to handle")), nil
}

func handleTest(consul *api.Client, consulToken string) (*datasource.DatasourceResponse, error) {
	e, _, err := consul.ACL().Info(consulToken, &api.QueryOptions{})
	if err != nil {
		return nil, fmt.Errorf("error retrieving acl info for token: %v", err)
	}
	if e.ID == consulToken {
		return &datasource.DatasourceResponse{}, nil
	}
	return &datasource.DatasourceResponse{
		Results: []*datasource.QueryResult{
			{
				Error: "consulToken is no valid token",
			},
		},
	}, nil
}

func handleTimeseries(consul *api.Client, qs []query) (*datasource.DatasourceResponse, error) {
	var qrs []*datasource.QueryResult
	for _, q := range qs {

		var qr *datasource.QueryResult
		var err error
		switch q.Type {
		case "get":
			qr, err = handleGet(consul, q.Target)
		case "keys":
			qr, err = handleKeys(consul, q.Target)
		case "tags":
			qr, err = handleTags(consul, q.Target, false)
		case "tagsrec":
			qr, err = handleTags(consul, q.Target, true)
		}
		if err != nil {
			return nil, err
		}
		qrs = append(qrs, qr)
	}

	return &datasource.DatasourceResponse{Results: qrs}, nil
}

func handleGet(consul *api.Client, target string) (*datasource.QueryResult, error) {
	if strings.HasSuffix(target, "/") {
		target = target[:len(target)-1]
	}

	var kvs []*api.KVPair

	kv, _, err := consul.KV().Get(target, &api.QueryOptions{RequireConsistent: true})
	if err != nil {
		return nil, fmt.Errorf("error consul get %s: %v", target, err)
	}
	if kv != nil {
		kvs = append(kvs, kv)
	}

	qr, err := generateQueryResultFromKVPairs(kvs)
	if err != nil {
		return nil, fmt.Errorf("error generating time series: %v", err)
	}
	return qr, nil
}

func handleKeys(consul *api.Client, target string) (*datasource.QueryResult, error) {
	if !strings.HasSuffix(target, "/") {
		target = target + "/"
	}
	keys, _, err := consul.KV().Keys(target, "/", &api.QueryOptions{RequireConsistent: true})
	if err != nil {
		return nil, fmt.Errorf("error consul list %s: %v", target, err)
	}
	return generateQueryResultFromKeys(keys), nil
}

func handleTags(consul *api.Client, target string, recursive bool) (*datasource.QueryResult, error) {
	if !strings.HasSuffix(target, "/") {
		target = target + "/"
	}
	separator := "/"
	if recursive {
		separator = ""
	}

	keys, _, err := consul.KV().Keys(target, separator, &api.QueryOptions{RequireConsistent: true})
	if err != nil {
		return nil, fmt.Errorf("error consul get %s: %v", target, err)
	}

	var tagKVs []*api.KVPair
	for _, key := range keys {
		tagKV, _, err := consul.KV().Get(key, &api.QueryOptions{RequireConsistent: true})
		if err != nil {
			return nil, fmt.Errorf("error consul get %s: %v", key, err)
		}
		if tagKV != nil {
			tagKVs = append(tagKVs, tagKV)
		}
	}
	qr, err := generateQueryResultWithTags(target, tagKVs)
	if err != nil {
		return nil, fmt.Errorf("error generating time series: %v", err)
	}
	return qr, nil
}

func handleTable(consul *api.Client, qs []query) (*datasource.DatasourceResponse, error) {

	var qrs []*datasource.QueryResult
	for _, q := range qs {

		targetRegex := strings.Replace(q.Target, "*", ".*", -1)
		regex, err := regexp.Compile(targetRegex)
		if err != nil {
			return nil, fmt.Errorf("error compiling regex: %v", err)
		}

		firstStar := strings.Index(q.Target, "*")
		prefix := q.Target
		if firstStar > 0 {
			prefix = q.Target[:firstStar]
		}

		columns := strings.Split(q.Columns, " ")

		keys, _, err := consul.KV().Keys(prefix, "", &api.QueryOptions{})
		if err != nil {
			return nil, fmt.Errorf("error gettings keys %s from consul: %v", prefix, err)
		}

		var matchingKeys []string
		for _, key := range keys {
			if regex.Match([]byte(key)) {
				matchingKeys = append(matchingKeys, key)
			}
		}

		var tableCols []*datasource.TableColumn
		var tableRows []*datasource.TableRow

		for i := 0; i < len(matchingKeys); i++ {
			firstRow := i == 0

			var tableRowValues []*datasource.RowValue

			for _, col := range columns {
				key := matchingKeys[i]

				colKey := calculateColumnKey(key, col)

				kv, _, err := consul.KV().Get(colKey, &api.QueryOptions{})
				var kvKey, kvValue string
				if err != nil || kv == nil {
					kvKey = "Not Found"
					kvValue = "Not Found"
				} else {
					lastSlash := strings.LastIndex(kv.Key, "/")
					kvKey = kv.Key[lastSlash+1:]
					kvValue = string(kv.Value)
				}
				if firstRow {
					tableCols = append(tableCols, &datasource.TableColumn{Name: kvKey})
				}
				tableRowValues = append(tableRowValues, &datasource.RowValue{Kind: datasource.RowValue_TYPE_STRING, StringValue: kvValue})
			}
			tableRows = append(tableRows, &datasource.TableRow{Values: tableRowValues})
		}
		qrs = append(qrs, &datasource.QueryResult{
			Tables: []*datasource.Table{
				{
					Columns: tableCols,
					Rows:    tableRows,
				},
			},
		})
	}

	return &datasource.DatasourceResponse{Results: qrs}, nil
}

func calculateColumnKey(key string, col string) string {
	for strings.HasPrefix(col, "../") {
		lastSlash := strings.LastIndex(key, "/")
		key = key[:lastSlash]
		col = strings.TrimPrefix(col, "../")
	}
	return path.Join(key, col)
}

func generateQueryResultFromKVPairs(kvs []*api.KVPair) (*datasource.QueryResult, error) {
	var series []*datasource.TimeSeries

	for _, kv := range kvs {
		value, err := strconv.ParseFloat(string(kv.Value), 64)
		if err != nil {
			return nil, err
		}
		series = append(series, &datasource.TimeSeries{
			Name: kv.Key,
			Points: []*datasource.Point{
				{
					Timestamp: time.Now().UnixNano(),
					Value:     value,
				},
			},
		})
	}
	return &datasource.QueryResult{
		Series: series,
	}, nil
}

func generateQueryResultFromKeys(keys []string) (*datasource.QueryResult) {
	var series []*datasource.TimeSeries

	for _, key := range keys {
		series = append(series, &datasource.TimeSeries{
			Name: key,
			Points: []*datasource.Point{
				{
					Timestamp: time.Now().UnixNano(),
					Value:     1,
				},
			},
		})
	}
	return &datasource.QueryResult{
		Series: series,
	}
}

func generateQueryResultWithTags(target string, tagKVs []*api.KVPair) (*datasource.QueryResult, error) {
	var series []*datasource.TimeSeries

	tags := map[string]string{}

	for _, tagKV := range tagKVs {
		tagName := strings.TrimPrefix(tagKV.Key, target)
		tagName = strings.Replace(tagName, "/", ".", -1)
		tags[tagName] = string(tagKV.Value)
	}

	series = append(series, &datasource.TimeSeries{
		Name: target,
		Tags: tags,
		Points: []*datasource.Point{
			{
				Timestamp: time.Now().UnixNano(),
				Value:     1,
			},
		},
	})

	return &datasource.QueryResult{
		Series: series,
	}, nil
}

func generateErrorResponse(err error) *datasource.DatasourceResponse {
	return &datasource.DatasourceResponse{
		Results: []*datasource.QueryResult{
			{
				Error: err.Error(),
			},
		},
	}
}

func parseQueries(req *datasource.DatasourceRequest) ([]query, error) {
	var qs []query
	for _, rawQuery := range req.Queries {
		var q query
		err := json.Unmarshal([]byte(rawQuery.ModelJson), &q)
		if err != nil {
			return nil, fmt.Errorf("error parsing query %s: %v", rawQuery.ModelJson, err)
		}
		qs = append(qs, q)
	}
	return qs, nil
}

type query struct {
	Target       string `json:"target"`
	Format       string `json:"format"`
	Type         string `json:"type"`
	RefID        string `json:"refId"`
	DatasourceId int    `json:"datasourceId"`
	Columns      string `json:"columns"`
}

func NewConsulFromReq(req *datasource.DatasourceRequest) (*api.Client, string, error) {
	consulToken := req.Datasource.DecryptedSecureJsonData["consulToken"]
	if consulToken == "" {
		return nil, "", fmt.Errorf("unable to get consulToken")
	}

	consulAddr := req.Datasource.Url
	if consulAddr == "" {
		return nil, "", fmt.Errorf("unable to get consulAddr")
	}

	consul, err := NewConsul(consulAddr, consulToken)
	if err != nil {
		return nil, "", fmt.Errorf("creating consul client failed: %v", err)
	}
	return consul, consulToken, nil
}

func NewConsul(consulAddr, consulToken string) (*api.Client, error) {
	conf := api.DefaultConfig()
	conf.Address = consulAddr
	conf.Token = consulToken
	conf.TLSConfig.InsecureSkipVerify = true

	return api.NewClient(conf)
}
