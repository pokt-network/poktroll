
FLAGS = --home=$(POKTROLLD_HOME) --keyring-backend test --from $(PNF_ADDRESS) --node $(POCKET_NODE)
IMPORT_EXEC_JSON = ./tools/scripts/migrate/morse_account_state.json
MORSE_ACCOUNT_STATE_JSON = ./tools/scripts/migrate/morse_account_state.json

.PHONY: migrate_import_morse_accounts
migrate_import_morse_accounts: ## TODO_IN_THIS_COMMIT: Add a description
	poktrolld tx authz exec $(IMPORT_EXEC_JSON) $(MORSE_ACCOUNT_STATE_JSON) $(PARAM_FLAGS)
