package main

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"

	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/hashicorp/consul/api"
)

func newDatasource() datasource.ServeOpts {
	im := datasource.NewInstanceManager(newDataSourceInstance)
	ds := &ConsulDataSource{
		im: im,
	}

	return datasource.ServeOpts{
		QueryDataHandler:   ds,
		CheckHealthHandler: ds,
	}
}

type ConsulDataSource struct {
	im instancemgmt.InstanceManager
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (td *ConsulDataSource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	log.DefaultLogger.Debug("QueryData", "request", req)

	consul, err := td.getConsulClient(req.PluginContext)
	if err != nil {
		return nil, err
	}

	queries, err := parseQueries(req)
	if err != nil {
		return nil, err
	}

	if len(queries) == 0 {
		return nil, fmt.Errorf("no queries found in request")
	}

	return query(ctx, consul, queries), nil
}

func (td *ConsulDataSource) getConsulClient(pluginCtx backend.PluginContext) (*api.Client, error) {
	instance, err := td.im.Get(pluginCtx)
	if err != nil {
		return nil, fmt.Errorf("could not get plugin instance: %v", err)
	}
	instanceSettings, ok := instance.(*instanceSettings)
	if !ok {
		return nil, fmt.Errorf("could not get plugin instance")
	}
	return instanceSettings.consul, nil
}

type queryModel struct {
	Format  string `json:"format"`
	Target  string `json:"target"`
	Type    string `json:"type"`
	Columns string `json:"columns"`
	Error   error
}

func parseQueries(req *backend.QueryDataRequest) (map[string]queryModel, error) {
	log.DefaultLogger.Debug("parseQueries", "queries", req.Queries)

	queries := map[string]queryModel{}
	for _, rawQuery := range req.Queries {
		var q queryModel
		if err := json.Unmarshal(rawQuery.JSON, &q); err != nil {
			queries[rawQuery.RefID] = queryModel{Error: fmt.Errorf("error parsing query %s: %v", rawQuery.JSON, err)}
			continue
		}
		queries[rawQuery.RefID] = q
	}
	return queries, nil
}

func query(ctx context.Context, consul *api.Client, queries map[string]queryModel) *backend.QueryDataResponse {
	log.DefaultLogger.Debug("query", "queries", queries)

	response := backend.NewQueryDataResponse()
	for refID, query := range queries {
		if query.Error != nil {
			response.Responses[refID] = backend.DataResponse{Error: query.Error}
			continue
		}

		switch query.Format {
		case "", "timeseries":
			response.Responses[refID] = queryTimeSeries(ctx, consul, query)
		case "table":
			response.Responses[refID] = queryTable(ctx, consul, query)
		default:
			response.Responses[refID] = backend.DataResponse{Error: fmt.Errorf("unknown format %s", query.Format)}
		}
	}

	return response
}

func queryTimeSeries(ctx context.Context, consul *api.Client, query queryModel) backend.DataResponse {
	log.DefaultLogger.Debug("queryTimeSeries", "query", query)

	if query.Format == "" {
		log.DefaultLogger.Debug("format is empty. defaulting to time series")
		query.Format = "timeseries"
	}
	if query.Type == "" {
		log.DefaultLogger.Debug("type is empty. defaulting to get value")
		query.Type = "get"
	}

	// clean target
	q := strings.Replace(query.Target, "\\.", ".", -1)

	switch query.Type {
	case "get":
		return handleGet(ctx, consul, q)
	case "keys":
		return handleKeys(ctx, consul, q)
	case "tags":
		return handleTags(ctx, consul, q, false)
	case "tagsrec":
		return handleTags(ctx, consul, q, true)
	}
	return backend.DataResponse{Error: fmt.Errorf("unknown query type: %s", query.Type)}
}

func handleGet(ctx context.Context, consul *api.Client, target string) backend.DataResponse {
	log.DefaultLogger.Debug("handleGet", "target", target)

	if strings.HasSuffix(target, "/") {
		target = target[:len(target)-1]
	}

	var kvs []*api.KVPair
	kv, _, err := consul.KV().Get(target, (&api.QueryOptions{RequireConsistent: true}).WithContext(ctx))
	if err != nil {
		return backend.DataResponse{Error: fmt.Errorf("error consul get %s: %v", target, err)}
	}
	if kv != nil {
		kvs = append(kvs, kv)
	}

	return generateDataResponseFromKV(kvs)
}

func handleKeys(ctx context.Context, consul *api.Client, target string) backend.DataResponse {
	log.DefaultLogger.Debug("handleKeys", "target", target)

	if !strings.HasSuffix(target, "/") {
		target = target + "/"
	}

	keys, _, err := consul.KV().Keys(target, "/", (&api.QueryOptions{RequireConsistent: true}).WithContext(ctx))
	if err != nil {
		return backend.DataResponse{Error: fmt.Errorf("error consul keys %s: %v", target, err)}
	}
	return generateDataResponseFromKeys(keys)
}

func handleTags(ctx context.Context, consul *api.Client, target string, recursive bool) backend.DataResponse {
	log.DefaultLogger.Debug("handleTags", "target", target)

	if !strings.HasSuffix(target, "/") {
		target = target + "/"
	}
	separator := "/"
	if recursive {
		separator = ""
	}

	keys, _, err := consul.KV().Keys(target, separator, (&api.QueryOptions{RequireConsistent: true}).WithContext(ctx))
	if err != nil {
		return backend.DataResponse{Error: fmt.Errorf("error consul keys %s: %v", target, err)}
	}

	var tagKVs []*api.KVPair
	for _, key := range keys {
		tagKV, _, err := consul.KV().Get(key, (&api.QueryOptions{RequireConsistent: true}).WithContext(ctx))
		if err != nil {
			return backend.DataResponse{Error: fmt.Errorf("error consul get %s: %v", key, err)}
		}
		if tagKV != nil {
			tagKVs = append(tagKVs, tagKV)
		}
	}
	return generateDataResponseWithTags(target, tagKVs)
}

func generateDataResponseFromKV(kvs []*api.KVPair) backend.DataResponse {
	log.DefaultLogger.Debug("generateDataResponseFromKV", "kv", kvs)

	response := backend.DataResponse{}

	for _, kv := range kvs {
		floatValue, err := strconv.ParseFloat(string(kv.Value), 64)
		if err != nil {
			return backend.DataResponse{Error: err}
		}

		now := time.Now()
		value := []float64{floatValue}
		log.DefaultLogger.Debug("appending data frame to response", "name", kv.Key, "time", now, "value", value)
		response.Frames = append(response.Frames, data.NewFrame(kv.Key,
			data.NewField("time", nil, []time.Time{now}),
			data.NewField("values", nil, value),
		))
	}
	return response
}

func generateDataResponseFromKeys(keys []string) backend.DataResponse {
	log.DefaultLogger.Debug("generateDataResponseFromKeys", "keys", keys)

	response := backend.DataResponse{}

	for _, key := range keys {
		now := time.Now()
		value := []float64{1}
		log.DefaultLogger.Debug("appending data frame to response", "name", key, "time", now, "value", value)
		response.Frames = append(response.Frames, data.NewFrame(key,
			data.NewField("time", nil, []time.Time{now}),
			data.NewField("values", nil, value),
		))
	}
	return response
}

func generateDataResponseWithTags(target string, tagKVs []*api.KVPair) backend.DataResponse {
	log.DefaultLogger.Debug("generateDataResponseWithTags", "tags", tagKVs)

	response := backend.DataResponse{}

	tags := data.Labels{}
	for _, tagKV := range tagKVs {
		tagName := strings.TrimPrefix(tagKV.Key, target)
		tagName = strings.Replace(tagName, "/", ".", -1)
		tags[tagName] = string(tagKV.Value)
	}

	now := time.Now()
	value := []float64{1}
	log.DefaultLogger.Debug("appending data frame to response", "name", target, "time", now, "value", value, "tags", tags)
	response.Frames = append(response.Frames, data.NewFrame(target,
		data.NewField("time", nil, []time.Time{now}),
		data.NewField("values", tags, value),
	))
	return response
}

func queryTable(ctx context.Context, consul *api.Client, query queryModel) backend.DataResponse {
	log.DefaultLogger.Debug("queryTable", "query", query)
	defer func() {
		if err := recover(); err != nil {
			log.DefaultLogger.Error("Recovered in queryTable", "err", err)
		}
	}()

	// Compile targetRegex
	target := strings.Replace(query.Target, "*", ".*", -1)
	targetRegex, err := regexp.Compile(target)
	if err != nil {
		return backend.DataResponse{Error: fmt.Errorf("error compiling regex %s: %v", target, err)}
	}

	// Calculate Prefix to execute consul.KV().Keys() on
	firstStar := strings.Index(query.Target, "*")
	prefix := query.Target
	if firstStar > 0 {
		prefix = query.Target[:firstStar]
	}

	// Get keys with prefix
	log.DefaultLogger.Debug("queryTable: get keys below prefix", "prefix", prefix)
	keys, _, err := consul.KV().Keys(prefix, "", &api.QueryOptions{})
	if err != nil {
		return backend.DataResponse{Error: fmt.Errorf("error gettings keys %s from consul: %v", prefix, err)}
	}

	// Filter keys that match the targetRegex
	// One matchingKey will be one line in the table
	var matchingKeys []string
	for _, key := range keys {
		if targetRegex.Match([]byte(key)) {
			matchingKeys = append(matchingKeys, key)
		}
	}

	columns := strings.Split(query.Columns, ",")

	fields := []*data.Field{}
	for rowIdx, key := range matchingKeys {
		for colIdx, col := range columns {

			// calculate key for column value
			colKey := calculateColumnKey(key, col)

			// get field from Consul
			field, value := getColumnValueForKey(ctx, consul, colKey)

			// If it's the first row ,append it to the fields array
			if rowIdx == 0 {
				log.DefaultLogger.Debug("queryTable: appending first row field", "value", value, "rowIdx", rowIdx, "colIdx", colIdx)
				fields = append(fields, field)
				continue
			}

			// Else, append it to the field of the current column
			log.DefaultLogger.Debug("queryTable: appending value to field", "value", value, "rowIdx", rowIdx, "colIdx", colIdx)
			fields[colIdx].Append(value)
		}
	}

	return backend.DataResponse{Frames: []*data.Frame{data.NewFrame("table", fields...)}}
}

func getColumnValueForKey(ctx context.Context, consul *api.Client, colKey string) (*data.Field, interface{}) {
	log.DefaultLogger.Debug("getColumnValueForKey", "key", colKey)

	kv, _, err := consul.KV().Get(colKey, (&api.QueryOptions{}).WithContext(ctx))
	if err != nil || kv == nil {
		return data.NewField(path.Base(colKey), nil, []string{"Not Found"}), "Not Found"
	}

	// try to parse int
	intValue, err := strconv.ParseInt(string(kv.Value), 10, 64)
	if err != nil {
		return data.NewField(path.Base(colKey), nil, []string{string(kv.Value)}), string(kv.Value)
	}

	return data.NewField(path.Base(colKey), nil, []int64{intValue}), intValue
}

func calculateColumnKey(key string, col string) string {
	for strings.HasPrefix(col, "../") {
		lastSlash := strings.LastIndex(key, "/")
		key = key[:lastSlash]
		col = strings.TrimPrefix(col, "../")
	}
	return path.Join(key, col)
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (td *ConsulDataSource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	log.DefaultLogger.Debug("CheckHealth", "request", req)

	consul, err := td.getConsulClient(req.PluginContext)
	if err != nil {
		return nil, err
	}

	if _, err := consul.Status().Leader(); err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("Consul health check failed: %v", err),
		}, nil
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Consul data source is working",
	}, nil
}

type instanceSettings struct {
	consul *api.Client
}

type jsonData struct {
	ConsulAddr string
}

func newDataSourceInstance(setting backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	jData := jsonData{}

	if err := json.Unmarshal(setting.JSONData, &jData); err != nil {
		return nil, fmt.Errorf("error decoding jsonData: %v", err)
	}

	if jData.ConsulAddr == "" {
		log.DefaultLogger.Error("newDataSourceInstance", "ConsulAddr", jData.ConsulAddr, "err", "consulAddr should not be empty")
		return nil, fmt.Errorf("consulAddr should not be empty")
	}

	conf := api.DefaultConfig()
	conf.Address = jData.ConsulAddr
	conf.Token = setting.DecryptedSecureJSONData["consulToken"]
	conf.TLSConfig.InsecureSkipVerify = true

	client, err := api.NewClient(conf)
	if err != nil {
		return nil, fmt.Errorf("error creating consul client: %v", err)
	}
	return &instanceSettings{
		consul: client,
	}, nil
}

func (s *instanceSettings) Dispose() {
}
