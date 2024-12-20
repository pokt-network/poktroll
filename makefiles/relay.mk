#####################
### Relay Helpers ###
#####################

.PHONY: send_relay_path_JSONRPC
send_relay_path_JSONRPC: test_e2e_env ## Send a JSONRPC relay through PATH to a local anvil (test ETH) node
	curl -X POST -H "Content-Type: application/json" \
		-H "X-App-Address: pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4" \
		--data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
		http://anvil.localhost:3000/v1/
# $(subst http://,http://anvil.,$(PATH_URL))/v1

# TODO_MAINNET(@red-0ne): Re-enable this once PATH Gateway supports REST.
# See https://github.com/buildwithgrove/path/issues/87
.PHONY: send_relay_path_REST
send_relay_path_REST: acc_initialize_pubkeys ## Send a REST relay through PATH to a local ollama (LLM) service
	@echo "Not implemented yet. Check if PATH supports REST relays yet: https://github.com/buildwithgrove/path/issues/87"
# curl -X POST -H "Content-Type: application/json" -H "X-App-Address: pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4" \
# --data '{"model": "qwen:0.5b", "stream": false, "messages": [{"role": "user", "content":"count from 1 to 10"}]}' \
# $(subst http://,http://ollama.,$(PATH_URL))/api/chat

# TODO_MAINNET(@olshansk): Add all the permissionless/delegated/centralized variations once
# the following documentation is ready: https://www.notion.so/buildwithgrove/Different-Modes-of-Operation-PATH-LocalNet-Discussions-122a36edfff6805e9090c9a14f72f3b5?pvs=4#151a36edfff680d681a2dd7f4e5fee55