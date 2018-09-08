
# Grafana datasource for Consul

[![Travis](https://img.shields.io/travis/sbueringer/consul-datasource.svg)](https://travis-ci.org/sbueringer/consul-datasource)[![Codecov](https://img.shields.io/codecov/c/github/sbueringer/consul-datasource.svg)](https://codecov.io/gh/sbueringer/consul-datasource)[![CodeFactor](https://www.codefactor.io/repository/github/sbueringer/consul-datasource/badge)](https://www.codefactor.io/repository/github/sbueringer/consul-datasource)[![GoReportCard](https://goreportcard.com/badge/github.com/sbueringer/consul-datasource?style=plastic)](https://goreportcard.com/report/github.com/sbueringer/consul-datasource)![GitHub release](https://img.shields.io/github/release/sbueringer/consul-datasource.svg)

[![contributions welcome](https://img.shields.io/badge/contributions-welcome-brightgreen.svg?style=flat)](https://github.com/dwyl/esta/issues) 

This datasource lets you integrate key value data from Consul in Grafana dashboards.

![Overview](https://github.com/sbueringer/consul-datasource/docs/overview.png)

# Usage

The data can be used in **table** and **single-stat** panels. The following explanations are based on the example available in the [example folder](https://github.com/sbueringer/consul-datasource/example/README.md).

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


