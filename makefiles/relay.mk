#####################
### Relay Helpers ###
#####################

.PHONY: send_relay_path_JSONRPC
send_relay_path_JSONRPC: # Send a JSONRPC relay through PATH to a local anvil (test ETH) node
	curl -X POST -H "Content-Type: application/json" \
	--data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
	$(PATH_URL)/anvil

.PHONY: send_relay_path_REST
send_relay_path_REST: # Send a REST relay through PATH to a local ollama (LLM) service
	curl -X POST -H "Content-Type: application/json" \
	--data '{"model": "qwen:0.5b", "stream": false, "messages": [{"role": "user", "content":"count from 1 to 10"}]}' \
	$(PATH_URL)/ollama/api/chat

# TODO_MAINNET(@olshansk): Add all the permissionless/delegated/centralized variations once
# the following documentation is ready: https://www.notion.so/buildwithgrove/Different-Modes-of-Operation-PATH-LocalNet-Discussions-122a36edfff6805e9090c9a14f72f3b5?pvs=4#151a36edfff680d681a2dd7f4e5fee55