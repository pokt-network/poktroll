.SILENT:

SHELL = /bin/sh
POKTROLLD_HOME ?= ./localnet/poktrolld
POCKET_NODE ?= tcp://127.0.0.1:36657 # The pocket rollup node (full node and sequencer in the localnet context)
APPGATE_SERVER ?= http://localhost:42069
POCKET_ADDR_PREFIX = pokt

####################
### Dependencies ###
####################

# TODO: Add other dependencies (ignite, docker, k8s, etc) here
.PHONY: install_ci_deps
install_ci_deps: ## Installs `mockgen` and other go tools
	go install "github.com/golang/mock/mockgen@v1.6.0" && mockgen --version
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && golangci-lint --version
	go install golang.org/x/tools/cmd/goimports@latest

########################
### Makefile Helpers ###
########################

.PHONY: prompt_user
# Internal helper target - prompt the user before continuing
prompt_user:
	@echo "Are you sure? [y/N] " && read ans && [ $${ans:-N} = y ]

.PHONY: list
list: ## List all make targets
	@${MAKE} -pRrn : -f $(MAKEFILE_LIST) 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | sort

.PHONY: help
.DEFAULT_GOAL := help
help: ## Prints all the targets in all the Makefiles
	@grep -h -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

##############
### Checks ###
##############

# TODO_DOCUMENT: All of the `check_` helpers can be installed differently depending
# on the user's OS and enviornment.

.PHONY: check_go_version
# Internal helper target - check go version
check_go_version:
	@# Extract the version number from the `go version` command.
	@GO_VERSION=$$(go version | cut -d " " -f 3 | cut -c 3-) && \
	MAJOR_VERSION=$$(echo $$GO_VERSION | cut -d "." -f 1) && \
	MINOR_VERSION=$$(echo $$GO_VERSION | cut -d "." -f 2) && \
	\
	if [ "$$MAJOR_VERSION" -ne 1 ] || [ "$$MINOR_VERSION" -ge 21 ] ||  [ "$$MINOR_VERSION" -le 18 ] ; then \
		echo "Invalid Go version. Expected 1.19.x or 1.20.x but found $$GO_VERSION"; \
		exit 1; \
	fi

.PHONY: check_act
# Internal helper target - check if `act` is installed
check_act:
	{ \
	if ( ! ( command -v act >/dev/null )); then \
		echo "Seems like you don't have `act` installed. Please visit https://github.com/nektos/act before continuing"; \
		exit 1; \
	fi; \
	}

.PHONY: check_gh
# Internal helper target - check if `gh` is installed
check_gh:
	{ \
	if ( ! ( command -v gh >/dev/null )); then \
		echo "Seems like you don't have `gh` installed. Please visit https://cli.github.com/ before continuing"; \
		exit 1; \
	fi; \
	}


.PHONY: check_docker
# Internal helper target - check if docker is installed
check_docker:
	{ \
	if ( ! ( command -v docker >/dev/null && (docker compose version >/dev/null || command -v docker-compose >/dev/null) )); then \
		echo "Seems like you don't have Docker or docker-compose installed. Make sure you review build/localnet/README.md and docs/development/README.md  before continuing"; \
		exit 1; \
	fi; \
	}

.PHONY: check_godoc
# Internal helper target - check if godoc is installed
check_godoc:
	{ \
	if ( ! ( command -v godoc >/dev/null )); then \
		echo "Seems like you don't have godoc installed. Make sure you install it via 'go install golang.org/x/tools/cmd/godoc@latest' before continuing"; \
		exit 1; \
	fi; \
	}

.PHONY: check_npm
# Internal helper target - check if npm is installed
check_npm:
	{ \
	if ( ! ( command -v npm >/dev/null )); then \
		echo "Seems like you don't have npm installed. Make sure you install it before continuing"; \
		exit 1; \
	fi; \
	}

.PHONY: check_jq
# Internal helper target - check if jq is installed
check_jq:
	{ \
	if ( ! ( command -v jq >/dev/null )); then \
		echo "Seems like you don't have jq installed. Make sure you install it before continuing"; \
		exit 1; \
	fi; \
	}


.PHONY: check_node
# Internal helper target - check if node is installed
check_node:
	{ \
	if ( ! ( command -v node >/dev/null )); then \
		echo "Seems like you don't have node installed. Make sure you install it before continuing"; \
		exit 1; \
	fi; \
	}


.PHONY: warn_destructive
warn_destructive: ## Print WARNING to the user
	@echo "This is a destructive action that will affect docker resources outside the scope of this repo!"

#######################
### Proto  Helpers ####
#######################

.PHONY: proto_regen
proto_regen: ## Delete existing protobuf artifacts and regenerate them
	find . \( -name "*.pb.go" -o -name "*.pb.gw.go" \) | xargs --no-run-if-empty rm
	ignite generate proto-go --yes

#######################
### Docker  Helpers ###
#######################

.PHONY: docker_wipe
docker_wipe: check_docker warn_destructive prompt_user ## [WARNING] Remove all the docker containers, images and volumes.
	docker ps -a -q | xargs -r -I {} docker stop {}
	docker ps -a -q | xargs -r -I {} docker rm {}
	docker images -q | xargs -r -I {} docker rmi {}
	docker volume ls -q | xargs -r -I {} docker volume rm {}

########################
### Localnet Helpers ###
########################

.PHONY: localnet_up
localnet_up: ## Starts localnet
	make localnet_regenesis
	tilt up

.PHONY: localnet_down
localnet_down: ## Delete resources created by localnet
	tilt down
	kubectl delete secret celestia-secret || exit 1

.PHONY: localnet_regenesis
localnet_regenesis: acc_initialize_pubkeys_warn_message ## Regenerate the localnet genesis file
# NOTE: intentionally not using --home <dir> flag to avoid overwriting the test keyring
	ignite chain init
	mkdir -p $(POKTROLLD_HOME)/config/
	cp -r ${HOME}/.poktroll/keyring-test $(POKTROLLD_HOME)
	cp ${HOME}/.poktroll/config/*_key.json $(POKTROLLD_HOME)/config/
	cp ${HOME}/.poktroll/config/genesis.json $(POKTROLLD_HOME)/config/

# TODO_BLOCKER(@okdas): Figure out how to copy these over w/ a functional state.
# cp ${HOME}/.poktroll/config/app.toml $(POKTROLLD_HOME)/config/app.toml
# cp ${HOME}/.poktroll/config/config.toml $(POKTROLLD_HOME)/config/config.toml
# cp ${HOME}/.poktroll/config/client.toml $(POKTROLLD_HOME)/config/client.toml

###############
### Linting ###
###############

.PHONY: go_lint
go_lint: ## Run all go linters
	golangci-lint run --timeout 5m --build-tags test

go_imports: check_go_version ## Run goimports on all go files
	go run ./tools/scripts/goimports

#############
### Tests ###
#############

.PHONY: test_e2e
test_e2e: acc_initialize_pubkeys_warn_message ## Run all E2E tests
	export POCKET_NODE=$(POCKET_NODE) && \
	export APPGATE_SERVER=$(APPGATE_SERVER) && \
	POKTROLLD_HOME=../../$(POKTROLLD_HOME) && \
	go test -v ./e2e/tests/... -tags=e2e,test

.PHONY: go_test_verbose
go_test_verbose: check_go_version ## Run all go tests verbosely
	go test -v -race -tags test ./...

.PHONY: go_test
go_test: check_go_version ## Run all go tests showing detailed output only on failures
	go test -race -tags test ./...

.PHONY: go_test_integration
go_test_integration: check_go_version ## Run all go tests, including integration
	go test -v -race -tags test,integration ./...

.PHONY: itest
itest: check_go_version ## Run tests iteratively (see usage for more)
	./tools/scripts/itest.sh $(filter-out $@,$(MAKECMDGOALS))
# catch-all target for itest
%:
	# no-op
	@:

.PHONY: go_mockgen
go_mockgen: ## Use `mockgen` to generate mocks used for testing purposes of all the modules.
	find . -name "*_mock.go" | xargs --no-run-if-empty rm
	go generate ./x/application/types/
	go generate ./x/gateway/types/
	go generate ./x/supplier/types/
	go generate ./x/session/types/
	go generate ./x/service/types/
	go generate ./x/tokenomics/types/
	go generate ./pkg/client/interface.go
	go generate ./pkg/miner/interface.go
	go generate ./pkg/relayer/interface.go
	go generate ./pkg/crypto/rings/interface.go

.PHONY: go_testgen_fixtures
go_testgen_fixtures: ## Generate fixture data for unit tests
	go generate ./pkg/relayer/miner/miner_test.go

.PHONY: go_testgen_accounts
go_testgen_accounts: ## Generate test accounts for usage in test environments
	go generate ./testutil/testkeyring/keyring.go

.PHONY: go_develop
go_develop: proto_regen go_mockgen ## Generate protos and mocks

.PHONY: go_develop_and_test
go_develop_and_test: go_develop go_test ## Generate protos, mocks and run all tests

.PHONY: load_test_simple
load_test_simple: ## Runs the simpliest load test through the whole stack (appgate -> relayminer -> anvil)
	k6 run load-testing/tests/appGateServerEtherium.js

#############
### TODOS ###
#############

# How do I use TODOs?
# 1. <KEYWORD>: <Description of follow up work>;
# 	e.g. TODO_HACK: This is a hack, we need to fix it later
# 2. If there's a specific issue, or specific person, add that in paranthesiss
#   e.g. TODO(@Olshansk): Automatically link to the Github user https://github.com/olshansk
#   e.g. TODO_INVESTIGATE(#420): Automatically link this to github issue https://github.com/pokt-network/pocket/issues/420
#   e.g. TODO_DISCUSS(@Olshansk, #420): Specific individual should tend to the action item in the specific ticket
#   e.g. TODO_CLEANUP(core): This is not tied to an issue, or a person, but should only be done by the core team.
#   e.g. TODO_CLEANUP: This is not tied to an issue, or a person, and can be done by the core team or external contributors.
# 3. Feel free to add additional keywords to the list above.

# Inspired by @goldinguy_ in this post: https://goldin.io/blog/stop-using-todo ###
# TODO                        - General Purpose catch-all.
# TODO_COMMUNITY              - A TODO that may be a candidate for outsourcing to the community.
# TODO_DECIDE                 - A TODO indicating we need to make a decision and document it using an ADR in the future; https://github.com/pokt-network/pocket-network-protocol/tree/main/ADRs
# TODO_TECHDEBT               - Not a great implementation, but we need to fix it later.
# TODO_BLOCKER                - Similar to TECHDEBT, but of higher priority, urgency & risk prior to the next release
# TODO_IMPROVE                - A nice to have, but not a priority. It's okay if we never get to this.
# TODO_OPTIMIZE               - An opportunity for performance improvement if/when it's necessary
# TODO_DISCUSS                - Probably requires a lengthy offline discussion to understand next steps.
# TODO_INCOMPLETE             - A change which was out of scope of a specific PR but needed to be documented.
# TODO_INVESTIGATE            - TBD what was going on, but needed to continue moving and not get distracted.
# TODO_CLEANUP                - Like TECHDEBT, but not as bad.  It's okay if we never get to this.
# TODO_HACK                   - Like TECHDEBT, but much worse. This needs to be prioritized
# TODO_REFACTOR               - Similar to TECHDEBT, but will require a substantial rewrite and change across the codebase
# TODO_CONSIDERATION          - A comment that involves extra work but was thoughts / considered as part of some implementation
# TODO_CONSOLIDATE            - We likely have similar implementations/types of the same thing, and we should consolidate them.
# TODO_ADDTEST                - Add more tests for a specific code section
# TODO_DEPRECATE              - Code that should be removed in the future
# TODO_RESEARCH               - A non-trivial action item that requires deep research and investigation being next steps can be taken
# TODO_DOCUMENT		          - A comment that involves the creation of a README or other documentation
# TODO_BUG                    - There is a known existing bug in this code
# TODO_NB                     - An important note to reference later
# TODO_DISCUSS_IN_THIS_COMMIT - SHOULD NEVER BE COMMITTED TO MASTER. It is a way for the reviewer of a PR to start / reply to a discussion.
# TODO_IN_THIS_COMMIT         - SHOULD NEVER BE COMMITTED TO MASTER. It is a way to start the review process while non-critical changes are still in progress

.PHONY: todo_list
todo_list: ## List all the TODOs in the project (excludes vendor and prototype directories)
	grep --exclude-dir={.git,vendor,./docusaurus} -r TODO  .

.PHONY: todo_count
todo_count: ## Print a count of all the TODOs in the project
	grep --exclude-dir={.git,vendor,./docusaurus} -r TODO  . | wc -l

.PHONY: todo_this_commit
todo_this_commit: ## List all the TODOs needed to be done in this commit
	grep --exclude-dir={.git,vendor,.vscode} --exclude=Makefile -r -e "TODO_IN_THIS_"

####################
###   Gateways   ###
####################

.PHONY: gateway_list
gateway_list: ## List all the staked gateways
	poktrolld --home=$(POKTROLLD_HOME) q gateway list-gateway --node $(POCKET_NODE)

.PHONY: gateway_stake
gateway_stake: ## Stake tokens for the gateway specified (must specify the gateway env var)
	poktrolld --home=$(POKTROLLD_HOME) tx gateway stake-gateway --config $(POKTROLLD_HOME)/config/$(STAKE) --keyring-backend test --from $(GATEWAY) --node $(POCKET_NODE)

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
	poktrolld --home=$(POKTROLLD_HOME) tx gateway unstake-gateway --keyring-backend test --from $(GATEWAY) --node $(POCKET_NODE)

.PHONY: gateway1_unstake
gateway1_unstake: ## Unstake gateway1
	GATEWAY=gateway1 make gateway_unstake

.PHONY: gateway2_unstake
gateway2_unstake: ## Unstake gateway2
	GATEWAY=gateway2 make gateway_unstake

.PHONY: gateway3_unstake
gateway3_unstake: ## Unstake gateway3
	GATEWAY=gateway3 make gateway_unstake

####################
### Applications ###
####################

.PHONY: app_list
app_list: ## List all the staked applications
	poktrolld --home=$(POKTROLLD_HOME) q application list-application --node $(POCKET_NODE)

.PHONY: app_stake
app_stake: ## Stake tokens for the application specified (must specify the APP and SERVICES env vars)
	poktrolld --home=$(POKTROLLD_HOME) tx application stake-application --config $(POKTROLLD_HOME)/config/$(SERVICES) --keyring-backend test --from $(APP) --node $(POCKET_NODE)

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
	poktrolld --home=$(POKTROLLD_HOME) tx application unstake-application --keyring-backend test --from $(APP) --node $(POCKET_NODE)

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
	poktrolld --home=$(POKTROLLD_HOME) tx application delegate-to-gateway $(GATEWAY_ADDR) --keyring-backend test --from $(APP) --node $(POCKET_NODE)

.PHONY: app1_delegate_gateway1
app1_delegate_gateway1: ## Delegate trust to gateway1
	GATEWAY1=$$(make poktrolld_addr ACC_NAME=gateway1) && \
	APP=app1 GATEWAY_ADDR=$$GATEWAY1 make app_delegate

.PHONY: app2_delegate_gateway2
app2_delegate_gateway2: ## Delegate trust to gateway2
	GATEWAY2=$$(make poktrolld_addr ACC_NAME=gateway2) && \
	APP=app2 GATEWAY_ADDR=$$GATEWAY2 make app_delegate

.PHONY: app3_delegate_gateway3
app3_delegate_gateway3: ## Delegate trust to gateway3
	GATEWAY3=$$(make poktrolld_addr ACC_NAME=gateway3) && \
	APP=app3 GATEWAY_ADDR=$$GATEWAY3 make app_delegate

.PHONY: app_undelegate
app_undelegate: ## Undelegate trust to a gateway (must specify the APP and GATEWAY_ADDR env vars). Requires the app to be staked
	poktrolld --home=$(POKTROLLD_HOME) tx application undelegate-from-gateway $(GATEWAY_ADDR) --keyring-backend test --from $(APP) --node $(POCKET_NODE)

.PHONY: app1_undelegate_gateway1
app1_undelegate_gateway1: ## Undelegate trust to gateway1
	GATEWAY1=$$(make poktrolld_addr ACC_NAME=gateway1) && \
	APP=app1 GATEWAY_ADDR=$$GATEWAY1 make app_undelegate

.PHONY: app2_undelegate_gateway2
app2_undelegate_gateway2: ## Undelegate trust to gateway2
	GATEWAY2=$$(make poktrolld_addr ACC_NAME=gateway2) && \
	APP=app2 GATEWAY_ADDR=$$GATEWAY2 make app_undelegate

.PHONY: app3_undelegate_gateway3
app3_undelegate_gateway3: ## Undelegate trust to gateway3
	GATEWAY3=$$(make poktrolld_addr ACC_NAME=gateway3) && \
	APP=app3 GATEWAY_ADDR=$$GATEWAY3 make app_undelegate

#################
### Suppliers ###
#################

.PHONY: supplier_list
supplier_list: ## List all the staked supplier
	poktrolld --home=$(POKTROLLD_HOME) q supplier list-supplier --node $(POCKET_NODE)

.PHONY: supplier_stake
supplier_stake: ## Stake tokens for the supplier specified (must specify the SUPPLIER and SUPPLIER_CONFIG env vars)
	poktrolld --home=$(POKTROLLD_HOME) tx supplier stake-supplier --config $(POKTROLLD_HOME)/config/$(SERVICES) --keyring-backend test --from $(SUPPLIER) --node $(POCKET_NODE)

.PHONY: supplier1_stake
supplier1_stake: ## Stake supplier1 (also staked in genesis)
	SUPPLIER=supplier1 SERVICES=supplier1_stake_config.yaml make supplier_stake

.PHONY: supplier2_stake
supplier2_stake: ## Stake supplier2
	SUPPLIER=supplier2 SERVICES=supplier2_stake_config.yaml make supplier_stake

.PHONY: supplier3_stake
supplier3_stake: ## Stake supplier3
	SUPPLIER=supplier3 SERVICES=supplier3_stake_config.yaml make supplier_stake

.PHONY: supplier_unstake
supplier_unstake: ## Unstake an supplier (must specify the SUPPLIER env var)
	poktrolld --home=$(POKTROLLD_HOME) tx supplier unstake-supplier --keyring-backend test --from $(SUPPLIER) --node $(POCKET_NODE)

.PHONY: supplier1_unstake
supplier1_unstake: ## Unstake supplier1
	SUPPLIER=supplier1 make supplier_unstake

.PHONY: supplier2_unstake
supplier2_unstake: ## Unstake supplier2
	SUPPLIER=supplier2 make supplier_unstake

.PHONY: supplier3_unstake
supplier3_unstake: ## Unstake supplier3
	SUPPLIER=supplier3 make supplier_unstake

###############
### Session ###
###############

.PHONY: get_session
get_session: ## Retrieve the session given the following env vars: (APP_ADDR, SVC, HEIGHT)
	poktrolld --home=$(POKTROLLD_HOME) q session get-session $(APP) $(SVC) $(HEIGHT) --node $(POCKET_NODE)

.PHONY: get_session_app1_anvil
get_session_app1_anvil: ## Retrieve the session for (app1, anvil, latest_height)
	APP1=$$(make poktrolld_addr ACC_NAME=app1) && \
	APP=$$APP1 SVC=anvil HEIGHT=0 make get_session

.PHONY: get_session_app2_anvil
get_session_app2_anvil: ## Retrieve the session for (app2, anvil, latest_height)
	APP2=$$(make poktrolld_addr ACC_NAME=app2) && \
	APP=$$APP2 SVC=anvil HEIGHT=0 make get_session

.PHONY: get_session_app3_anvil
get_session_app3_anvil: ## Retrieve the session for (app3, anvil, latest_height)
	APP3=$$(make poktrolld_addr ACC_NAME=app3) && \
	APP=$$APP3 SVC=anvil HEIGHT=0 make get_session

################
### Accounts ###
################

.PHONY: acc_balance_query
acc_balance_query: ## Query the balance of the account specified (make acc_balance_query ACC=pokt...)
	@echo "~~~ Balances ~~~"
	poktrolld --home=$(POKTROLLD_HOME) q bank balances $(ACC) --node $(POCKET_NODE)
	@echo "~~~ Spendable Balances ~~~"
	@echo "Querying spendable balance for $(ACC)"
	poktrolld --home=$(POKTROLLD_HOME) q bank spendable-balances $(ACC) --node $(POCKET_NODE)

.PHONY: acc_balance_query_module_app
acc_balance_query_module_app: ## Query the balance of the network level "application" module
	make acc_balance_query ACC=pokt1rl3gjgzexmplmds3tq3r3yk84zlwdl6djzgsvm

.PHONY: acc_balance_query_module_supplier
acc_balance_query_module_supplier: ## Query the balance of the network level "supplier" module
	SUPPLIER1=$(make poktrolld_addr ACC_NAME=supplier1)
	make acc_balance_query ACC=SUPPLIER1

.PHONY: acc_balance_query_app1
acc_balance_query_app1: ## Query the balance of app1
	APP1=$$(make poktrolld_addr ACC_NAME=app1) && \
	make acc_balance_query ACC=$$APP1

.PHONY: acc_balance_total_supply
acc_balance_total_supply: ## Query the total supply of the network
	poktrolld --home=$(POKTROLLD_HOME) q bank total --node $(POCKET_NODE)

# NB: Ignite does not populate `pub_key` in `accounts` within `genesis.json` leading
# to queries like this to fail: `poktrolld query account pokt1<addr> --node $(POCKET_NODE).
# We attempted using a `tx multi-send` from the `faucet` to all accounts, but
# that also did not solve this problem because the account itself must sign the
# transaction for its public key to be populated in the account keeper. As such,
# the solution is to send funds from every account in genesis to some address
# (PNF was selected ambigously) to make sure their public keys are populated.

.PHONY: acc_initialize_pubkeys
acc_initialize_pubkeys: ## Make sure the account keeper has public keys for all available accounts
	$(eval ADDRESSES=$(shell make -s ignite_acc_list | grep pokt | awk '{printf "%s ", $$2}' | sed 's/.$$//'))
	$(eval PNF_ADDR=pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw)
	# @printf "Addresses: ${ADDRESSES}"
	$(foreach addr, $(ADDRESSES),\
		echo $(addr);\
		poktrolld tx bank send \
			$(addr) $(PNF_ADDR) 1000upokt \
			--yes \
			--home=$(POKTROLLD_HOME) \
			--node $(POCKET_NODE);)

.PHONY: acc_initialize_pubkeys_warn_message
acc_initialize_pubkeys_warn_message: ## Print a warning message about the need to run `make acc_initialize_pubkeys`
	@printf "!!! YOU MUST RUN THE FOLLOWING COMMAND ONCE FOR E2E TESTS TO WORK AFTER THE NETWORK HAS STARTED!!!\n"\
	"\t\tmake acc_initialize_pubkeys\n"

##############
### Claims ###
##############

# These encoded values were generated using the `encodeSessionHeader` helpers in `query_claim_test.go` as dummy values.
ENCODED_SESSION_HEADER = "eyJhcHBsaWNhdGlvbl9hZGRyZXNzIjoicG9rdDFleXJuNDUwa3JoZnpycmVyemd0djd2c3J4bDA5NDN0dXN4azRhayIsInNlcnZpY2UiOnsiaWQiOiJhbnZpbCIsIm5hbWUiOiIifSwic2Vzc2lvbl9zdGFydF9ibG9ja19oZWlnaHQiOiI1Iiwic2Vzc2lvbl9pZCI6InNlc3Npb25faWQxIiwic2Vzc2lvbl9lbmRfYmxvY2tfaGVpZ2h0IjoiOSJ9"
ENCODED_ROOT_HASH = "cm9vdF9oYXNo"
.PHONY: claim_create_dummy
claim_create_dummy: ## Create a dummy claim by supplier1
	poktrolld --home=$(POKTROLLD_HOME) tx supplier create-claim \
	$(ENCODED_SESSION_HEADER) \
	$(ENCODED_ROOT_HASH) \
	--from supplier1 --node $(POCKET_NODE)

.PHONY: claims_list
claim_list: ## List all the claims
	poktrolld --home=$(POKTROLLD_HOME) q supplier list-claims --node $(POCKET_NODE)

.PHONY: claims_list_address
claim_list_address: ## List all the claims for a specific address (specified via ADDR variable)
	poktrolld --home=$(POKTROLLD_HOME) q supplier list-claims --supplier-address $(ADDR) --node $(POCKET_NODE)

.PHONY: claims_list_address_supplier1
claim_list_address_supplier1: ## List all the claims for supplier1
	SUPPLIER1=$$(make poktrolld_addr ACC_NAME=supplier1) && \
	ADDR=$$SUPPLIER1 make claim_list_address

.PHONY: claim_list_height
claim_list_height: ## List all the claims ending at a specific height (specified via HEIGHT variable)
	poktrolld --home=$(POKTROLLD_HOME) q supplier list-claims --session-end-height $(HEIGHT) --node $(POCKET_NODE)

.PHONY: claim_list_height_5
claim_list_height_5: ## List all the claims at height 5
	HEIGHT=5 make claim_list_height

.PHONY: claim_list_session
claim_list_session: ## List all the claims ending at a specific session (specified via SESSION variable)
	poktrolld --home=$(POKTROLLD_HOME) q supplier list-claims --session-id $(SESSION) --node $(POCKET_NODE)

##############
### Params ###
##############

MODULES := application gateway pocket service session supplier tokenomics

# TODO_IMPROVE(#322): Improve once we decide how to handle parameter updates
.PHONY: update_tokenomics_params
update_tokenomics_params: ## Update the tokenomics module params
	poktrolld --home=$(POKTROLLD_HOME) tx tokenomics update-params 43 --keyring-backend test --from pnf --node $(POCKET_NODE)

.PHONY: query_all_params
query_all_params: check_jq ## Query the params from all available modules
	@for module in $(MODULES); do \
	    echo "~~~ Querying $$module module params ~~~"; \
	    poktrolld query $$module params --node $(POCKET_NODE) --output json | jq; \
	    echo ""; \
	done

######################
### Ignite Helpers ###
######################

.PHONY: ignite_acc_list
ignite_acc_list: ## List all the accounts in LocalNet
	ignite account list --keyring-dir=$(POKTROLLD_HOME) --keyring-backend test --address-prefix $(POCKET_ADDR_PREFIX)

##################
### CI Helpers ###
##################

.PHONY: trigger_ci
trigger_ci: ## Trigger the CI pipeline by submitting an empty commit; See https://github.com/pokt-network/pocket/issues/900 for details
	git commit --allow-empty -m "Empty commit"
	git push

#####################
### Documentation ###
#####################

.PHONY: go_docs
go_docs: check_godoc ## Generate documentation for the project
	echo "Visit http://localhost:6060/pkg/github.com/pokt-network/poktroll/"
	godoc -http=:6060

.PHONY: openapi_gen
openapi_gen: ## Generate the OpenAPI spec for the Ignite API
	ignite generate openapi --yes

.PHONY: docusaurus_start
docusaurus_start: check_npm check_node ## Start the Docusaurus server
	(cd docusaurus && npm i && npm run start)

######################
### Ignite Helpers ###
######################

.PHONY: poktrolld_addr
poktrolld_addr: ## Retrieve the address for an account by ACC_NAME
	@echo $(shell poktrolld --home=$(POKTROLLD_HOME) keys show -a $(ACC_NAME))


###################
### Act Helpers ###
###################

.PHONY: detect_arch
# Internal helper to avoid the caller needing to specify the architecture
detect_arch:
	@ARCH=`uname -m`; \
	case $$ARCH in \
	x86_64) \
		echo linux/amd64 ;; \
	arm64) \
		echo linux/arm64 ;; \
	*) \
		echo "Unsupported architecture: $$ARCH" >&2; \
		exit 1 ;; \
	esac

.PHONY: act_list
act_list: check_act ## List all github actions that can be executed locally with act
	act --list

.PHONY: act_reviewdog
act_reviewdog: check_act check_gh ## Run the reviewdog workflow locally like so: `GITHUB_TOKEN=$(gh auth token) make act_reviewdog`
	$(eval CONTAINER_ARCH := $(shell make -s detect_arch))
	@echo "Detected architecture: $(CONTAINER_ARCH)"
	act -v -s GITHUB_TOKEN=$(GITHUB_TOKEN) -W .github/workflows/reviewdog.yml --container-architecture $(CONTAINER_ARCH)