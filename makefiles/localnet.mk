########################
### Localnet Helpers ###
########################

.PHONY: localnet_up
localnet_up: check_pocketd check_kubectl check_docker_ps proto_regen localnet_regenesis dev_up check_kind_context warn_message_acc_initialize_pubkeys ## Starts up a clean localnet
	tilt up

.PHONY: localnet_up_quick
localnet_up_quick: check_kubectl check_docker_ps dev_up check_kind_context ## Starts up a localnet without regenerating fixtures
	tilt up

.PHONY: localnet_down
localnet_down: ## Delete resources created by localnet
	tilt down

# DEV_NOTE: the "create namespace" commands in 'dev_up' are here to satisfy the 
# requirements of the `path` helm charts. The requirement for these namespaces 
# to exist may be removed in the future. For reference see repo:
# https://github.com/buildwithgrove/helm-charts/tree/main/charts/path
.PHONY: dev_up
# Internal helper: Spins up Kind cluster if it doesn't already exist
dev_up: check_kind
	@if ! kind get clusters | grep -q "^kind$$"; then \
		echo "[INFO] Creating kind cluster..."; \
		kind create cluster --config ./localnet/kubernetes/kind-config.yaml; \
		kubectl config use-context kind-kind; \
		kubectl create namespace path; \
		kubectl create namespace monitoring; \
		kubectl create namespace middleware; \
	else \
		echo "[INFO] Kind cluster already exists. Skipping creation."; \
	fi

# Optional context for 'move_poktroll_to_pocket' to answer this question:
# https://github.com/pokt-network/poktroll/pull/1151#discussion_r2013801486 
#
# When running 'ignite chain --help', it states:
# > By default the validator node will be initialized in your $HOME directory in a hidden directory that matches the name of your project.
# This DOES NOT reference: chain-id, app-id, or other "logical" things we expect it to be.
# This DOES reference: the project name (i.e. the basename of the directory).
# Until the `poktroll` repository is renamed to `pocket`, the following will be required.
# TODO_TECHDEBT: Once this repository is renamed from `poktroll` to `pocket, remove the helper below.

.PHONY: move_poktroll_to_pocket
# Internal Helper to move the .poktroll directory to .pocket
move_poktroll_to_pocket:
	@echo "###############################################"
	@echo "TODO_MAINNET_MIGRATION(@olshansky): Manually moving HOME/.poktroll to HOME/.pocket. This is a temporary fix until ignite CLI uses the project (not chain) name. Ref: https://docs.ignite.com/nightly/references/cli"
	@echo "Creating new .pocket directory if it doesn't exist..."
	@mkdir -p $(HOME)/.pocket
	@echo "Moving contents from .poktroll to .pocket..."
	@rsync -av --quiet --remove-source-files $(HOME)/.poktroll/ $(HOME)/.pocket/
	@echo "Removing old .poktroll directory..."
	@rm -rf $(HOME)/.poktroll
	@echo "Move completed successfully: .poktroll to .pocket!"
	@echo "###############################################"

.PHONY: localnet_regenesis
localnet_regenesis: check_yq ## Regenerate the localnet genesis file
# NOTE: intentionally not using --home <dir> flag to avoid overwriting the test keyring
	@echo "Initializing chain..."
	@set -e
	@ignite chain init --skip-proto
# DEV_NOTE: We want the following command to run every time localnet is spun up (i.e. localnet re-genesis)
	$(MAKE) move_poktroll_to_pocket
	AUTH_CONTENT=$$(cat ./tools/scripts/authz/dao_genesis_authorizations.json | jq -r tostring); \
	$(SED) -i -E 's!^(\s*)"authorization": (\[\]|null)!\1"authorization": '$$AUTH_CONTENT'!' ${HOME}/.pocket/config/genesis.json;
	@cp -r ${HOME}/.pocket/keyring-test $(POCKETD_HOME)
	@cp -r ${HOME}/.pocket/config $(POCKETD_HOME)/

.PHONY: cosmovisor_start_node
cosmovisor_start_node: ## Starts the node using cosmovisor that waits for an upgrade plan
	bash tools/scripts/upgrades/cosmovisor-start-node.sh

.PHONY: localnet_cancel_upgrade
localnet_cancel_upgrade: ## Cancels the planed upgrade on local node
	pocketd tx authz exec tools/scripts/upgrades/authz_cancel_upgrade_tx.json --gas=auto --from=pnf

.PHONY: localnet_show_upgrade_plan
localnet_show_upgrade_plan: ## Shows the upgrade plan on local node
	pocketd query upgrade plan
