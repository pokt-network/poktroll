# Localnet Load Testing

## Infra Prep

1. Pull poktroll repo
2. Pull path repo
3. Pull path helm chart repo
4. Start localnet
5. Update `localnet_config.yaml` to reflect `path_local_repo` to true
6. Update `localnet_config.yaml` to reflect `grove_helm_chart_local_repo` to true
7. Set `defaultDashboardsEnabled` to true in `localnet/kubernetes/prometheus/prometheus.yaml`

## Running the test

```bash
make localnet_up
make acc_initialize_pubkeys
make test_load_relays_stress_localnet_single_supplier

go test -v ./load-testing/... -tags=e2e,test,load --features-path=tests/relays_stress_single_supplier.feature
```

## Check params

```bash
pocketd q shared params
```

Check blocks:

```
pocketd q block -o json | tail -n 1 | jq '.header.height'
```

```bash
faucet:
  enabled: true
grove_helm_chart_local_repo:
  enabled: true
  path: ../grove-helm-charts
helm_chart_local_repo:
  enabled: false
  path: ../helm-charts
hot-reloading: true
indexer:
  clone_if_not_present: false
  enabled: false
  repo_path: ../pocketdex
observability:
  enabled: true
  grafana:
    defaultDashboardsEnabled: true
ollama:
  enabled: false
  model: qwen:0.5b
path_gateways:
  count: 1
path_local_repo:
  enabled: true
  path: ../path
relayminers:
  count: 1
  delve:
    enabled: false
  logs:
    level: debug
rest:
  enabled: true
validator:
  cleanupBeforeEachStart: true
  delve:
    enabled: false
  logs:
    format: json
    level: info
```
