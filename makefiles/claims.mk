##############
### Claims ###
##############

# These encoded values were generated using the `encodeSessionHeader` helpers in `query_claim_test.go` as dummy values.
ENCODED_SESSION_HEADER = "eyJhcHBsaWNhdGlvbl9hZGRyZXNzIjoicG9rdDFleXJuNDUwa3JoZnpycmVyemd0djd2c3J4bDA5NDN0dXN4azRhayIsInNlcnZpY2UiOnsiaWQiOiJhbnZpbCIsIm5hbWUiOiIifSwic2Vzc2lvbl9zdGFydF9ibG9ja19oZWlnaHQiOiI1Iiwic2Vzc2lvbl9pZCI6InNlc3Npb25faWQxIiwic2Vzc2lvbl9lbmRfYmxvY2tfaGVpZ2h0IjoiOSJ9"
ENCODED_ROOT_HASH = "cm9vdF9oYXNo"
.PHONY: claim_create_dummy
claim_create_dummy: ## Create a dummy claim by supplier1
	pocketd --home=$(POKTROLLD_HOME) tx supplier create-claim \
	$(ENCODED_SESSION_HEADER) \
	$(ENCODED_ROOT_HASH) \
	--from supplier1 --node $(POCKET_NODE)

.PHONY: claims_list
claim_list: ## List all the claims
	pocketd --home=$(POKTROLLD_HOME) q supplier list-claims --node $(POCKET_NODE)

.PHONY: claims_list_address
claim_list_address: ## List all the claims for a specific address (specified via ADDR variable)
	pocketd --home=$(POKTROLLD_HOME) q supplier list-claims --supplier-operator-address $(ADDR) --node $(POCKET_NODE)

.PHONY: claims_list_address_supplier1
claim_list_address_supplier1: ## List all the claims for supplier1
	SUPPLIER1=$$(make pocketd_addr ACC_NAME=supplier1) && \
	ADDR=$$SUPPLIER1 make claim_list_address

.PHONY: claim_list_height
claim_list_height: ## List all the claims ending at a specific height (specified via HEIGHT variable)
	pocketd --home=$(POKTROLLD_HOME) q supplier list-claims --session-end-height $(HEIGHT) --node $(POCKET_NODE)

.PHONY: claim_list_height_5
claim_list_height_5: ## List all the claims at height 5
	HEIGHT=5 make claim_list_height

.PHONY: claim_list_session
claim_list_session: ## List all the claims ending at a specific session (specified via SESSION variable)
	pocketd --home=$(POKTROLLD_HOME) q supplier list-claims --session-id $(SESSION) --node $(POCKET_NODE)
