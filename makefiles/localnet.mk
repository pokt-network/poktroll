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

.PHONY: cosmovisor_start_node
cosmovisor_start_node: # Starts the node using cosmovisor that waits for an upgrade plan
	bash tools/scripts/upgrades/cosmovisor-start-node.sh

.PHONY: localnet_cancel_upgrade
localnet_cancel_upgrade: ## Cancels the planed upgrade on local node
	poktrolld tx authz exec tools/scripts/upgrades/authz_cancel_upgrade_tx.json --gas=auto --from=pnf

.PHONY: localnet_show_upgrade_plan
localnet_show_upgrade_plan: ## Shows the upgrade plan on local node
	poktrolld query upgrade plan
