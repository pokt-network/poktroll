########################
### Warning Messages ###
########################

.PHONY: warn_message_acc_initialize_pubkeys
warn_message_acc_initialize_pubkeys: ## Print a warning message about the need to run `make acc_initialize_pubkeys`
	@echo "+----------------------------------------------------------------------------------+"
	@echo "|                                                                                  |"
	@echo "|     IMPORTANT: Please run the following command once to initialize               |"
	@echo "|                E2E tests after the network has started:                          |"
	@echo "|                                                                                  |"
	@echo "|     make acc_initialize_pubkeys                                                  |"
	@echo "|                                                                                  |"
	@echo "+----------------------------------------------------------------------------------+"

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

.PHONY: warn_message_grove_helm_charts
warn_message_grove_helm_charts: ## Print a temporary message about using local PATH helm charts while the codebase is in flux.
	@echo "+-----------------------------------------------------------------------------------+"
	@echo "|     TODO_MAINNET_MIGRATION(@olshansky): Remove this check after poktroll & path   |"
	@echo "|     align                                                                         |"
	@echo "|                                                                                   |"
	@echo "|     IMPORTANT: Please run the following commands to set up Grove Helm charts:     |"
	@echo "|                                                                                   |"
	@echo "|     git clone https://github.com/buildwithgrove/helm-charts grove-helm-charts     |"
	@echo "|     cd grove-helm-charts && git checkout d8ac9df02af7a258dfaaa044580ff21f0412cc33 |"
	@echo "|                                                                                   |"
	@echo "|     Then update localnet_config_yaml with:                                        |"
	@echo "|     grove_helm_chart_local_repo:                                                  |"
	@echo "|       enabled: true                                                               |"
	@echo "|       path: ../grove-helm-charts                                                  |"
	@echo "|                                                                                   |"
	@echo "+-----------------------------------------------------------------------------------+"