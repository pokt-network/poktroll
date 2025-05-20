####################
### Applications ###
####################

.PHONY: app_list
app_list: ## List all the staked applications
	pocketd --home=$(POCKETD_HOME) q application list-application --node $(POCKET_NODE)

.PHONY: app_stake
app_stake: ## Stake tokens for the application specified (must specify the APP and SERVICES env vars)
	pocketd --home=$(POCKETD_HOME) tx application stake-application -y --config $(POCKETD_HOME)/config/$(SERVICES) --keyring-backend test --from $(APP) --node $(POCKET_NODE)

.PHONY: app1_stake
app1_stake: ## Stake app1 (also staked in genesis)
	APP=app1 SERVICES=application1_stake_config.yaml make app_stake

.PHONY: app2_stake
app2_stake: ## Stake app2
	APP=app2 SERVICES=application2_stake_config.yaml make app_stake

.PHONY: app3_stake
app3_stake: ## Stake app3
	APP=app3 SERVICES=application3_stake_config.yaml make app_stake

.PHONY: app_unstake
app_unstake: ## Unstake an application (must specify the APP env var)
	pocketd --home=$(POCKETD_HOME) tx application unstake-application -y --keyring-backend test --from $(APP) --node $(POCKET_NODE)

.PHONY: app1_unstake
app1_unstake: ## Unstake app1
	APP=app1 make app_unstake

.PHONY: app2_unstake
app2_unstake: ## Unstake app2
	APP=app2 make app_unstake

.PHONY: app3_unstake
app3_unstake: ## Unstake app3
	APP=app3 make app_unstake

.PHONY: app_delegate
app_delegate: ## Delegate trust to a gateway (must specify the APP and GATEWAY_ADDR env vars). Requires the app to be staked
	pocketd --home=$(POCKETD_HOME) tx application delegate-to-gateway $(GATEWAY_ADDR) --keyring-backend test --from $(APP) --node $(POCKET_NODE)

.PHONY: app1_delegate_gateway1
app1_delegate_gateway1: ## Delegate trust to gateway1
	GATEWAY1=$$(make pocketd_addr ACC_NAME=gateway1) && \
	APP=app1 GATEWAY_ADDR=$$GATEWAY1 make app_delegate

.PHONY: app2_delegate_gateway2
app2_delegate_gateway2: ## Delegate trust to gateway2
	GATEWAY2=$$(make pocketd_addr ACC_NAME=gateway2) && \
	APP=app2 GATEWAY_ADDR=$$GATEWAY2 make app_delegate

.PHONY: app3_delegate_gateway3
app3_delegate_gateway3: ## Delegate trust to gateway3
	GATEWAY3=$$(make pocketd_addr ACC_NAME=gateway3) && \
	APP=app3 GATEWAY_ADDR=$$GATEWAY3 make app_delegate

.PHONY: app_undelegate
app_undelegate: ## Undelegate trust to a gateway (must specify the APP and GATEWAY_ADDR env vars). Requires the app to be staked
	pocketd --home=$(POCKETD_HOME) tx application undelegate-from-gateway $(GATEWAY_ADDR) --keyring-backend test --from $(APP) --node $(POCKET_NODE)

.PHONY: app1_undelegate_gateway1
app1_undelegate_gateway1: ## Undelegate trust to gateway1
	GATEWAY1=$$(make pocketd_addr ACC_NAME=gateway1) && \
	APP=app1 GATEWAY_ADDR=$$GATEWAY1 make app_undelegate

.PHONY: app2_undelegate_gateway2
app2_undelegate_gateway2: ## Undelegate trust to gateway2
	GATEWAY2=$$(make pocketd_addr ACC_NAME=gateway2) && \
	APP=app2 GATEWAY_ADDR=$$GATEWAY2 make app_undelegate

.PHONY: app3_undelegate_gateway3
app3_undelegate_gateway3: ## Undelegate trust to gateway3
	GATEWAY3=$$(make pocketd_addr ACC_NAME=gateway3) && \
	APP=app3 GATEWAY_ADDR=$$GATEWAY3 make app_undelegate
