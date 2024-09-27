##############
### Params ###
##############

# TODO_CONSIDERATION: additional factoring (e.g. POKTROLLD_FLAGS).
PARAM_FLAGS = --home=$(POKTROLLD_HOME) --keyring-backend test --from $(PNF_ADDRESS) --node $(POCKET_NODE)

### Tokenomics Module Params ###
.PHONY: params_get_tokenomics
params_get_tokenomics: ## Get the tokenomics module params
	poktrolld query tokenomics params --node $(POCKET_NODE)

.PHONY: update_tokenomics_params_all
params_update_tokenomics_all: ## Update the tokenomics module params
	poktrolld tx authz exec ./tools/scripts/params/tokenomics_all.json $(PARAM_FLAGS)

.PHONY: params_update_tokenomics_compute_units_to_tokens_multiplier
params_update_tokenomics_compute_units_to_tokens_multiplier: ## Update the tokenomics module compute_units_to_tokens_multiplier param
	poktrolld tx authz exec ./tools/scripts/params/tokenomics_compute_units_to_tokens_multiplier.json $(PARAM_FLAGS)

### Service Module Params ###
.PHONY: params_get_service
params_get_service: ## Get the service module params
	poktrolld query service params --node $(POCKET_NODE)

.PHONY: params_update_service_all
params_update_service_all: ## Update the service module params
	poktrolld tx authz exec ./tools/scripts/params/service_all.json $(PARAM_FLAGS)

.PHONY: params_update_service_add_service_fee
params_update_service_add_service_fee: ## Update the service module add_service_fee param
	poktrolld tx authz exec ./tools/scripts/params/service_add_service_fee.json $(PARAM_FLAGS)

### Proof Module Params ###
.PHONY: params_get_proof
params_get_proof: ## Get the proof module params
	poktrolld query proof params --node $(POCKET_NODE)

.PHONY: params_update_proof_all
params_update_proof_all: ## Update the proof module params
	poktrolld tx authz exec ./tools/scripts/params/proof_all.json $(PARAM_FLAGS)

.PHONY: params_update_proof_min_relay_difficulty_bits
params_update_proof_min_relay_difficulty_bits: ## Update the proof module min_relay_difficulty_bits param
	poktrolld tx authz exec ./tools/scripts/params/proof_min_relay_difficulty_bits.json $(PARAM_FLAGS)

.PHONY: params_update_proof_proof_request_probability
params_update_proof_proof_request_probability: ## Update the proof module proof_request_probability param
	poktrolld tx authz exec ./tools/scripts/params/proof_proof_request_probability.json $(PARAM_FLAGS)

.PHONY: params_update_proof_proof_requirement_threshold
params_update_proof_proof_requirement_threshold: ## Update the proof module proof_requirement_threshold param
	poktrolld tx authz exec ./tools/scripts/params/proof_proof_requirement_threshold.json $(PARAM_FLAGS)

.PHONY: params_update_proof_proof_missing_penalty
params_update_proof_proof_missing_penalty: ## Update the proof module proof_missing_penalty param
	poktrolld tx authz exec ./tools/scripts/params/proof_proof_missing_penalty.json $(PARAM_FLAGS)

.PHONY: params_update_proof_proof_submission_fee
params_update_proof_proof_submission_fee: ## Update the proof module proof_submission_fee param
	poktrolld tx authz exec ./tools/scripts/params/proof_proof_submission_fee.json $(PARAM_FLAGS)

### Shared Module Params ###
.PHONY: params_get_shared
params_get_shared: ## Get the shared module params
	poktrolld query shared params --node $(POCKET_NODE)

.PHONY: params_update_shared_all
params_update_shared_all: ## Update the session module params
	poktrolld tx authz exec ./tools/scripts/params/shared_all.json $(PARAM_FLAGS)

.PHONY: params_update_shared_num_blocks_per_session
params_update_shared_num_blocks_per_session: ## Update the shared module num_blocks_per_session param
	poktrolld tx authz exec ./tools/scripts/params/shared_num_blocks_per_session.json $(PARAM_FLAGS)

.PHONY: params_update_shared_grace_period_end_offset_blocks
params_update_shared_grace_period_end_offset_blocks: ## Update the shared module grace_period_end_offset_blocks param
	poktrolld tx authz exec ./tools/scripts/params/shared_grace_period_end_offset_blocks.json $(PARAM_FLAGS)

.PHONY: params_update_shared_claim_window_open_offset_blocks
params_update_shared_claim_window_open_offset_blocks: ## Update the shared module claim_window_open_offset_blocks param
	poktrolld tx authz exec ./tools/scripts/params/shared_claim_window_open_offset_blocks.json $(PARAM_FLAGS)

.PHONY: params_update_shared_claim_window_close_offset_blocks
params_update_shared_claim_window_close_offset_blocks: ## Update the shared module claim_window_close_offset_blocks param
	poktrolld tx authz exec ./tools/scripts/params/shared_claim_window_close_offset_blocks.json $(PARAM_FLAGS)

.PHONY: params_update_shared_proof_window_open_offset_blocks
params_update_shared_proof_window_open_offset_blocks: ## Update the shared module proof_window_open_offset_blocks param
	poktrolld tx authz exec ./tools/scripts/params/shared_proof_window_open_offset_blocks.json $(PARAM_FLAGS)

.PHONY: params_update_shared_proof_window_close_offset_blocks
params_update_shared_proof_window_close_offset_blocks: ## Update the shared module proof_window_close_offset_blocks param
	poktrolld tx authz exec ./tools/scripts/params/shared_proof_window_close_offset_blocks.json $(PARAM_FLAGS)

.PHONY: params_update_shared_compute_units_to_tokens_multiplier
params_update_shared_compute_units_to_tokens_multiplier: ## Update the shared module compute_units_to_tokens_multiplier param
	poktrolld tx authz exec ./tools/scripts/params/shared_compute_units_to_tokens_multiplier.json $(PARAM_FLAGS)

### Gateway Module Params ###
.PHONY: params_get_gateway
params_get_gateway: ## Get the gateway module params
	poktrolld query gateway params --node $(POCKET_NODE)

.PHONY: params_update_gateway_all
params_update_gateway_all: ## Update the session module params
	poktrolld tx authz exec ./tools/scripts/params/gateway_all.json $(PARAM_FLAGS)

.PHONY: params_update_gateway_min_stake
params_update_gateway_min_stake: ## Update the gateway module min_stake param
	poktrolld tx authz exec ./tools/scripts/params/gateway_min_stake.json $(PARAM_FLAGS)

### Application Module Params ###
.PHONY: params_get_application
params_get_application: ## Get the application module params
	poktrolld query application params --node $(POCKET_NODE)

.PHONY: params_update_application_all
params_update_application_all: ## Update the application module params
	poktrolld tx authz exec ./tools/scripts/params/application_all.json $(PARAM_FLAGS)

.PHONY: params_update_application_max_delegated_gateways
params_update_application_max_delegated_gateways: ## Update the application module max_delegated_gateways param
	poktrolld tx authz exec ./tools/scripts/params/application_max_delegated_gateways.json $(PARAM_FLAGS)

.PHONY: params_update_application_min_stake
params_update_application_min_stake: ## Update the application module min_stake param
	poktrolld tx authz exec ./tools/scripts/params/application_min_stake.json $(PARAM_FLAGS)

.PHONY: params_query_all
params_query_all: check_jq ## Query the params from all available modules
	@for module in $(MODULES); do \
	    echo "~~~ Querying $$module module params ~~~"; \
	    poktrolld query $$module params --node $(POCKET_NODE) --output json | jq; \
	    echo ""; \
	done
