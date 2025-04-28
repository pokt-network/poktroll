###############################
# Agoric `agd` helper targets #
###############################
.PHONY: agd_shell
agd_shell: check_kubectl check_docker_ps check_kind
	kubectl exec -it $$(kubectl get pods|grep agoric|cut -f 1 -d " ") -- bash

.PHONY: agd_query_tx
agd_query_tx:
	kubectl exec -it $$(kubectl get pods|grep agoric|cut -f 1 -d " ") -- agd query tx $(TX_HASH) --chain-id=agoriclocal

.PHONY: agd_query_tx_json
agd_query_tx_json:
	kubectl exec -it $$(kubectl get pods|grep agoric|cut -f 1 -d " ") -- agd query tx $(TX_HASH) --chain-id=agoriclocal -o json

.PHONY: fund_agoric_account
fund_agoric_account: check_kubectl check_docker_ps check_kind
	kubectl exec $$(kubectl get pods|grep agoric|cut -f 1 -d " ") -- agd tx bank send validator agoric1vaj34dfx94y6nvwt57dfyag5gfsp6eqjmvzu8c 1000000000ubld --keyring-backend=test --chain-id=agoriclocal --from=validator --yes

#####################
# IBC query targets #
#####################
.PHONY: ibc_list_channels
ibc_list_channels:
	pocketd --home=$(POCKETD_HOME) q ibc channel channels --node $(POCKET_NODE)

.PHONY: ibc_list_connections
ibc_list_connections:
	pocketd --home=$(POCKETD_HOME) q ibc connection connections --node $(POCKET_NODE)

########################
# IBC transfer targets #
########################
.PHONY: ibc_test_transfer_agoric_to_pocket
ibc_test_transfer:
	kubectl exec -it $$(kubectl get pods|grep agoric|cut -f 1 -d " ") -- agd tx ibc-transfer transfer transfer channel-0 pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4 1000ubld --keyring-backend=test --chain-id=agoriclocal --from=validator --yes

.PHONY: ibc_test_transfer_pocket_to_agoric
ibc_test_transfer:
	pocketd --home=$(POCKETD_HOME) tx ibc-transfer transfer transfer channel-0 agoric1vaj34dfx94y6nvwt57dfyag5gfsp6eqjmvzu8c 1000upokt --node $(POCKET_NODE) --keyring-backend=test --yes
