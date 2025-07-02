#############
# Variables #
#############

NETWORK ?= local
POCKET_ACCOUNT ?= pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4 # key name: app1
AGORIC_ACCOUNT ?= agoric1vaj34dfx94y6nvwt57dfyag5gfsp6eqjmvzu8c # key name: foreigner
AGORIC_CHAIN_ID ?= agoriclocal
AXELAR_ACCOUNT ?= axelar1sz7nw80886tuenrhvg2tttlemgfxy734st6f5e # key name: validator
AXELAR_CHAIN_ID ?= axelar
OSMOSIS_ACCOUNT ?= osmo1sz7nw80886tuenrhvg2tttlemgfxy734u7l3f2 # key name: validator
OSMOSIS_CHAIN_ID ?= osmosis



###############################
# Agoric `agd` helper targets #
###############################
.PHONY: agoric_shell
agoric_shell: check_kubectl check_docker_ps check_kind
	bash -c '\
		source ./tools/scripts/ibc-channels.sh && \
		kubectl_exec_grep_pod_interactive agoric-validator "bash" \
	'

.PHONY: axelar_shell
axelar_shell: check_kubectl check_docker_ps check_kind
	bash -c '\
		source ./tools/scripts/ibc-channels.sh && \
		kubectl_exec_grep_pod_interactive axelar-validator "bash" \
	'

.PHONY: osmosis_shell
osmosis_shell: check_kubectl check_docker_ps check_kind
	bash -c '\
		source ./tools/scripts/ibc-channels.sh && \
		kubectl_exec_grep_pod_interactive osmosis-validator "bash" \
	'

.PHONY: ibc_localnet_query_tx
ibc_localnet_query_tx:
	bash -c '\
		source ./tools/scripts/ibc-channels.sh && \
		kubectl_exec_grep_pod $(POD_REGEX) "$(CHAIN_BIN) query tx $(TX_HASH) --chain-id=$(CHAIN_ID)" \
	'

.PHONY: ibc_localnet_query_tx_json
ibc_localnet_query_tx_json:
	bash -c '\
		source ./tools/scripts/ibc-channels.sh && \
		kubectl_exec_grep_pod $(POD_REGEX) "$(CHAIN_BIN) query tx $(TX_HASH) --chain-id=$(CHAIN_ID)" -o json \
	'

.PHONY: agoric_query_tx
agoric_query_tx:
	POD_REGEX="agoric-validator" CHAIN_BIN="agd" CHAIN_ID="agoric" ${MAKE} ibc_localnet_query_tx

.PHONY: agoric_query_tx_json
agoric_query_tx_json:
	POD_REGEX="agoric-validator" CHAIN_BIN="agd" CHAIN_ID="agoriclocal" ${MAKE} ibc_localnet_query_tx

.PHONY: axelar_query_tx
axelar_query_tx:
	POD_REGEX="axelar-validator" CHAIN_BIN="agd" CHAIN_ID="axelar" ${MAKE} ibc_localnet_query_tx

.PHONY: axelar_query_tx_json
axelar_query_tx_json:
	POD_REGEX="axelar-validator" CHAIN_BIN="axelard" CHAIN_ID="axelar" ${MAKE} ibc_localnet_query_tx_json

.PHONY: osmosis_query_tx
osmosis_query_tx:
	POD_REGEX="osmosis-validator" CHAIN_BIN="osmosisd" CHAIN_ID="osmosis" ${MAKE} ibc_localnet_query_tx

.PHONY: osmosis_query_tx_json
osmosis_query_tx_json:
	POD_REGEX="osmosis-validator" CHAIN_BIN="osmosisd" CHAIN_ID="osmosis" ${MAKE} ibc_localnet_query_tx_json



.PHONY: fund_agoric_account
fund_agoric_account: check_kubectl check_docker_ps check_kind
	bash -c '\
		source ./tools/scripts/ibc-channels.sh && \
		kubectl_exec_grep_pod agoric-validator "\
			agd tx bank send validator \
			$(AGORIC_ACCOUNT) \
			1000000000ubld \
			--keyring-backend=test \
			--chain-id=$(AGORIC_CHAIN_ID) \
			--from=validator \
			--yes" \
	'

#####################
# IBC query targets #
#####################
.PHONY: ibc_list_localnet_pocket_clients
ibc_list_pocket_clients:
	pocketd --home=$(POCKETD_HOME) q ibc client states --node=$(POCKET_NODE) --network=$(NETWORK)

.PHONY: ibc_list_localnet_pocket_connections
ibc_list_pocket_connections:
	pocketd --home=$(POCKETD_HOME) q ibc connection connections --node=$(POCKET_NODE) --network=$(NETWORK)

.PHONY: ibc_list_localnet_pocket_channels
ibc_list_pocket_channels:
	pocketd --home=$(POCKETD_HOME) q ibc channel channels --node=$(POCKET_NODE) --network=$(NETWORK)

.PHONY: ibc_list_agoric_clients
ibc_list_agoric_clients:
	bash -c '\
		source ./tools/scripts/ibc-channels.sh && \
		kubectl_exec_grep_pod agoric-validator agd query ibc client states \
	'

.PHONY: ibc_list_agoric_connections
ibc_list_agoric_connections:
	bash -c '\
		source ./tools/scripts/ibc-channels.sh && \
		kubectl_exec_grep_pod agoric-validator agd query ibc connection connections \
	'

.PHONY: ibc_list_agoric_channels
ibc_list_agoric_channels:
	bash -c '\
		source ./tools/scripts/ibc-channels.sh && \
		kubectl_exec_grep_pod agoric-validator agd query ibc channel channels \
	'

.PHONY: ibc_list_axelar_clients
ibc_list_axelar_clients:
	bash -c '\
		source ./tools/scripts/ibc-channels.sh && \
		kubectl_exec_grep_pod axelar-validator axelard query ibc client states \
	'

.PHONY: ibc_list_axelar_connections
ibc_list_axelar_connections:
	bash -c '\
		source ./tools/scripts/ibc-channels.sh && \
		kubectl_exec_grep_pod axelar-validator axelard query ibc connection connections \
	'

.PHONY: ibc_list_axelar_channels
ibc_list_axelar_channels:
	bash -c '\
		source ./tools/scripts/ibc-channels.sh && \
		kubectl_exec_grep_pod axelar-validator axelard query ibc channel channels \
	'

.PHONY: ibc_list_osmosis_clients
ibc_list_osmosis_clients:
	bash -c '\
		source ./tools/scripts/ibc-channels.sh && \
		kubectl_exec_grep_pod osmosis-validator osmosisd query ibc client states \
	'

.PHONY: ibc_list_osmosis_connections
ibc_list_osmosis_connections:
	bash -c '\
		source ./tools/scripts/ibc-channels.sh && \
		kubectl_exec_grep_pod osmosis-validator osmosisd query ibc connection connections \
	'

.PHONY: ibc_list_osmosis_connections
ibc_list_osmosis_channels:
	bash -c '\
		source ./tools/scripts/ibc-channels.sh && \
		kubectl_exec_grep_pod osmosis-validator osmosisd query ibc channel channels \
	'

##########################
# Remote Balance Queries #
##########################
.PHONY: ibc_query_agoric_balance
ibc_query_agoric_balance:
	bash -c '\
		source ./tools/scripts/ibc-channels.sh && \
		kubectl_exec_grep_pod agoric-validator agd query bank balances $(AGORIC_ACCOUNT) \
	'

.PHONY: ibc_query_axelar_balance
ibc_query_axelar_balance:
	bash -c '\
		source ./tools/scripts/ibc-channels.sh && \
		kubectl_exec_grep_pod axelar-validator axelard query bank balances $(AXELAR_ACCOUNT) \
	'

.PHONY: ibc_query_osmosis_balance
ibc_query_osmosis_balance:
	bash -c '\
		source ./tools/scripts/ibc-channels.sh && \
		kubectl_exec_grep_pod osmosis-validator osmosisd query bank balances $(OSMOSIS_ACCOUNT) \
	'

########################
# IBC transfer targets #
########################

## Axelar ##
############
.PHONY: ibc_test_transfer_axelar_to_pocket
ibc_test_transfer_axelar_to_pocket:
	bash -c '\
		source ./tools/scripts/ibc-channels.sh && \
		kubectl_exec_grep_pod axelar-validator \
			axelard tx ibc-transfer transfer transfer \
			$${AXELAR_POCKET_SRC_CHANNEL_ID} $(POCKET_ACCOUNT) 1000uaxl \
			--keyring-backend=test \
			--chain-id=$(AXELAR_CHAIN_ID) \
			--from=validator --yes \
	'

.PHONY: ibc_test_transfer_pocket_to_axelar
ibc_test_transfer_pocket_to_axelar:
	bash -c '\
		source ./tools/scripts/ibc-channels.sh && \
		pocketd --home=$(POCKETD_HOME) tx ibc-transfer transfer transfer \
		$${POCKET_AXELAR_SRC_CHANNEL_ID} $(AXELAR_ACCOUNT) 1000upokt \
			--network $(NETWORK) \
			--keyring-backend=test \
			--from=app1 --yes \
	'

## Agoric ##
############
.PHONY: ibc_test_transfer_agoric_to_pocket
ibc_test_transfer_agoric_to_pocket:
	bash -c '\
		source ./tools/scripts/ibc-channels.sh && \
		kubectl_exec_grep_pod agoric-validator \
			agd tx ibc-transfer transfer transfer \
			$${AGORIC_POCKET_SRC_CHANNEL_ID} $(POCKET_ACCOUNT) 1000ubld \
			--keyring-backend=test \
			--chain-id=$(AGORIC_CHAIN_ID) \
			--from=validator --yes \
	'

.PHONY: ibc_test_transfer_pocket_to_agoric
ibc_test_transfer_pocket_to_agoric:
	bash -c '\
		source ./tools/scripts/ibc-channels.sh && \
		pocketd --home=$(POCKETD_HOME) tx ibc-transfer transfer transfer \
		$${POCKET_AGORIC_SRC_CHANNEL_ID} $(AGORIC_ACCOUNT) 1000upokt \
			--network=$(NETWORK) \
			--keyring-backend=test \
			--from=app1 --yes \
	'
## Osmosis ##
############
.PHONY: ibc_test_transfer_osmosis_to_pocket
ibc_test_transfer_osmosis_to_pocket:
	bash -c '\
		source ./tools/scripts/ibc-channels.sh && \
		kubectl_exec_grep_pod osmosis-validator \
			agd tx ibc-transfer transfer transfer \
			$${OSMOSIS_POCKET_SRC_CHANNEL_ID} $(POCKET_ACCOUNT) 1000ubld \
			--keyring-backend=test \
			--chain-id=$(OSMOSIS_CHAIN_ID) \
			--from=validator --yes \
	'

.PHONY: ibc_test_transfer_pocket_to_osmosis
ibc_test_transfer_pocket_to_osmosis:
	bash -c '\
		source ./tools/scripts/ibc-channels.sh && \
		pocketd --home=$(POCKETD_HOME) tx ibc-transfer transfer transfer \
		$${POCKET_OSMOSIS_SRC_CHANNEL_ID} $(OSMOSIS_ACCOUNT) 1000upokt \
			--network=$(NETWORK) \
			--keyring-backend=test \
			--from=app1 --yes \
	'

#############################
# IBC Restart and Setup    #
#############################

.PHONY: ibc_restart_setup
ibc_restart_setup: ## Restart IBC validators and setup connections using the dynamic restart script
	bash ./tools/scripts/restart-ibc-setup.sh
