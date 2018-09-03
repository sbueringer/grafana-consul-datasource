package main

import (
	"testing"
	"io/ioutil"
	"encoding/json"
	"github.com/hashicorp/consul/testutil"
	"github.com/hashicorp/consul/api"
	"path/filepath"
	"os"
	"path"

	"github.com/sergi/go-diff/diffmatchpatch"
	"bytes"
	"fmt"
	"strings"
)

func TestHandleTable(t *testing.T) {

	srv, consul := setupTest(t)
	defer srv.Stop()

	var tests = []struct {
		query   query
		golden  string
		wantErr string
	}{
		{
			query: query{
				Type: "test",
			},
			wantErr: "ACL support disabled",
		},
		{
			query: query{
				Format: "timeseries",
				Type:   "get",
				Target: "registry/apiregistration.k8s.io/apiservices/v1beta1.rbac.authorization.k8s.io/spec/groupPriorityMinimum",
			},
			golden: "timeseries-get.json",
		},
		{
			query: query{
				Format: "timeseries",
				Type:   "get",
				Target: "registry/apiregistration.k8s.io/apiservices/v1beta1.rbac.authorization.k8s.io/spec/groupPriorityMinimum/",
			},
			golden: "timeseries-get.json",
		},
		{
			query: query{
				Format: "timeseries",
				Type:   "keys",
				Target: "registry/apiregistration.k8s.io/apiservices",
			},
			golden: "timeseries-keys.json",
		},
		{
			query: query{
				Format: "timeseries",
				Type:   "keys",
				Target: "registry/apiregistration.k8s.io/apiservices/",
			},
			golden: "timeseries-keys.json",
		},
		{
			query: query{
				Format: "timeseries",
				Type:   "tags",
				Target: "registry/apiregistration.k8s.io/apiservices/v1.authentication.k8s.io",
			},
			golden: "timeseries-tags.json",
		},
		{
			query: query{
				Format: "timeseries",
				Type:   "tags",
				Target: "registry/apiregistration.k8s.io/apiservices/v1.authentication.k8s.io/",
			},
			golden: "timeseries-tags.json",
		},
		{
			query: query{
				Format: "timeseries",
				Type:   "tagsrec",
				Target: "registry/apiregistration.k8s.io/apiservices/v1.autoscaling",
			},
			golden: "timeseries-tagsrec.json",
		},
		{
			query: query{
				Format: "timeseries",
				Type:   "tagsrec",
				Target: "registry/apiregistration.k8s.io/apiservices/v1.autoscaling/",
			},
			golden: "timeseries-tagsrec.json",
		},
		{
			query: query{
				Format:  "table",
				Target:  "registry/apiregistration.k8s.io/apiservices/*/name",
				Columns: "../name ../kind ../apiVersion ../spec/group ../spec/groupPriorityMinimum ../spec/version ../spec/versionPriority",
			},
			golden: "table.json",
		},
	}

	writeGolden := false

	for _, test := range tests {

		qrs, err := handleQueries(consul, "", []query{test.query})
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
		for _, r := range qrs.Results {
			for _, s := range r.Series {
				for _, p := range s.Points {
					p.Timestamp = 0
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

		if ! (len(diffs) == 1 && diffs[0].Type == diffmatchpatch.DiffEqual) {
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

func setupTest(t *testing.T) (*testutil.TestServer, *api.Client) {
	srv, err := testutil.NewTestServerConfig(func(c *testutil.TestServerConfig) {
		c.LogLevel = "warn"
		c.Stdout = ioutil.Discard
		c.Stderr = ioutil.Discard
	})
	if err != nil {
		t.Fatal(err)
	}

	consul, err := NewConsul(srv.HTTPAddr, srv.Config.ACLMasterToken)
	if err != nil {
		t.Fatalf("could not create consul client: %v", err)
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
