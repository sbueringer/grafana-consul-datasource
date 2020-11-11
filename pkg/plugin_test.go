package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestQuery(t *testing.T) {

	var tests = []struct {
		name    string
		queries map[string]queryModel
		golden  string
	}{
		{
			name:    "empty query",
			queries: map[string]queryModel{},
			golden:  "empty-query.json",
		},
		{
			name:    "nil query",
			queries: nil,
			golden:  "no-query.json",
		},
		{
			name: "timeseries get",
			queries: map[string]queryModel{
				"abc": {
					Format: "timeseries",
					Type:   "get",
					Target: "registry/apiregistration.k8s.io/apiservices/v1beta1.rbac.authorization.k8s.io/spec/groupPriorityMinimum",
				},
			},
			golden: "timeseries-get.json",
		},
		{
			name: "timeseries get with trailing slash",
			queries: map[string]queryModel{
				"abc": {
					Format: "timeseries",
					Type:   "get",
					Target: "registry/apiregistration.k8s.io/apiservices/v1beta1.rbac.authorization.k8s.io/spec/groupPriorityMinimum/",
				},
			},
			golden: "timeseries-get.json",
		},
		{
			name: "timeseries keys",
			queries: map[string]queryModel{
				"def": {
					Format: "timeseries",
					Type:   "keys",
					Target: "registry/apiregistration.k8s.io/apiservices",
				},
			},
			golden: "timeseries-keys.json",
		},
		{
			name: "timeseries keys with trailing slash",
			queries: map[string]queryModel{
				"def": {
					Format: "timeseries",
					Type:   "keys",
					Target: "registry/apiregistration.k8s.io/apiservices/",
				},
			},
			golden: "timeseries-keys.json",
		},
		{
			name: "timeseries tags",
			queries: map[string]queryModel{
				"xyz": {
					Format: "timeseries",
					Type:   "tags",
					Target: "registry/apiregistration.k8s.io/apiservices/v1.authentication.k8s.io",
				},
			},
			golden: "timeseries-tags.json",
		},
		{
			name: "timeseries tags with trailing slash",
			queries: map[string]queryModel{
				"xyz": {
					Format: "timeseries",
					Type:   "tags",
					Target: "registry/apiregistration.k8s.io/apiservices/v1.authentication.k8s.io/",
				},
			},
			golden: "timeseries-tags.json",
		},
		{
			name: "timeseries tagsrec",
			queries: map[string]queryModel{
				"xyz": {
					Format: "timeseries",
					Type:   "tagsrec",
					Target: "registry/apiregistration.k8s.io/apiservices/v1.autoscaling",
				},
			},
			golden: "timeseries-tagsrec.json",
		},
		{
			name: "timeseries tagsrec with trailing slash",
			queries: map[string]queryModel{
				"xyz": {
					Format: "timeseries",
					Type:   "tagsrec",
					Target: "registry/apiregistration.k8s.io/apiservices/v1.autoscaling/",
				},
			},
			golden: "timeseries-tagsrec.json",
		},
		{
			name: "timeseries type unknown",
			queries: map[string]queryModel{
				"xyz": {
					Format: "timeseries",
					Type:   "unknown",
					Target: "registry",
				},
			},
			golden: "timeseries-unknown.json",
		},
		{
			name: "table",
			queries: map[string]queryModel{
				"xyz": {
					Format:  "table",
					Target:  "registry/apiregistration.k8s.io/apiservices/*/name",
					Columns: "../name,../kind,../apiVersion,../spec/group,../spec/groupPriorityMinimum,../spec/version,../spec/versionPriority",
				},
			},
			golden: "table.json",
		},
	}

	srv, consul := setupTestServer(t)
	defer srv.Stop()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := query(context.TODO(), consul, tt.queries)

			text, _ := json.MarshalIndent(response, "", "  ")

			goldenFile := path.Join("testdata/golden", tt.golden)
			writeGolden := false
			if writeGolden {
				ioutil.WriteFile(path.Join(goldenFile), text, os.ModePerm)
			}

			// errors and values are not printed, because they are unexported fields
			golden, err := ioutil.ReadFile(goldenFile)
			if err != nil {
				t.Fatalf("could not read golden file %s: %v", goldenFile, err)
			}

			dmp := diffmatchpatch.New()

			diffs := dmp.DiffMain(string(golden), string(text), false)

			if !(len(diffs) == 1 && diffs[0].Type == diffmatchpatch.DiffEqual) {
				t.Errorf("result for query %+v was not as expected\n", tt.queries)
				t.Logf("diff to golden %s:\n%s", tt.golden, diffPrettyText(diffs))
			}
		})
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
	srv, err := testutil.NewTestServerConfigT(&testing.T{}, func(c *testutil.TestServerConfig) {
		//c.Stdout = ioutil.Discard
		//c.Stderr = ioutil.Discard
		c.Ports = &testutil.TestPortConfig{
			HTTP: 8500,
		}

		c.Datacenter = "default"
		c.LogLevel = "debug"
	})
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("Importing example data to Consul")

	consul, err := newConsul(srv.HTTPAddr, srv.Config.ACLMasterToken)
	if err != nil {
		t.Fatalf("could not create consul client: %v", err)
	}

	files, err := filepath.Glob("testdata/*.json")
	if err != nil {
		t.Fatalf("error getting import json files")
	}

	for _, file := range files {
		fmt.Printf("Importing: %s\n", file)

		data, err := ioutil.ReadFile(file)
		if err != nil {
			t.Fatalf("error reading import json file %s: %v", file, err)
		}
		var entries []*Entry
		if err := json.Unmarshal(data, &entries); err != nil {
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
		}
		fmt.Printf("Imported: %s\n", file)
	}

	return srv, consul
}

type Entry struct {
	Key   string `json:"key"`
	Flags uint64 `json:"flags"`
	Value string `json:"value"`
}

func newConsul(consulAddr, consulToken string) (*api.Client, error) {
	conf := api.DefaultConfig()
	conf.Address = consulAddr
	conf.Token = consulToken
	conf.TLSConfig.InsecureSkipVerify = true

	client, err := api.NewClient(conf)
	if err != nil {
		return nil, fmt.Errorf("error creating consul client: %v", err)
	}

	return client, nil
}
