#################
### Suppliers ###
#################

.PHONY: supplier_list
supplier_list: ## List all the staked supplier
	poktrolld --home=$(POKTROLLD_HOME) q supplier list-supplier --node $(POCKET_NODE)

.PHONY: supplier_stake
supplier_stake: ## Stake tokens for the supplier specified (must specify the SUPPLIER and SUPPLIER_CONFIG env vars)
	poktrolld --home=$(POKTROLLD_HOME) tx supplier stake-supplier -y --config $(POKTROLLD_HOME)/config/$(SERVICES) --keyring-backend test --from $(SUPPLIER) --node $(POCKET_NODE)

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
	poktrolld --home=$(POKTROLLD_HOME) tx supplier unstake-supplier $(SUPPLIER) --keyring-backend test --from $(SUPPLIER) --node $(POCKET_NODE)

.PHONY: supplier1_unstake
supplier1_unstake: ## Unstake supplier1
	SUPPLIER1=$$(make -s poktrolld_addr ACC_NAME=supplier1) && \
	SUPPLIER=$$SUPPLIER1 make supplier_unstake

.PHONY: supplier2_unstake
supplier2_unstake: ## Unstake supplier2
	SUPPLIER2=$$(make -s poktrolld_addr ACC_NAME=supplier2) && \
	SUPPLIER=$$SUPPLIER2 make supplier_unstake


.PHONY: supplier3_unstake
supplier3_unstake: ## Unstake supplier3
	SUPPLIER3=$$(make -s poktrolld_addr ACC_NAME=supplier3) && \
	SUPPLIER=$$SUPPLIER3 make supplier_unstake
