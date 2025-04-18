########################
### Warning Messages ###
########################

.PHONY: warn_message_acc_initialize_pubkeys
warn_message_acc_initialize_pubkeys: ## Print a warning message about the need to run `make acc_initialize_pubkeys`
	@echo "+---------------------------------------------------------------------------------------+"
	@echo "|                                                                                       |"
	@echo "| 🚨 IMPORTANT: Please run the following make command after the network has started: 🚨 |"
	@echo "|                                                                                       |"
	@echo "|     make acc_initialize_pubkeys POCKET_NODE=http://localhost:26657                    |"
	@echo "|                                                                                       |"
	@echo "|     This is required for the following scenarios:                                     |"
	@echo "|       - Running Localnet                                                              |"
	@echo "|       - Running E2E tests                                                             |"
	@echo "|                                                                                       |"
	@echo "|     💡 If you receive the following error response when sending a relay:              |"
	@echo "|                                                                                       |"
	@echo "|     'Failed to receive any response from endpoints. This could be due to              |"
	@echo "|     network issues or high load. Please try again.'                                   |"
	@echo "|                                                                                       |"
	@echo "|     You probably forgot to run 'make acc_initialize_pubkeys'.                         |"
	@echo "+---------------------------------------------------------------------------------------+"

.PHONY: warn_message_local_stress_test
warn_message_local_stress_test: ## Print a warning message when kicking off a local E2E relay stress test
	@echo "+-----------------------------------------------------------------------------------------------+"
	@echo "|                                                                                               |"
	@echo "|     IMPORTANT: Please read the following before continuing with the stress test.              |"
	@echo "|                                                                                               |"
	@echo "|     1. Review the # of suppliers & gateways in 'load-testing/localnet_loadtest_manifest.yaml' |"
	@echo "|     2. Update 'localnet_config.yaml' to reflect what you found in (1)                         |"
	@echo "|     DEVELOPER_TIP: If you're operating off defaults, you'll likely need to update to 3        |"
	@echo "|                                                                                               |"
	@echo "|     TODO_DOCUMENT(@okdas): Move this into proper documentation w/ clearer explanations        |"
	@echo "|                                                                                               |"
	@echo "+-----------------------------------------------------------------------------------------------+"

PHONY: warn_flaky_tests
warn_flaky_tests: ## Print a warning message that some unit tests may be flaky
	@echo "+-----------------------------------------------------------------------------------------------+"
	@echo "|                                                                                               |"
	@echo "|     IMPORTANT: READ ME IF YOUR TESTS FAIL!!!                                                  |"
	@echo "|                                                                                               |"
	@echo "|     1. Our unit / integration tests are far from perfect & some are flaky                     |"
	@echo "|     2. If you ran 'make go_develop_and_test' and a failure occurred, try to run:              |"
	@echo "|     'make test_all' once or twice more                                                        |"
	@echo "|     3. If the same error persists, isolate it with 'go test -v ./path/to/failing/module       |"
	@echo "|                                                                                               |"
	@echo "+-----------------------------------------------------------------------------------------------+"

.PHONY: warn_destructive
warn_destructive: ## Print WARNING to the user
	@echo "This is a destructive action that will affect docker resources outside the scope of this repo!"
