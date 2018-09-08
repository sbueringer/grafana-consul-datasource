package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"

	"bytes"
	"fmt"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
	"context"
	"github.com/grafana/grafana_plugin_model/go/datasource"
)

func TestQuery(t *testing.T) {

	srv, _ := setupTestServer(t)
	defer srv.Stop()

	var tests = []struct {
		dr          *datasource.DatasourceRequest
		wantErr     bool
	}{
		{
			dr: &datasource.DatasourceRequest{
				Datasource: &datasource.DatasourceInfo{
					DecryptedSecureJsonData: map[string]string{
						"consulToken": srv.Config.ACLMasterToken,
					},
					JsonData: fmt.Sprintf("{\"consulAddr\":\"%s\"}", srv.HTTPAddr),
				},
				Queries: []*datasource.Query{
					{
						ModelJson: "{\"Type\":\"test\"}",
					},
				},
			},
			wantErr: false,
		},
		{
			dr: &datasource.DatasourceRequest{
				Datasource: &datasource.DatasourceInfo{
					DecryptedSecureJsonData: map[string]string{
						"consulToken": srv.Config.ACLMasterToken,
					},
					JsonData: fmt.Sprintf("{\"consulAddr\":\"%s\"}", srv.HTTPAddr),
				},
				Queries: []*datasource.Query{
					{
						ModelJson: "{abc}",
					},
				},
			},
			wantErr: true,
		},
		{
			dr: &datasource.DatasourceRequest{
				Datasource: &datasource.DatasourceInfo{
					JsonData: fmt.Sprintf("{\"consulAddr\":\"%s\"}", srv.HTTPAddr),
				},
			},
			wantErr: true,
		},
	}

	ds := &ConsulDatasource{}

	for _, test := range tests {
		dr, err := ds.Query(context.Background(), test.dr)

		if !test.wantErr && err != nil {
			t.Fatalf("Expected no error, but received one: %v", err)
		}

		if !test.wantErr && len(dr.Results) > 0 {
			t.Fatalf("Expected no results, but received: %+v", dr.Results)
		}

		if test.wantErr && dr.Results[0].Error == "" {
			t.Fatal("Expected error, but didn't got one")
		}
	}
}

func TestHandleQueries(t *testing.T) {

	var tests = []struct {
		query       *query
		golden      string
		consulToken string
		wantErr     string
	}{
		{
			query:  &query{},
			golden: "empty-query-error.json",
		},
		{
			query:  nil,
			golden: "no-query-error.json",
		},
		{
			query: &query{
				Type: "test",
			},
			golden: "test.json",
		},
		{
			query: &query{
				Type: "test",
			},
			consulToken: "wrongToken",
			golden:      "test-error.json",
		},
		{
			query: &query{
				Format: "timeseries",
				Type:   "get",
				Target: "registry/apiregistration.k8s.io/apiservices/v1beta1.rbac.authorization.k8s.io/spec/groupPriorityMinimum",
			},
			golden: "timeseries-get.json",
		},
		{
			query: &query{
				Format: "timeseries",
				Type:   "get",
				Target: "registry/apiregistration.k8s.io/apiservices/v1beta1.rbac.authorization.k8s.io/spec/groupPriorityMinimum/",
			},
			golden: "timeseries-get.json",
		},
		{
			query: &query{
				Format: "timeseries",
				Type:   "keys",
				Target: "registry/apiregistration.k8s.io/apiservices",
			},
			golden: "timeseries-keys.json",
		},
		{
			query: &query{
				Format: "timeseries",
				Type:   "keys",
				Target: "registry/apiregistration.k8s.io/apiservices/",
			},
			golden: "timeseries-keys.json",
		},
		{
			query: &query{
				Format: "timeseries",
				Type:   "tags",
				Target: "registry/apiregistration.k8s.io/apiservices/v1.authentication.k8s.io",
			},
			golden: "timeseries-tags.json",
		},
		{
			query: &query{
				Format: "timeseries",
				Type:   "tags",
				Target: "registry/apiregistration.k8s.io/apiservices/v1.authentication.k8s.io/",
			},
			golden: "timeseries-tags.json",
		},
		{
			query: &query{
				Format: "timeseries",
				Type:   "tagsrec",
				Target: "registry/apiregistration.k8s.io/apiservices/v1.autoscaling",
			},
			golden: "timeseries-tagsrec.json",
		},
		{
			query: &query{
				Format: "timeseries",
				Type:   "tagsrec",
				Target: "registry/apiregistration.k8s.io/apiservices/v1.autoscaling/",
			},
			golden: "timeseries-tagsrec.json",
		},
		{
			query: &query{
				Format: "timeseries",
				Type:   "unknown",
				Target: "registry",
			},
			golden: "timeseries-unknown.json",
		},
		{
			query: &query{
				Format:  "table",
				Target:  "registry/apiregistration.k8s.io/apiservices/*/name",
				Columns: "../name,../kind,../apiVersion,../spec/group,../spec/groupPriorityMinimum,../spec/version,../spec/versionPriority",
			},
			golden: "table.json",
		},
	}

	srv, consul := setupTestServer(t)
	defer srv.Stop()

	writeGolden := false
	writeGolden = true

	for _, test := range tests {

		consulToken := test.consulToken
		if consulToken == "" {
			consulToken = srv.Config.ACLMasterToken
		}

		var qs []query
		if test.query != nil {
			qs = append(qs, *test.query)
		}

		qrs, err := handleQueries(consul, consulToken, qs)
		if err != nil {
			if test.wantErr == "" {
				t.Fatalf("error handling queries: %v", err)
			} else {
				if strings.Contains(err.Error(), test.wantErr) {
					continue
				} else {
					t.Fatalf("Expected error %s, but did get: %v", test.wantErr, err)
				}
			}
		} else {
			if test.wantErr != "" {
				t.Fatalf("Expected error %s, but didn't get one.", test.wantErr)
			}
		}

		// Overwrite timestamp, so the results can be compared against golden
		if qrs.Results != nil {
			for _, r := range qrs.Results {
				if r != nil {
					for _, s := range r.Series {
						for _, p := range s.Points {
							p.Timestamp = 0
						}
					}
				}
			}
		}

		text, _ := json.MarshalIndent(qrs, "", "  ")

		goldenFile := path.Join("test/golden", test.golden)
		if writeGolden {
			ioutil.WriteFile(path.Join(goldenFile), text, os.ModePerm)
		}

		golden, err := ioutil.ReadFile(goldenFile)
		if err != nil {
			t.Fatalf("could not read golden file %s: %v", goldenFile, err)
		}

		dmp := diffmatchpatch.New()

		diffs := dmp.DiffMain(string(golden), string(text), false)

		if !(len(diffs) == 1 && diffs[0].Type == diffmatchpatch.DiffEqual) {
			t.Errorf("query result for query %+v was not as expected, diff is:\n", test.query)
			fmt.Println(diffPrettyText(diffs))
		}
	}
}

func diffPrettyText(diffs []diffmatchpatch.Diff) string {
	var buff bytes.Buffer
	for _, diff := range diffs {
		text := diff.Text

		switch diff.Type {
		case diffmatchpatch.DiffInsert:
			_, _ = buff.WriteString("\x1b[32m")
			text = strings.Replace(text, "\n", "\n\x1b[32m", -1)
			_, _ = buff.WriteString(text)
			_, _ = buff.WriteString("\x1b[0m")
		case diffmatchpatch.DiffDelete:
			_, _ = buff.WriteString("\x1b[31m")
			text = strings.Replace(text, "\n", "\n\x1b[31m", -1)
			_, _ = buff.WriteString(text)
			_, _ = buff.WriteString("\x1b[0m")
		case diffmatchpatch.DiffEqual:
			newLineCount := strings.Count(text, "\n")
			if newLineCount > 3 {
				firstNewLine := strings.Index(text, "\n")
				secondNewLine := firstNewLine + strings.Index(text[firstNewLine+1:], "\n")
				secondLastNewLine := strings.LastIndex(text[:strings.LastIndex(text, "\n")], "\n")
				firstText := text[:secondNewLine+1]
				lastText := text[secondLastNewLine:]
				text = firstText + "\n..." + lastText
			}
			_, _ = buff.WriteString(text)
		}
	}

	return buff.String()
}

func setupTestServer(t *testing.T) (*testutil.TestServer, *api.Client) {
	srv, err := testutil.NewTestServerConfig(func(c *testutil.TestServerConfig) {
		//c.Stdout = ioutil.Discard
		//c.Stderr = ioutil.Discard
		c.Datacenter = "default"
		c.ACLDefaultPolicy = "allow"
		c.ACLDatacenter = "default"
		c.ACLMasterToken = "master"
		c.LogLevel = "warn"
	})
	if err != nil {
		t.Fatal(err)
	}

	consul, err := newConsul(1, srv.HTTPAddr, srv.Config.ACLMasterToken)
	if err != nil {
		t.Fatalf("could not create consul client: %v", err)
	}

	_, _, err = consul.ACL().Create(&api.ACLEntry{
		ID:    "master",
		Type:  "client",
		Rules: "key \"\" { policy = \"write\" }",
	}, &api.WriteOptions{})
	if err != nil {
		panic(fmt.Errorf("could not create master acl: %v", err))
	}

	files, err := filepath.Glob("test/*.json")
	if err != nil {
		t.Fatalf("error getting import json files")
	}

	for _, file := range files {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			t.Fatalf("error reading import json file %s: %v", file, err)
		}
		var entries []*Entry
		if err := json.Unmarshal([]byte(data), &entries); err != nil {
			t.Fatalf("Cannot unmarshal data: %s", err)
		}

		for _, entry := range entries {
			pair := &api.KVPair{
				Key:   entry.Key,
				Flags: entry.Flags,
				Value: []byte(entry.Value),
			}
			if _, err := consul.KV().Put(pair, nil); err != nil {
				t.Fatalf("Error! Failed writing data for key %s: %s", pair.Key, err)
			}
			//t.Logf("Imported: %s", pair.Key)
		}
	}

	return srv, consul
}

type Entry struct {
	Key   string `json:"key"`
	Flags uint64 `json:"flags"`
	Value string `json:"value"`
}
