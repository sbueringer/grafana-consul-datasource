
# Development

Can be build locally via:

```bash
yarn dev && mage -v 
```

Release build can be executed locally via:

```bash
mage -v
./node_modules/.bin/grafana-toolkit plugin:ci-build
./node_modules/.bin/grafana-toolkit plugin:ci-build --finish
./node_modules/.bin/grafana-toolkit plugin:ci-package
ls -la ./ci
```

Github build can be run locally via

```bash
act -s GRAFANA_API_KEY=$GRAFANA_API_KEY
```

