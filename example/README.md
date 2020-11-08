
# Example

## Start Consul

````
git clone https://github.com/sbueringer/grafana-consul-datasource.git
cd grafana-consul-datasource/example
go run main.go
````

## Configure the Consul datasource

1. Open Grafana in your browser and open the side menu by clicking the Grafana icon in the top header.
    ```bash
   docker run --rm -it -p 3000:3000 --name=grafana --net=host  \
   -e "GF_INSTALL_PLUGINS=https://github.com/sbueringer/grafana-consul-datasource/releases/download/v0.1.9/sbueringer-consul-datasource-0.1.9.zip;sbueringer-consul-datasource" grafana/grafana
    ```
1. In the side menu in the `Configuration` section you should find a link named `Data Sources`.
1. Click the `Add data source` button in the top header.
1. Select `Consul`.
1. Fill in the:
    1. name: `Consul`
    2. Consul address: `http://localhost:8500`
    3. Consul token can be left empty: ``
1. Click the `Save & Test` button
 
# Import the example dashboard
 
1. In the side menu in the `+` section you should find a link named `Import`.
1. Click the `Upload .json File` button and select the dashboard from `example/Consul_Kubernetes_Example.json`.
1. Select `Consul` as datasource and click `Import`.

*For further explanations see [README.md](https://github.com/sbueringer/grafana-consul-datasource/).*
