
# Grafana datasource for Consul

[![Travis](https://img.shields.io/travis/sbueringer/consul-datasource.svg)](https://travis-ci.org/sbueringer/consul-datasource)[![Codecov](https://img.shields.io/codecov/c/github/sbueringer/consul-datasource.svg)](https://codecov.io/gh/sbueringer/consul-datasource)[![CodeFactor](https://www.codefactor.io/repository/github/sbueringer/consul-datasource/badge)](https://www.codefactor.io/repository/github/sbueringer/consul-datasource)[![GoReportCard](https://goreportcard.com/badge/github.com/sbueringer/consul-datasource?style=plastic)](https://goreportcard.com/report/github.com/sbueringer/consul-datasource)![GitHub release](https://img.shields.io/github/release/sbueringer/consul-datasource.svg)

[![contributions welcome](https://img.shields.io/badge/contributions-welcome-brightgreen.svg?style=flat)](https://github.com/dwyl/esta/issues) 

This datasource lets you integrate key value data from Consul in Grafana dashboards.

![Overview](https://github.com/sbueringer/consul-datasource/raw/master/docs/overview.png)

# Usage

The data can be used in **table** and **single-stat** panels. The following explanations are based on the example available in the [example folder](https://github.com/sbueringer/consul-datasource/tree/master/example).

## Adding the datasource

1. Open the side menu by clicking the Grafana icon in the top header.
2. In the side menu in the Configuration section you should find a link named `Data Sources`.
3. Click the `+ Add data source` button in the top header.
4. Select Consul from the `Type` dropdown.
5. Fill in the datasource name, the Consul address and the Consul token
6. Click the `Save & Test` button

## Features

* Consul keys can be used as Dashboard variable values
* Numeric Consul keys can be retrieved directly and displayed in Singlestat panels
* Consul key/value pairs can be retrieved via Timeseries tags and displayed in Singlestat panels
* Consul key/value pairs can be displayed in Table panels.

## Examples

### Variables

![Variables](https://github.com/sbueringer/consul-datasource/raw/master/docs/keys.png)

This example shows how keys can be queried to use them as variables. This query retrieves all direct subkeys of `registry/apiregistration.k8s.io/apiservices/`. The subkeys are then matched via the regex and can then be used as variable values.

### Singlestat Panel

![Tags](https://github.com/sbueringer/consul-datasource/raw/master/docs/tags.png)

This example shows how subkeys can be retrieved as tags. These tags can then be displayed in the Single Stat panel by defining a legend format. *Note*: This only works if `Value / Stat` in the `Option` tab is set to `Name`.

### Table Panel

![Table](https://github.com/sbueringer/consul-datasource/raw/master/docs/table.png)

The final examples shows how key/value pairs can be displayed in tables. Every matching key of the query results in one row. Columns can then be retrieved relative from this key. 
