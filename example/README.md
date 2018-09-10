
# Example

## Start Consul

````
git clone https://github.com/sbueringer/grafana-consul-datasource.git
cd consul-datasource/example
go run main.go
````

## Configure the Consul datasource

1. Open Grafana in your browser and open the side menu by clicking the Grafana icon in the top header.
2. In the side menu in the `Configuration` section you should find a link named `Data Sources`.
3. Click the `+ Add data source` button in the top header.
4. Select `Consul` from the `Type` dropdown.
5. Fill in the:
    1. name: `Consul`
    2. Consul address: `http://localhost:8500`
    3. Consul token: `master`
6. Click the `Save & Test` button
 
# Import the example dashboard
 
1. Open the side menu by clicking the Grafana icon in the top header.
2. In the side menu in the `+` section you should find a link named `Import`.
3. Click the `Upload .json File` button and select the dashboard from `example/Consul_Kubernetes_Example.json`.
4. Select `Consul` as datasource and click `Import`.

*For further explanations see [README.md](https://github.com/sbueringer/grafana-consul-datasource/).*
