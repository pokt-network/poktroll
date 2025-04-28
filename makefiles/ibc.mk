.PHONY: ibc_list_channels
ibc_list_channels:
	pocketd --home=$(POCKETD_HOME) q ibc channel channels --node $(POCKET_NODE)

.PHONY: ibc_list_connections
ibc_list_connections:
	pocketd --home=$(POCKETD_HOME) q ibc connection connections --node $(POCKET_NODE)

.PHONY: fund-agoric_account
fund_agoric_account:
	# TODO_IN_THIS_COMMIT: log about localnet running and localnet_config ibc...
	kubectl exec $$(kubectl get pods|grep agoric|cut -f 1 -d " ") -- agd tx bank send validator agoric1sz7nw80886tuenrhvg2tttlemgfxy734xcw70w 1000000000ubld --keyring-backend=test --chain-id=agoriclocal --from=validator --yes

.PHONY: agd_shell
agd_shell:
	# TODO_IN_THIS_COMMIT: log about localnet running and localnet_config ibc...
	kubectl exec -it $$(kubectl get pods|grep agoric|cut -f 1 -d " ") -- bash

.PHONY: ibc_test_transfer
ibc_test_transfer:
	kubectl exec -it $$(kubectl get pods|grep agoric|cut -f 1 -d " ") -- agd tx ibc-transfer transfer transfer channel-0 pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4 1000ubld --keyring-backend=test --chain-id=agoriclocal --from=validator --yes

.PHONY: agd_query_tx
agd_query_tx:
	kubectl exec -it $$(kubectl get pods|grep agoric|cut -f 1 -d " ") -- agd query tx $(TX_HASH) --chain-id=agoriclocal

.PHONY: agd_query_tx_json
agd_query_tx_json:
	kubectl exec -it $$(kubectl get pods|grep agoric|cut -f 1 -d " ") -- agd query tx $(TX_HASH) --chain-id=agoriclocal -o json
