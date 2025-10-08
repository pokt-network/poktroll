#####################
### Relay Helpers ###
#####################

# TODO_MAINNET(@olshansk): Add all the permissionless/delegated/centralized variations once
# the following documentation is ready: https://www.notion.so/buildwithgrove/Different-Modes-of-Operation-PATH-LocalNet-Discussions-122a36edfff6805e9090c9a14f72f3b5?pvs=4#151a36edfff680d681a2dd7f4e5fee55

.PHONY: send_relay_path_JSONRPC
send_relay_path_JSONRPC: check_path_up test_e2e_env ## Send a JSONRPC relay through PATH to a local anvil (test ETH) node
	curl http://localhost:3069/v1 \
		-H "Authorization: test_api_key" \
		-H "Target-Service-Id: anvil" \
		-H "App-Address: pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4" \
		-d '{"jsonrpc":"2.0","method":"eth_blockNumber","id":1}'

.PHONY: send_relay_path_WEBSOCKET
send_relay_path_WEBSOCKET: check_path_up test_e2e_env ## Send a WEBSOCKET relay through PATH to a local anvil (test ETH) node
	@echo "Opening WebSocket connection...."
	@echo "After the connection opens, copy & paste this to subscribe to new blocks:"
	@echo '{"id":1,"jsonrpc":"2.0","method":"eth_subscribe","params":["newHeads"]}'
	@echo "You should receive a subscription ID and subsequent block headers"
	wscat -c ws://localhost:3069/v1/ \
		-H "App-Address: pokt1lqyu4v88vp8tzc86eaqr4lq8rwhssyn6rfwzex" \
		-H "Target-Service-Id: anvilws"


# TODO_POST_MAINNET(@red-0ne): Re-enable this once PATH Gateway supports REST.
# See https://github.com/buildwithgrove/path/issues/87
.PHONY: send_relay_path_REST
send_relay_path_REST: acc_initialize_pubkeys ## Send a REST relay through PATH to a local ollama (LLM) service
	@echo "Not implemented yet. Check if PATH supports REST relays yet: https://github.com/buildwithgrove/path/issues/87"
# curl http://localhost:3070/v1/api/chat \
# 	-H "Authorization: test_api_key" \
# 	-H "Target-Service-Id: ollama" \
# 	-H "App-Address: pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4" \
# 	-d '{"model": "qwen:0.5b", "stream": false, "messages": [{"role": "user", "content":"count from 1 to 10"}]}'


##################################
#### Relay Util Test Requests ####
##################################

.PHONY: test_relay_util_100
test_relay_util_100: check_path_up check_relay_util  ## Test anvil PATH behind GUARD with 10 eth_blockNumber requests using relay-util
	relay-util \
		-u http://localhost:3069/v1 \
		-H "Authorization: test_api_key" \
		-H "Target-Service-Id: anvil" \
		-H "App-Address: pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4" \
		-d '{"jsonrpc":"2.0","method":"eth_blockNumber","id":1}' \
		-x 100 \
		-b

.PHONY: test_baseline_static_server_load
test_baseline_static_server_load: ## Establish baseline load test performance against static nginx chainid server (default: R=100000 T=16 C=5000 D=30s)
	@echo "=== Load Testing Options ==="
	@echo "Parameters:"
	@echo "  R: Requests per second rate (default: 100,000)"
	@echo "  T: Number of threads (default: 16)"
	@echo "  C: Concurrent connections (default: 5,000)"
	@echo "  D: Test duration (default: 30s)"
	@echo ""
	@echo "Examples:"
	@echo "  make test_baseline_static_server_load                              # Default test"
	@echo "  make test_baseline_static_server_load R=10000 C=1000               # Light load"
	@echo "  make test_baseline_static_server_load R=5000 C=500 D=10s           # Quick test"
	@echo "  make test_baseline_static_server_load R=200000 T=32 C=15000 D=45s  # Maximum load"
	@echo ""
	@echo "Running test with: R=$(or $(R),100000) T=$(or $(T),16) C=$(or $(C),5000) D=$(or $(D),30s)"
	@echo "=========================="
	kubectl exec -it deployment/wrk2 -- wrk -R $(or $(R),100000) -L -d $(or $(D),30s) -t $(or $(T),16) -c $(or $(C),5000) http://nginx-chainid/

.PHONY: test_relayminer_only_load
test_relayminer_only_load: ## Generate and run load test against RelayMiner using real RelayRequest data (default: R=512 t=16 c=256 d=300s)
	@echo "=== RelayRequest Load Testing ==="
	@echo "This tool generates proper RelayRequest data and runs load tests against RelayMiner endpoints"
	@echo ""
	@echo "Parameters:"
	@echo "  R: Requests per second rate (default: 512)"
	@echo "  d: Test duration (default: 300s)"
	@echo "  t: Number of threads (default: 16)"
	@echo "  c: Concurrent connections (default: 256)"
	@echo ""
	@echo "Examples:"
	@echo "  make test_relayminer_only_load                          # Default test"
	@echo "  make test_relayminer_only_load R=1000 d=60s             # Higher rate, shorter duration"
	@echo "  make test_relayminer_only_load R=100 t=4 c=50 d=30s     # Light load test"
	@echo "  make test_relayminer_only_load R=2000 t=32 c=1000       # Heavy load test"
	@echo ""
	@echo "Running with: R=$(or $(R),512) d=$(or $(d),300s) t=$(or $(t),16) c=$(or $(c),256)"
	@echo "================================"
	go run tools/scripts/wrk2_relays/main.go \
		-R $(or $(R),512) \
		-d $(or $(d),300s) \
		-t $(or $(t),16) \
		-c $(or $(c),256)