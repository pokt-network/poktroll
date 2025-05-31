##############
### Params ###
##############

# TODO_CONSIDERATION: additional factoring (e.g. POCKETD_FLAGS).
PARAM_FLAGS = --home=$(POCKETD_HOME) --keyring-backend test --from $(PNF_ADDRESS) --node $(POCKET_NODE)

##################
### Query All ###
##################

.PHONY: params_query_all
params_query_all: check_jq ## Query the params from all available modules
	@for module in $(MODULES); do \
	    echo "~~~ Querying $$module module params ~~~"; \
	    pocketd query $$module params --node $(POCKET_NODE) --output json | jq; \
	    echo ""; \
	done

##################
### Cosmos Modules ###
####################

.PHONY: params_auth_get_all
params_auth_get_all: ## Get the cosmos auth module params
	pocketd query auth params --node $(POCKET_NODE)

.PHONY: params_bank_get_all
params_bank_get_all: ## Get the cosmos bank module params
	pocketd query bank params --node $(POCKET_NODE)

.PHONY: params_consensus_get_all
params_consensus_get_all: ## Get the cosmos consensus module params
	pocketd query consensus params --node $(POCKET_NODE)

.PHONY: params_crisis_get_all
params_crisis_get_all: ## Get the cosmos crisis module params
	pocketd query crisis params --node $(POCKET_NODE)

.PHONY: params_distribution_get_all
params_distribution_get_all: ## Get the cosmos distribution module params
	pocketd query distribution params --node $(POCKET_NODE)

.PHONY: params_gov_get_all
params_gov_get_all: ## Get the cosmos gov module params
	pocketd query gov params --node $(POCKET_NODE)

.PHONY: params_mint_get_all
params_mint_get_all: ## Get the cosmos mint module params
	pocketd query mint params --node $(POCKET_NODE)

.PHONY: params_protocolpool_get_all
params_protocolpool_get_all: ## Get the cosmos protocolpool module params
	pocketd query protocolpool params --node $(POCKET_NODE)

.PHONY: params_slashing_get_all
params_slashing_get_all: ## Get the cosmos slashing module params
	pocketd query slashing params --node $(POCKET_NODE)

.PHONY: params_staking_get_all
params_staking_get_all: ## Get the cosmos staking module params
	pocketd query staking params --node $(POCKET_NODE)

#########################
### Tokenomics Module ###
#########################

.PHONY: params_tokenomics_get_all
params_tokenomics_get_all: ## Get the tokenomics module params
	pocketd query tokenomics params --node $(POCKET_NODE)

.PHONY: params_tokenomics_update_all
params_tokenomics_update_all: ## Update the tokenomics module params
	pocketd tx authz exec ./tools/scripts/params_templates/tokenomics_0_all.json $(PARAM_FLAGS)

.PHONY: params_tokenomics_update_mint_allocation_percentages
params_tokenomics_update_mint_allocation_percentages: ## Update the tokenomics module mint_allocation_percentages param
	pocketd tx authz exec ./tools/scripts/params_templates/tokenomics_1_mint_allocation_percentages.json $(PARAM_FLAGS)

.PHONY: params_tokenomics_update_dao_reward_address
params_tokenomics_update_dao_reward_address: ## Update the tokenomics module dao_reward_address param
	pocketd tx authz exec ./tools/scripts/params_templates/tokenomics_2_dao_reward_address.json $(PARAM_FLAGS)

.PHONY: params_tokenomics_update_global_inflation_per_claim
params_tokenomics_update_global_inflation_per_claim: ## Update the tokenomics module global_inflation_per_claim param
	pocketd tx authz exec ./tools/scripts/params_templates/tokenomics_3_global_inflation_per_claim.json $(PARAM_FLAGS)

#####################
### Service Module ###
######################

.PHONY: params_service_get_all
params_service_get_all: ## Get the service module params
	pocketd query service params --node $(POCKET_NODE)

.PHONY: params_service_update_all
params_service_update_all: ## Update the service module params
	pocketd tx authz exec ./tools/scripts/params_templates/service_0_all.json $(PARAM_FLAGS)

.PHONY: params_service_update_add_service_fee
params_service_update_add_service_fee: ## Update the service module add_service_fee param
	pocketd tx authz exec ./tools/scripts/params_templates/service_1_add_service_fee.json $(PARAM_FLAGS)

.PHONY: params_service_update_target_num_relays
params_service_update_target_num_relays: ## Update the service module target_num_relays param
	pocketd tx authz exec ./tools/scripts/params_templates/service_2_target_num_relays.json $(PARAM_FLAGS)

####################
### Proof Module ###
###################

.PHONY: params_proof_get_all
params_proof_get_all: ## Get the proof module params
	pocketd query proof params --node $(POCKET_NODE)

.PHONY: params_proof_update_all
params_proof_update_all: ## Update the proof module params
	pocketd tx authz exec ./tools/scripts/params_templates/proof_0_all.json $(PARAM_FLAGS)

.PHONY: params_proof_update_proof_request_probability
params_proof_update_proof_request_probability: ## Update the proof module proof_request_probability param
	pocketd tx authz exec ./tools/scripts/params_templates/proof_1_proof_request_probability.json $(PARAM_FLAGS)

.PHONY: params_proof_update_proof_requirement_threshold
params_proof_update_proof_requirement_threshold: ## Update the proof module proof_requirement_threshold param
	pocketd tx authz exec ./tools/scripts/params_templates/proof_2_proof_requirement_threshold.json $(PARAM_FLAGS)

.PHONY: params_proof_update_proof_missing_penalty
params_proof_update_proof_missing_penalty: ## Update the proof module proof_missing_penalty param
	pocketd tx authz exec ./tools/scripts/params_templates/proof_3_proof_missing_penalty.json $(PARAM_FLAGS)

.PHONY: params_proof_update_proof_submission_fee
params_proof_update_proof_submission_fee: ## Update the proof module proof_submission_fee param
	pocketd tx authz exec ./tools/scripts/params_templates/proof_4_proof_submission_fee.json $(PARAM_FLAGS)

#####################
### Shared Module ###
####################

.PHONY: params_shared_get_all
params_shared_get_all: ## Get the shared module params
	pocketd query shared params --node $(POCKET_NODE)

.PHONY: params_shared_update_all
params_shared_update_all: ## Update the shared module params
	pocketd tx authz exec ./tools/scripts/params_templates/shared_0_all.json $(PARAM_FLAGS)

.PHONY: params_shared_update_num_blocks_per_session
params_shared_update_num_blocks_per_session: ## Update the shared module num_blocks_per_session param
	pocketd tx authz exec ./tools/scripts/params_templates/shared_1_num_blocks_per_session.json $(PARAM_FLAGS)

.PHONY: params_shared_update_grace_period_end_offset_blocks
params_shared_update_grace_period_end_offset_blocks: ## Update the shared module grace_period_end_offset_blocks param
	pocketd tx authz exec ./tools/scripts/params_templates/shared_2_grace_period_end_offset_blocks.json $(PARAM_FLAGS)

.PHONY: params_shared_update_claim_window_open_offset_blocks
params_shared_update_claim_window_open_offset_blocks: ## Update the shared module claim_window_open_offset_blocks param
	pocketd tx authz exec ./tools/scripts/params_templates/shared_3_claim_window_open_offset_blocks.json $(PARAM_FLAGS)

.PHONY: params_shared_update_claim_window_close_offset_blocks
params_shared_update_claim_window_close_offset_blocks: ## Update the shared module claim_window_close_offset_blocks param
	pocketd tx authz exec ./tools/scripts/params_templates/shared_4_claim_window_close_offset_blocks.json $(PARAM_FLAGS)

.PHONY: params_shared_update_proof_window_open_offset_blocks
params_shared_update_proof_window_open_offset_blocks: ## Update the shared module proof_window_open_offset_blocks param
	pocketd tx authz exec ./tools/scripts/params_templates/shared_5_proof_window_open_offset_blocks.json $(PARAM_FLAGS)

.PHONY: params_shared_update_proof_window_close_offset_blocks
params_shared_update_proof_window_close_offset_blocks: ## Update the shared module proof_window_close_offset_blocks param
	pocketd tx authz exec ./tools/scripts/params_templates/shared_6_proof_window_close_offset_blocks.json $(PARAM_FLAGS)

.PHONY: params_shared_update_supplier_unbonding_period_sessions
params_shared_update_supplier_unbonding_period_sessions: ## Update the shared module supplier_unbonding_period_sessions param
	pocketd tx authz exec ./tools/scripts/params_templates/shared_7_supplier_unbonding_period_sessions.json $(PARAM_FLAGS)

.PHONY: params_shared_update_application_unbonding_period_sessions
params_shared_update_application_unbonding_period_sessions: ## Update the shared module application_unbonding_period_sessions param
	pocketd tx authz exec ./tools/scripts/params_templates/shared_8_application_unbonding_period_sessions.json $(PARAM_FLAGS)

.PHONY: params_shared_update_compute_units_to_tokens_multiplier
params_shared_update_compute_units_to_tokens_multiplier: ## Update the shared module compute_units_to_tokens_multiplier param
	pocketd tx authz exec ./tools/scripts/params_templates/shared_9_compute_units_to_tokens_multiplier.json $(PARAM_FLAGS)

.PHONY: params_shared_update_gateway_unbonding_period_sessions
params_shared_update_gateway_unbonding_period_sessions: ## Update the shared module gateway_unbonding_period_sessions param
	pocketd tx authz exec ./tools/scripts/params_templates/shared_10_gateway_unbonding_period_sessions.json $(PARAM_FLAGS)

.PHONY: params_shared_update_compute_unit_cost_granularity
params_shared_update_compute_unit_cost_granularity: ## Update the shared module compute_unit_cost_granularity param
	pocketd tx authz exec ./tools/scripts/params_templates/shared_11_compute_unit_cost_granularity.json $(PARAM_FLAGS)

####################
### Gateway Module ###
####################

.PHONY: params_gateway_get_all
params_gateway_get_all: ## Get the gateway module params
	pocketd query gateway params --node $(POCKET_NODE)

.PHONY: params_gateway_update_all
params_gateway_update_all: ## Update the gateway module params
	pocketd tx authz exec ./tools/scripts/params_templates/gateway_0_all.json $(PARAM_FLAGS)

.PHONY: params_gateway_update_min_stake
params_gateway_update_min_stake: ## Update the gateway module min_stake param
	pocketd tx authz exec ./tools/scripts/params_templates/gateway_1_min_stake.json $(PARAM_FLAGS)

########################
### Application Module ###
########################

.PHONY: params_application_get_all
params_application_get_all: ## Get the application module params
	pocketd query application params --node $(POCKET_NODE)

.PHONY: params_application_update_all
params_application_update_all: ## Update the application module params
	pocketd tx authz exec ./tools/scripts/params_templates/application_0_all.json $(PARAM_FLAGS)

.PHONY: params_application_update_max_delegated_gateways
params_application_update_max_delegated_gateways: ## Update the application module max_delegated_gateways param
	pocketd tx authz exec ./tools/scripts/params_templates/application_1_max_delegated_gateways.json $(PARAM_FLAGS)

.PHONY: params_application_update_min_stake
params_application_update_min_stake: ## Update the application module min_stake param
	pocketd tx authz exec ./tools/scripts/params_templates/application_2_min_stake.json $(PARAM_FLAGS)

#####################
### Supplier Module ###
#####################

.PHONY: params_supplier_get_all
params_supplier_get_all: ## Get the supplier module params
	pocketd query supplier params --node $(POCKET_NODE)

.PHONY: params_supplier_update_all
params_supplier_update_all: ## Update the supplier module params
	pocketd tx authz exec ./tools/scripts/params_templates/supplier_0_all.json $(PARAM_FLAGS)

.PHONY: params_supplier_update_min_stake
params_supplier_update_min_stake: ## Update the supplier module min_stake param
	pocketd tx authz exec ./tools/scripts/params_templates/supplier_1_min_stake.json $(PARAM_FLAGS)

.PHONY: params_supplier_update_staking_fee
params_supplier_update_staking_fee: ## Update the supplier module staking_fee param
	pocketd tx authz exec ./tools/scripts/params_templates/supplier_2_staking_fee.json $(PARAM_FLAGS)

####################
### Session Module ###
####################

.PHONY: params_session_get_all
params_session_get_all: ## Get the session module params
	pocketd query session params --node $(POCKET_NODE)

.PHONY: params_session_update_all
params_session_update_all: ## Update the session module params
	pocketd tx authz exec ./tools/scripts/params_templates/session_0_all.json $(PARAM_FLAGS)

.PHONY: params_session_update_num_suppliers_per_session
params_session_update_num_suppliers_per_session: ## Update the session module num_suppliers_per_session param
	pocketd tx authz exec ./tools/scripts/params_templates/session_1_num_suppliers_per_session.json $(PARAM_FLAGS)

####################
### Migration Module ###
####################

.PHONY: params_migration_get_all
params_migration_get_all: ## Get the migration module params
	pocketd query migration params --node $(POCKET_NODE)

.PHONY: params_migration_update_all
params_migration_update_all: ## Update the migration module params
	pocketd tx authz exec ./tools/scripts/params_templates/migration_0_all.json $(PARAM_FLAGS)

####################
### Consensus Module ###
####################

.PHONY: params_consensus_update_block_size_6mb
params_consensus_update_block_size_6mb: ## Update consensus block size to 6MB
	pocketd tx authz exec ./tools/scripts/params_templates/consensus_block_size_6mb.json $(PARAM_FLAGS)