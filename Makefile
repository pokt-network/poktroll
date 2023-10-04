POCKETD_HOME := ./localnet/pocketd

.PHONY: localnet_up
localnet_up: ## Starts localnet
	tilt up

.PHONY: localnet_down
localnet_down: ## Delete resources created by localnet
	tilt down
	kubectl delete secret celestia-secret || exit 1

.PHONY: localnet_regenesis
localnet_regenesis:
	# NOTE: intentionally not using --home <dir> flag to avoid overwriting the test keyring
	ignite chain init --skip-proto
	rm -rf $(POCKETD_HOME)/keyring-test
	cp -r ${HOME}/.pocket/keyring-test $(POCKETD_HOME)
	cp ${HOME}/.pocket/config/*_key.json $(POCKETD_HOME)/config/
	cp ${HOME}/.pocket/config/genesis.json ./localnet/