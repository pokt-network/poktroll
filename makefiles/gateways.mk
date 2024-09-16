####################
###   Gateways   ###
####################

.PHONY: gateway_list
gateway_list: ## List all the staked gateways
	poktrolld --home=$(POKTROLLD_HOME) q gateway list-gateway --node $(POCKET_NODE)

.PHONY: gateway_stake
gateway_stake: ## Stake tokens for the gateway specified (must specify the gateway env var)
	poktrolld --home=$(POKTROLLD_HOME) tx gateway stake-gateway -y --config $(POKTROLLD_HOME)/config/$(STAKE) --keyring-backend test --from $(GATEWAY) --node $(POCKET_NODE)

.PHONY: gateway1_stake
gateway1_stake: ## Stake gateway1
	GATEWAY=gateway1 STAKE=gateway1_stake_config.yaml make gateway_stake

.PHONY: gateway2_stake
gateway2_stake: ## Stake gateway2
	GATEWAY=gateway2 STAKE=gateway2_stake_config.yaml make gateway_stake

.PHONY: gateway3_stake
gateway3_stake: ## Stake gateway3
	GATEWAY=gateway3 STAKE=gateway3_stake_config.yaml make gateway_stake

.PHONY: gateway_unstake
gateway_unstake: ## Unstake an gateway (must specify the GATEWAY env var)
	poktrolld --home=$(POKTROLLD_HOME) tx gateway unstake-gateway -y --keyring-backend test --from $(GATEWAY) --node $(POCKET_NODE)

.PHONY: gateway1_unstake
gateway1_unstake: ## Unstake gateway1
	GATEWAY=gateway1 make gateway_unstake

.PHONY: gateway2_unstake
gateway2_unstake: ## Unstake gateway2
	GATEWAY=gateway2 make gateway_unstake

.PHONY: gateway3_unstake
gateway3_unstake: ## Unstake gateway3
	GATEWAY=gateway3 make gateway_unstake
