package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/sdk/testutil"
)

func main() {
	fmt.Println("Starting Consul")
	srv := startServer()
	defer srv.Stop()
	fmt.Printf("Consul is now listening on: %s\n", srv.HTTPAddr)

	var wait sync.WaitGroup
	wait.Add(1)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	go func() {
		<-sigChan
		fmt.Println("Stopping Consul")
		wait.Done()
	}()

	fmt.Println("Server can be stopped with CTRL-c")
	wait.Wait()
	fmt.Println("Consul stopped")
}

func startServer() *testutil.TestServer {
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
		panic(err)
	}

	fmt.Println("Importing example data to Consul")

	consul, err := newConsul(srv.HTTPAddr, srv.Config.ACLMasterToken)
	if err != nil {
		panic(fmt.Errorf("could not create consul client: %v", err))
	}

	files, err := filepath.Glob("data/data.json")
	if err != nil {
		panic(fmt.Errorf("error getting import json files"))
	}

	if len(files) == 0 {
		files, err = filepath.Glob("example/data/data.json")
		if err != nil {
			panic(fmt.Errorf("error getting import json files"))
		}
	}

	for _, file := range files {
		fmt.Printf("Importing: %s\n", file)

		data, err := ioutil.ReadFile(file)
		if err != nil {
			panic(fmt.Errorf("error reading import json file %s: %v", file, err))
		}
		var entries []*entry
		if err := json.Unmarshal((data), &entries); err != nil {
			panic(fmt.Errorf("cannot unmarshal data: %s", err))
		}

		for _, entry := range entries {
			pair := &api.KVPair{
				Key:   entry.Key,
				Flags: entry.Flags,
				Value: []byte(entry.Value),
			}
			if _, err := consul.KV().Put(pair, nil); err != nil {
				panic(fmt.Errorf("error! Failed writing data for key %s: %s", pair.Key, err))
			}
		}
		fmt.Printf("Imported: %s\n", file)
	}

	return srv
}

type entry struct {
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
