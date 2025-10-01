########################################
### pocketd relayminer relay Helpers ###
########################################

.PHONY: pocketd_relayminer_relay_JSONRPC
pocketd_relayminer_relay_JSONRPC: test_e2e_env ## Send a JSONRPC relay through relayminer to a local anvil (test ETH) node
	pocketd relayminer relay \
		--app=pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4 \
		--payload='{"jsonrpc": "2.0", "id": 1, "method": "eth_blockNumber", "params": []}' \
		--home=./localnet/pocketd \
		--network=local \
		--supplier-public-endpoint-override=http://localhost:8085

#####################
### Curl Helpers ###
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

.PHONY: send_relay_path_REST
send_relay_path_REST: check_path_up test_e2e_env ## Send a REST relay through PATH to a local ollama (LLM) service
	curl http://localhost:3069/v1/api/chat \
		-H "Authorization: test_api_key" \
		-H "Target-Service-Id: ollama" \
		-H "App-Address: pokt1pn64d94e6u5g8cllsnhgrl6t96ysnjw59j5gst" \
		-d '{"model": "qwen:0.5b", "stream": false, "messages": [{"role": "user", "content":"count from 1 to 10"}]}'

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