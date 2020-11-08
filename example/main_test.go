package main

import (
	"fmt"
	"testing"

	"github.com/hashicorp/consul/api"
)

func TestAnonymous(t *testing.T) {

	client, err := newConsul("http://localhost:8500", "")
	if err != nil {
		panic(err)
	}

	kv, _, err := client.KV().Get("registry/apiregistration.k8s.io/apiservices/v1.apps/apiVersion", &api.QueryOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v", kv)

	status := client.Status()
	fmt.Printf("%v", status)

	leader, err := client.Status().Leader()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s", leader)
}
