
# Example

*Start the example Consul*

````
git clone https://github.com/sbueringer/consul-datasource.git
cd consul-datasource/example
go run main.go
````

*Configure the Consul datasource*

 1. Open the side menu by clicking the Grafana icon in the top header.
 2. In the side menu in the Configuration section you should find a link named `Data Sources`.
 3. Click the `+ Add data source` button in the top header.
 4. Select `Consul` from the `Type` dropdown.
 5. Fill in the:
    1. datasource name: `consul`
    2. the Consul address: `http://localhost:8500`
    3. the Consul token: `master`
 6. Click the `Save & Test` button
 
*Import the example dashboard*
 
 TODO
 from example/dashboard.json
