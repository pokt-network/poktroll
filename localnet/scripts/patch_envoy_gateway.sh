#!/bin/bash
set -euo pipefail

# Envoy Gateway Port Configuration
#
# This script patches the Envoy Gateway LoadBalancer service to expose a static port (30070),
# making it reachable from the local machine on port 3070 as defined in the `kind-config.yaml` file.
#
# Implementation context:
#   - Envoy Gateway service is created as a LoadBalancer 
#   - When running in kind, LoadBalancer services automatically use NodePort underneath
#   - In cloud environments, a LoadBalancer provisioner would map a public IP to this NodePort
#   - In kind environments (without external load balancers), Kubernetes auto-assigns a dynamic 
#     NodePort (30000-32767 range) unless explicitly specified

PATCH_PAYLOAD='[{"op": "replace", "path": "/spec/ports/0/nodePort", "value":30070}]'

# Gets the name of the Envoy Gateway service in the local cluster.
# eg. "envoy-default-guard-envoy-gateway-f82f158a"
get_envoy_gateway_service_name() {
  local envoy_gateway_service_name
  envoy_gateway_service_name=$(kubectl get svc -o json | jq -r '.items[] | select(.metadata.name | startswith("envoy-") and contains("guard-envoy-gateway")) | .metadata.name')
  echo "$envoy_gateway_service_name"
}

# Patches the Envoy Gateway service to enforce a consistent port number of 30070 inside the container.
# eg.  NAME                                     TYPE          CLUSTER-IP    EXTERNAL-IP  PORT(S)         AGE
#      envoy-path-guard-envoy-gateway-55375d27  LoadBalancer  10.96.50.102  <pending>    3070:30070/TCP  3m19s
patch_guard_port() {
    kubectl patch svc "$(get_envoy_gateway_service_name)" --type='json' -p="$PATCH_PAYLOAD"
}

echo "Waiting for Envoy Gateway service..."
while true; do
  svc=$(get_envoy_gateway_service_name)
  if [ -n "$svc" ]; then
    echo "Found Envoy Gateway service: $svc"
    echo "Patching service $svc..."
    patch_guard_port
    exit 0
  fi
  echo "Envoy Gateway service not found, retrying..."
  sleep 2
done