package main

import (
	"encoding/json"
	"fmt"
	"log"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/grafana_plugin_model/go/datasource"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-plugin"
	"golang.org/x/net/context"
)

// ConsulDatasource implements a datasource which connects to a Consul instance
type ConsulDatasource struct {
	plugin.NetRPCUnsupportedPlugin
}

// Query returns responses to req based on data in Consul
func (t *ConsulDatasource) Query(ctx context.Context, req *datasource.DatasourceRequest) (*datasource.DatasourceResponse, error) {
	log.Printf("called consul plugin with: \n%v", req)

	consul, err := newConsulFromReq(req)
	if err != nil {
		return generateErrorResponse(err, ""), nil
	}

	queries, err := parseQueries(req)
	if err != nil {
		return generateErrorResponse(fmt.Errorf("error parsing queries: %v", err), ""), nil
	}

	return handleQueries(consul, queries), nil
}

func handleQueries(consul *api.Client, queries []query) *datasource.DatasourceResponse {
	if len(queries) == 0 {
		return generateErrorResponse(fmt.Errorf("no queries found in request"), "")
	}
	if len(queries) == 1 && queries[0].Type == "test" {
		return handleTest(consul, queries[0].RefID)
	}

	switch queries[0].Format {
	case "timeseries":
		return handleTimeseries(consul, queries)
	case "table":
		return handleTable(consul, queries)
	}
	return generateErrorResponse(fmt.Errorf("unknown format, nothing to handle"), "")
}

func handleTest(consul *api.Client, refID string) *datasource.DatasourceResponse {
	_, err := consul.Status().Leader()
	if err != nil {
		return generateErrorResponse(fmt.Errorf("error retrieving acl info for token: %v", err), refID)
	}
	return &datasource.DatasourceResponse{}
}

func handleTimeseries(consul *api.Client, qs []query) *datasource.DatasourceResponse {
	var qrs []*datasource.QueryResult
	for _, q := range qs {

		target := cleanTarget(q.Target)

		var qr *datasource.QueryResult
		var err error
		switch q.Type {
		case "get":
			qr, err = handleGet(consul, target)
		case "keys":
			qr, err = handleKeys(consul, target)
		case "tags":
			qr, err = handleTags(consul, target, false)
		case "tagsrec":
			qr, err = handleTags(consul, target, true)
		}
		if err != nil {
			return generateErrorResponse(err, q.RefID)
		}
		if qr == nil {
			return generateErrorResponse(fmt.Errorf("unknown type %q for format timeseries", q.Type), q.RefID)
		}
		qr.RefId = q.RefID
		qrs = append(qrs, qr)
	}

	return &datasource.DatasourceResponse{Results: qrs}
}

func cleanTarget(target string) string {
	return strings.Replace(target, "\\.", ".", -1)
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

func handleTable(consul *api.Client, qs []query) *datasource.DatasourceResponse {

	var qrs []*datasource.QueryResult
	for _, q := range qs {

		targetRegex := strings.Replace(q.Target, "*", ".*", -1)
		regex, err := regexp.Compile(targetRegex)
		if err != nil {
			return generateErrorResponse(fmt.Errorf("error compiling regex: %v", err), q.RefID)
		}

		firstStar := strings.Index(q.Target, "*")
		prefix := q.Target
		if firstStar > 0 {
			prefix = q.Target[:firstStar]
		}

		columns := strings.Split(q.Columns, ",")

		keys, _, err := consul.KV().Keys(prefix, "", &api.QueryOptions{})
		if err != nil {
			return generateErrorResponse(fmt.Errorf("error gettings keys %s from consul: %v", prefix, err), q.RefID)
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

				if firstRow {
					tableCols = append(tableCols, &datasource.TableColumn{Name: path.Base(colKey)})
				}

				kv, _, err := consul.KV().Get(colKey, &api.QueryOptions{})
				var kvValue string
				if err != nil || kv == nil {
					tableRowValues = append(tableRowValues, &datasource.RowValue{Kind: datasource.RowValue_TYPE_STRING, StringValue: "Not Found"})
				} else {
					kvValue = string(kv.Value)
					if i, err := strconv.ParseInt(kvValue, 10, 64); err != nil {
						tableRowValues = append(tableRowValues, &datasource.RowValue{Kind: datasource.RowValue_TYPE_STRING, StringValue: kvValue})
					} else {
						tableRowValues = append(tableRowValues, &datasource.RowValue{Kind: datasource.RowValue_TYPE_INT64, Int64Value: i})
					}
				}
			}
			tableRows = append(tableRows, &datasource.TableRow{Values: tableRowValues})
		}
		qrs = append(qrs, &datasource.QueryResult{
			RefId: q.RefID,
			Tables: []*datasource.Table{
				{
					Columns: tableCols,
					Rows:    tableRows,
				},
			},
		})
	}

	return &datasource.DatasourceResponse{Results: qrs}
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
					Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
					Value:     value,
				},
			},
		})
	}
	return &datasource.QueryResult{
		Series: series,
	}, nil
}

func generateQueryResultFromKeys(keys []string) *datasource.QueryResult {
	var series []*datasource.TimeSeries

	for _, key := range keys {
		series = append(series, &datasource.TimeSeries{
			Name: key,
			Points: []*datasource.Point{
				{
					Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
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
				Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
				Value:     1,
			},
		},
	})

	return &datasource.QueryResult{
		Series: series,
	}, nil
}

func generateErrorResponse(err error, refID string) *datasource.DatasourceResponse {
	return &datasource.DatasourceResponse{
		Results: []*datasource.QueryResult{
			{
				RefId: refID,
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

func newConsulFromReq(req *datasource.DatasourceRequest) (*api.Client, error) {
	consulToken := req.Datasource.DecryptedSecureJsonData["consulToken"]

	var jsonData map[string]interface{}
	err := json.Unmarshal([]byte(req.Datasource.JsonData), &jsonData)
	if err != nil {
		return nil, fmt.Errorf("unable to get consulAddr: %v", err)
	}

	consulAddr := jsonData["consulAddr"].(string)
	if consulAddr == "" {
		return nil, fmt.Errorf("unable to get consulAddr")
	}

	consul, err := newConsul(req.Datasource.Id, consulAddr, consulToken)
	if err != nil {
		return nil, fmt.Errorf("creating consul client failed: %v", err)
	}
	return consul, nil
}

type consulClientEntry struct {
	consulAddr  string
	consulToken string
	client      *api.Client
}

var consulClientCache = map[int64]consulClientEntry{}

func newConsul(datasourceId int64, consulAddr, consulToken string) (*api.Client, error) {
	if client, ok := clientInCache(datasourceId, consulAddr, consulToken); ok {
		return client, nil
	}

	conf := api.DefaultConfig()
	conf.Address = consulAddr
	conf.Token = consulToken
	conf.TLSConfig.InsecureSkipVerify = true

	client, err := api.NewClient(conf)
	if err != nil {
		return nil, fmt.Errorf("error creating consul client: %v", err)
	}
	consulClientCache[datasourceId] = consulClientEntry{consulAddr, consulToken, client}

	return client, nil
}

func clientInCache(datasourceId int64, consulAddr, consulToken string) (*api.Client, bool) {
	entry, ok := consulClientCache[datasourceId]
	if !ok {
		return nil, false
	}
	if entry.consulAddr != consulAddr || entry.consulToken != consulToken {
		return nil, false
	}
	return entry.client, true
}
