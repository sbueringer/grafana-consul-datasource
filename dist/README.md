
[![Codecov](https://img.shields.io/codecov/c/github/codecov/example-python.svg)](https://codecov.io/gh/sbueringer/consul-datasource)[![GoReportCard](https://goreportcard.com/badge/github.com/sbueringer/consul-datasource?style=plastic)](https://goreportcard.com/report/github.com/sbueringer/consul-datasource)

# Grafana datasource for Consul 

This datasource lets you integrate key value data from Consul in Grafana dashboards.

[TODO add screenshot]

# Usage

The data can be used in **table** and **single-stat** panels. The following examples are based on the 
example server & data available in the [example folder](https://github.com/sbueringer/consul-datasource/example/README.md).

## Adding the datasource

1. Open the side menu by clicking the Grafana icon in the top header.
2. In the side menu in the Configuration section you should find a link named `Data Sources`.
3. Click the `+ Add data source` button in the top header.
4. Select Consul from the `Type` dropdown.
5. Fill in the datasource name, the Consul address and the Consul token
6. Click the `Save & Test` button

## Panels

### Single Stat Panel

TODO all 4 types... (if possible)

### Table Panel

TODO


TODO Breaking:
* , comma separated
* plugin id

TODO PR:
* Ask because of plugin-id best-practise