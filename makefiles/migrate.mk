
MORSE_STATE_EXPORT_PATH ?= "./tools/scripts/migration/morse_state_export.json"
MORSE_ACCOUNT_STATE_PATH ?= "./tools/scripts/migration/morse_account_state.json"
MORSE_POCKET_PROD_RPC_ADDR ?= "https://pocket-rpc.liquify.com/"


######################################
### Migration Authority Operations ###
######################################

.PHONY: export_morse_state
export_morse_state: check_go_version ## Run the migration module export-morse-state subcommand.
# Default to the latest height if no height is provided.
	if [[ -z "$$MORSE_EXPORT_HEIGHT" ]]; then \
		export MORSE_EXPORT_HEIGHT=$$(curl -X POST $$MORSE_POCKET_PROD_RPC_ADDR/v1/query/block -d '{"opts": {"page":1,"per_page":1}}' | jq -r ".block.header.height"); \
	fi; \
	echo Exporting Morse state at height "$$MORSE_EXPORT_HEIGHT" to "$(MORSE_STATE_EXPORT_PATH)"; \
	pocket util export-genesis-for-reset "$$MORSE_EXPORT_HEIGHT" poktroll > "$(MORSE_STATE_EXPORT_PATH)"

.PHONY: collect_morse_accounts
collect_morse_accounts: check_go_version ## Run the migration module collect-morse-accounts subcommand.
	poktrolld tx migration collect-morse-accounts "$(MORSE_STATE_EXPORT_PATH)" "$(MORSE_ACCOUNT_STATE_PATH)"

.PHONY: import_morse_accounts
import_morse_accounts: check_go_version check_from_key_name ## Run the migration module import-morse-accounts subcommand.
	if [[ -z "$$SHANNON_GRPC_ADDR" ]]; then \
		echo "WARNING: SHANNON_GRPC_ADDR environment variable is not set. Defaulting to $(DEFAULT_POCKET_NODE_GRPC_ADDR)"; \
		export SHANNON_GRPC_ADDR=$(DEFAULT_POCKET_NODE_GRPC_ADDR); \
	fi; \
	poktrolld tx migration import-morse-accounts "$(MORSE_ACCOUNT_STATE_PATH)" --from=$(FROM_KEY_NAME) --grpc-addr=$(SHANNON_GRPC_ADDR)

#################################################
### Migration Account/Stake-holder Operations ###
#################################################

.PHONY: claim_morse_account
claim_morse_account: check_go_version check_from_key_name check_morse_private_key_path ## Run the migration module claim-morse-account subcommand.
	poktrolld tx migration claim-account "$(MORSE_PRIVATE_KEY_PATH)" --from="$(FROM_KEY_NAME)"

#########################
### Migration Testing ###
#########################

.PHONY: test_e2e_migration_fixture
test_e2e_migration_fixture: test_e2e_env ## Run only the E2E suite that exercises the migration module using fixture data.
	go test -v ./e2e/tests/... -count=1 -tags=e2e,oneshot --run=MigrationWithFixtureData

.PHONY: test_e2e_migration_snapshot
test_e2e_migration_snapshot: test_e2e_env ## Run only the E2E suite that exercises the migration module using local snapshot data.
	go test -v ./e2e/tests/... -count=1 -tags=e2e,oneshot,manual --run=MigrationWithSnapshotData

#########################
### Migration Helpers ###
#########################

.PHONY: check_from_key_name
## Internal helper to check that the FROM_KEY_NAME environment variable is set
check_from_key_name:
	if [[ -z "$(FROM_KEY_NAME)" ]]; then \
		echo "ERROR: set FROM_KEY_NAME environment variable to the name of the Shannon key to use for claiming"; \
		exit 1; \
	fi

.PHONY: check_morse_private_key_path
## Internal helper to check that the MORSE_PRIVATE_KEY_PATH environment variable is set
check_morse_private_key_path:
	if [[ -z "$(MORSE_PRIVATE_KEY_PATH)" ]]; then \
		echo "ERROR: set MORSE_PRIVATE_KEY_PATH environment variable to the path of the exported private key for the Morse account being claimed"; \
		exit 1; \
	fi
