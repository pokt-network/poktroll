###############
### Session ###
###############

.PHONY: get_session
get_session: ## Retrieve the session given the following env vars: (APP_ADDR, SVC, HEIGHT)
	pocketd --home=$(POKTROLLD_HOME) q session get-session $(APP) $(SVC) $(HEIGHT) --node $(POCKET_NODE)

.PHONY: get_session_app1_anvil
get_session_app1_anvil: ## Retrieve the session for (app1, anvil, latest_height)
	APP1=$$(make pocketd_addr ACC_NAME=app1) && \
	APP=$$APP1 SVC=anvil HEIGHT=0 make get_session

.PHONY: get_session_app2_anvil
get_session_app2_anvil: ## Retrieve the session for (app2, anvil, latest_height)
	APP2=$$(make pocketd_addr ACC_NAME=app2) && \
	APP=$$APP2 SVC=anvil HEIGHT=0 make get_session

.PHONY: get_session_app3_anvil
get_session_app3_anvil: ## Retrieve the session for (app3, anvil, latest_height)
	APP3=$$(make pocketd_addr ACC_NAME=app3) && \
	APP=$$APP3 SVC=anvil HEIGHT=0 make get_session
