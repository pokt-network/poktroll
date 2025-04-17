#####################
### Relay Helpers ###
#####################

.PHONY: send_relay_path_JSONRPC
send_relay_path_JSONRPC: test_e2e_env ## Send a JSONRPC relay through PATH to a local anvil (test ETH) node
	curl http://localhost:3070/v1 \
		-H "Authorization: test_api_key" \
		-H "Target-Service-Id: anvil" \
		-H "App-Address: pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4" \
		-d '{"jsonrpc":"2.0","method":"eth_blockNumber","id":1}'

# TODO_MAINNET(@red-0ne): Re-enable this once PATH Gateway supports REST.
# See https://github.com/buildwithgrove/path/issues/87
.PHONY: send_relay_path_REST
send_relay_path_REST: acc_initialize_pubkeys ## Send a REST relay through PATH to a local ollama (LLM) service
	@echo "Not implemented yet. Check if PATH supports REST relays yet: https://github.com/buildwithgrove/path/issues/87"
# curl http://localhost:3070/v1/api/chat \
# 	-H "Authorization: test_api_key" \
# 	-H "Target-Service-Id: ollama" \
# 	-H "App-Address: pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4" \
# 	-d '{"model": "qwen:0.5b", "stream": false, "messages": [{"role": "user", "content":"count from 1 to 10"}]}'

.PHONY: send_relay_path_WEBSOCKET
send_relay_path_WEBSOCKET: test_e2e_env ## Send a WEBSOCKET relay through PATH to a local anvil (test ETH) node
	@echo "Opening WebSocket connection...."
	@echo "After the connection opens, copy & paste this to subscribe to new blocks:"
	@echo '{"id":1,"jsonrpc":"2.0","method":"eth_subscribe","params":["newHeads"]}'
	@echo "You should receive a subscription ID and subsequent block headers"
	wscat -c ws://localhost:3000/v1/ \
		-H "App-Address: pokt1lqyu4v88vp8tzc86eaqr4lq8rwhssyn6rfwzex" \
		-H "Target-Service-Id: anvilws"

# TODO_MAINNET(@olshansk): Add all the permissionless/delegated/centralized variations once
# the following documentation is ready: https://www.notion.so/buildwithgrove/Different-Modes-of-Operation-PATH-LocalNet-Discussions-122a36edfff6805e9090c9a14f72f3b5?pvs=4#151a36edfff680d681a2dd7f4e5fee55