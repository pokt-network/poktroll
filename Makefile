.SILENT:

POCKETD_HOME := ./localnet/pocketd
POCKET_NODE = tcp://127.0.0.1:36657 # The pocket rollup node (full node and sequencer in the localnet context)
POCKET_ADDR_PREFIX = pokt

####################
### Dependencies ###
####################

# TODO: Add other dependencies (ignite, docker, k8s, etc) here
.PHONY: install_ci_deps
install_ci_deps: ## Installs `mockgen`
	go install "github.com/golang/mock/mockgen@v1.6.0" && mockgen --version

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

.PHONY: go_version_check
# Internal helper target - check go version
go_version_check:
	@# Extract the version number from the `go version` command.
	@GO_VERSION=$$(go version | cut -d " " -f 3 | cut -c 3-) && \
	MAJOR_VERSION=$$(echo $$GO_VERSION | cut -d "." -f 1) && \
	MINOR_VERSION=$$(echo $$GO_VERSION | cut -d "." -f 2) && \
	\
	if [ "$$MAJOR_VERSION" -ne 1 ] || [ "$$MINOR_VERSION" -ge 21 ] ||  [ "$$MINOR_VERSION" -le 18 ] ; then \
		echo "Invalid Go version. Expected 1.19.x or 1.20.x but found $$GO_VERSION"; \
		exit 1; \
	fi

.PHONY: docker_check
# Internal helper target - check if docker is installed
docker_check:
	{ \
	if ( ! ( command -v docker >/dev/null && (docker compose version >/dev/null || command -v docker-compose >/dev/null) )); then \
		echo "Seems like you don't have Docker or docker-compose installed. Make sure you review build/localnet/README.md and docs/development/README.md  before continuing"; \
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
docker_wipe: docker_check warn_destructive prompt_user ## [WARNING] Remove all the docker containers, images and volumes.
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
localnet_regenesis: ## Regenerate the localnet genesis file
# NOTE: intentionally not using --home <dir> flag to avoid overwriting the test keyring
	ignite chain init --skip-proto
	cp -r ${HOME}/.pocket/keyring-test $(POCKETD_HOME)
	cp ${HOME}/.pocket/config/*_key.json $(POCKETD_HOME)/config/
	cp ${HOME}/.pocket/config/genesis.json $(POCKETD_HOME)/config/

#############
### Tests ###
#############

.PHONY: test_e2e
test_e2e: ## Run all E2E tests
	export POCKET_NODE=$(POCKET_NODE) POCKETD_HOME=../../$(POCKETD_HOME) && go test -v ./e2e/tests/... -tags=e2e

.PHONY: go_test
go_test: go_version_check ## Run all go tests
	go test -v ./...

.PHONY: go_mockgen
go_mockgen: ## Use `mockgen` to generate mocks used for testing purposes of all the modules.
	go generate ./x/application/types/
	go generate ./x/gateway/types/

.PHONY: go_develop
go_develop: proto_regen go_mockgen go_test ## Generate protos, mocks and run all tests

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
# TODO_DECIDE                 - A TODO indicating we need to make a decision and document it using an ADR in the future; https://github.com/pokt-network/pocket-network-protocol/tree/main/ADRs
# TODO_TECHDEBT               - Not a great implementation, but we need to fix it later.
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
TODO_KEYWORDS = -e "TODO" -e "TODO_DECIDE" -e "TODO_TECHDEBT" -e "TODO_IMPROVE" -e "TODO_OPTIMIZE" -e "TODO_DISCUSS" -e "TODO_INCOMPLETE" -e "TODO_INVESTIGATE" -e "TODO_CLEANUP" -e "TODO_HACK" -e "TODO_REFACTOR" -e "TODO_CONSIDERATION" -e "TODO_IN_THIS_COMMIT" -e "TODO_DISCUSS_IN_THIS_COMMIT" -e "TODO_CONSOLIDATE" -e "TODO_DEPRECATE" -e "TODO_ADDTEST" -e "TODO_RESEARCH" -e "TODO_BUG" -e "TODO_NB" -e "TODO_DISCUSS_IN_THIS_COMMIT" -e "TODO_IN_THIS_COMMIT"

.PHONY: todo_list
todo_list: ## List all the TODOs in the project (excludes vendor and prototype directories)
	grep --exclude-dir={.git,vendor,prototype} -r ${TODO_KEYWORDS}  .

TODO_SEARCH ?= $(shell pwd)

.PHONY: todo_search
todo_search: ## List all the TODOs in a specific directory specific by `TODO_SEARCH`
	grep --exclude-dir={.git,vendor,prototype} -r ${TODO_KEYWORDS} ${TODO_SEARCH}

.PHONY: todo_count
todo_count: ## Print a count of all the TODOs in the project
	grep --exclude-dir={.git,vendor,prototype} -r ${TODO_KEYWORDS} . | wc -l

.PHONY: todo_this_commit
todo_this_commit: ## List all the TODOs needed to be done in this commit
	grep --exclude-dir={.git,vendor,prototype,.vscode} --exclude=Makefile -r -e "TODO_IN_THIS_COMMIT" -e "DISCUSS_IN_THIS_COMMIT"

####################
### Applications ###
####################

.PHONY: app_list
app_list: ## List all the staked applications
	pocketd --home=$(POCKETD_HOME) q application list-application --node $(POCKET_NODE)

.PHONY: app_stake
app_stake: ## Stake tokens for the application specified (must specify the APP env var)
	pocketd --home=$(POCKETD_HOME) tx application stake-application 1000upokt --keyring-backend test --from $(APP) --node $(POCKET_NODE)

.PHONY: app1_stake
app1_stake: ## Stake app1
	APP=app1 make app_stake

.PHONY: app2_stake
app2_stake: ## Stake app2
	APP=app2 make app_stake

.PHONY: app3_stake
app3_stake: ## Stake app3
	APP=app3 make app_stake

.PHONY: app_unstake
app_unstake: ## Unstake an application (must specify the APP env var)
	pocketd --home=$(POCKETD_HOME) tx application unstake-application --keyring-backend test --from $(APP) --node $(POCKET_NODE)

.PHONY: app1_unstake
app1_unstake: ## Unstake app1
	APP=app1 make app_unstake

.PHONY: app2_unstake
app2_unstake: ## Unstake app2
	APP=app2 make app_unstake

.PHONY: app3_unstake
app3_unstake: ## Unstake app3
	APP=app3 make app_unstake

################
### Accounts ###
################

.PHONY: acc_balance_query
acc_balance_query: ## Query the balance of the account specified (make acc_balance_query ACC=pokt...)
	@echo "~~~ Balances ~~~"
	pocketd --home=$(POCKETD_HOME) q bank balances $(ACC) --node $(POCKET_NODE)
	@echo "~~~ Spendable Balances ~~~"
	@echo "Querying spendable balance for $(ACC)"
	pocketd --home=$(POCKETD_HOME) q bank spendable-balances $(ACC) --node $(POCKET_NODE)

.PHONY: acc_balance_query_app_module
acc_balance_query_app_module: ## Query the balance of the network level "application" module
	make acc_balance_query ACC=pokt1rl3gjgzexmplmds3tq3r3yk84zlwdl6djzgsvm

.PHONY: acc_balance_query_app1
acc_balance_query_app1: ## Query the balance of app1
	make acc_balance_query ACC=pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4

.PHONY: acc_balance_total_supply
acc_balance_total_supply: ## Query the total supply of the network
	pocketd --home=$(POCKETD_HOME) q bank total --node $(POCKET_NODE)

######################
### Ignite Helpers ###
######################

.PHONY: ignite_acc_list
ignite_acc_list: ## List all the accounts in LocalNet
	ignite account list --keyring-dir=$(POCKETD_HOME) --keyring-backend test --address-prefix $(POCKET_ADDR_PREFIX)