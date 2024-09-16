#####################
### Relay Helpers ###
#####################
.PHONY: send_relay_sovereign_app_JSONRPC
send_relay_sovereign_app_JSONRPC: # Send a JSONRPC relay through the AppGateServer as a sovereign application
	curl -X POST -H "Content-Type: application/json" \
	--data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
	$(APPGATE_SERVER)/anvil

.PHONY: send_relay_delegating_app_JSONRPC
send_relay_delegating_app_JSONRPC: # Send a relay through the gateway as an application that's delegating to this gateway
	@appAddr=$$(poktrolld keys show app1 -a) && \
	curl -X POST -H "Content-Type: application/json" \
	--data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
	$(GATEWAY_URL)/anvil?applicationAddr=$$appAddr

.PHONY: send_relay_sovereign_app_REST
send_relay_sovereign_app_REST: # Send a REST relay through the AppGateServer as a sovereign application
	curl -X POST -H "Content-Type: application/json" \
	--data '{"model": "qwen:0.5b", "stream": false, "messages": [{"role": "user", "content":"count from 1 to 10"}]}' \
	$(APPGATE_SERVER)/ollama/api/chat
