#################
### Suppliers ###
#################

.PHONY: supplier_list
supplier_list: ## List all the staked supplier
	pocketd --home=$(POCKETD_HOME) q supplier list-suppliers --node $(POCKET_NODE)

.PHONY: supplier_list_anvil
supplier_list_anvil: ## List all the staked supplier staked for the anvil service
	pocketd --home=$(POCKETD_HOME) q supplier list-suppliers --service-id anvil --node $(POCKET_NODE)

.PHONY: supplier_show_supplier1
supplier_show_supplier1: ## Show supplier1 details
	pocketd --home=$(POCKETD_HOME) q supplier show-supplier supplier1 --node $(POCKET_NODE)

.PHONY: supplier_stake
supplier_stake: ## Stake tokens for the supplier specified (must specify the SUPPLIER and SUPPLIER_CONFIG env vars)
	pocketd --home=$(POCKETD_HOME) tx supplier stake-supplier -y --config $(POCKETD_HOME)/config/$(SERVICES) --keyring-backend test --from $(SUPPLIER) --node $(POCKET_NODE)

.PHONY: supplier1_stake
supplier1_stake: ## Stake supplier1 (also staked in genesis)
	SUPPLIER=supplier1 SERVICES=supplier1_stake_config.yaml make supplier_stake

.PHONY: supplier2_stake
supplier2_stake: ## Stake supplier2
	SUPPLIER=supplier2 SERVICES=supplier2_stake_config.yaml make supplier_stake

.PHONY: supplier3_stake
supplier3_stake: ## Stake supplier3
	SUPPLIER=supplier3 SERVICES=supplier3_stake_config.yaml make supplier_stake

.PHONY: supplier_unstake
supplier_unstake: ## Unstake an supplier (must specify the SUPPLIER env var)
	pocketd --home=$(POCKETD_HOME) tx supplier unstake-supplier $(SUPPLIER) -y --keyring-backend test --from $(SUPPLIER) --node $(POCKET_NODE)

.PHONY: supplier1_unstake
supplier1_unstake: ## Unstake supplier1
	SUPPLIER1=$$(make -s pocketd_addr ACC_NAME=supplier1) && \
	SUPPLIER=$$SUPPLIER1 make supplier_unstake

.PHONY: supplier2_unstake
supplier2_unstake: ## Unstake supplier2
	SUPPLIER2=$$(make -s pocketd_addr ACC_NAME=supplier2) && \
	SUPPLIER=$$SUPPLIER2 make supplier_unstake


.PHONY: supplier3_unstake
supplier3_unstake: ## Unstake supplier3
	SUPPLIER3=$$(make -s pocketd_addr ACC_NAME=supplier3) && \
	SUPPLIER=$$SUPPLIER3 make supplier_unstake
