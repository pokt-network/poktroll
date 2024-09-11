########################
### Localnet Helpers ###
########################

.PHONY: localnet_up
localnet_up: check_docker_ps check_kind_context proto_regen localnet_regenesis ## Starts up a clean localnet
	tilt up

.PHONY: localnet_up_quick
localnet_up_quick: check_docker_ps check_kind_context ## Starts up a localnet without regenerating fixtures
	tilt up

.PHONY: localnet_down
localnet_down: ## Delete resources created by localnet
	tilt down

.PHONY: localnet_regenesis
localnet_regenesis: check_yq warn_message_acc_initialize_pubkeys ## Regenerate the localnet genesis file
# NOTE: intentionally not using --home <dir> flag to avoid overwriting the test keyring
	@echo "Initializing chain..."
	@set -e
	@ignite chain init --skip-proto
	AUTH_CONTENT=$$(cat ./tools/scripts/authz/dao_genesis_authorizations.json | jq -r tostring); \
	$(SED) -i -E 's!^(\s*)"authorization": (\[\]|null)!\1"authorization": '$$AUTH_CONTENT'!' ${HOME}/.poktroll/config/genesis.json;

	@cp -r ${HOME}/.poktroll/keyring-test $(POKTROLLD_HOME)
	@cp -r ${HOME}/.poktroll/config $(POKTROLLD_HOME)/

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

.PHONY: cosmovisor_start_node
cosmovisor_start_node: # Starts the node using cosmovisor that waits for an upgrade plan
	bash tools/scripts/upgrades/cosmovisor-start-node.sh

.PHONY: query_tx
query_tx: ## Query for a transaction by hash and output as YAML (default).
	poktrolld --home=$(POKTROLLD_HOME) query tx $(HASH) --node $(POCKET_NODE)

.PHONY: query_tx_json
query_tx_json: ## Query for a transaction by hash and output as JSON.
	poktrolld --home=$(POKTROLLD_HOME) query tx $(HASH) --output json --node $(POCKET_NODE)

.PHONY: query_tx_log
query_tx_log: ## Query for a transaction and print its raw log.
	$(MAKE) -s query_tx_json | jq .raw_log
