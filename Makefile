.SILENT:

SHELL = /bin/sh

POKTROLLD_HOME ?= ./localnet/poktrolld
POCKET_NODE ?= tcp://127.0.0.1:26657 # The pocket node (validator in the localnet context)
TESTNET_RPC ?= https://testnet-validated-validator-rpc.poktroll.com/ # TestNet RPC endpoint for validator maintained by Grove. Needs to be update if there's another "primary" testnet.
APPGATE_SERVER ?= http://localhost:42069
GATEWAY_URL ?= http://localhost:42079
POCKET_ADDR_PREFIX = pokt
LOAD_TEST_CUSTOM_MANIFEST ?= loadtest_manifest_example.yaml

# The domain ending in ".town" is staging, ".city" is production
GROVE_GATEWAY_STAGING_ETH_MAINNET = https://eth-mainnet.rpc.grove.town
# JSON RPC data for a test relay request
JSON_RPC_DATA_ETH_BLOCK_HEIGHT = '{"jsonrpc":"2.0","id":"0","method":"eth_blockNumber", "params": []}'

# On-chain module account addresses. Search for `func TestModuleAddress` in the
# codebase to get an understanding of how we got these values.
APPLICATION_MODULE_ADDRESS = pokt1rl3gjgzexmplmds3tq3r3yk84zlwdl6djzgsvm
SUPPLIER_MODULE_ADDRESS = pokt1j40dzzmn6cn9kxku7a5tjnud6hv37vesr5ccaa
GATEWAY_MODULE_ADDRESS = pokt1f6j7u6875p2cvyrgjr0d2uecyzah0kget9vlpl
SERVICE_MODULE_ADDRESS = pokt1nhmtqf4gcmpxu0p6e53hpgtwj0llmsqpxtumcf
GOV_ADDRESS = pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t
# PNF acts on behalf of the DAO and who AUTHZ must delegate to
PNF_ADDRESS = pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw

MODULES := application gateway pocket service session supplier proof tokenomics

BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
COMMIT := $(shell git log -1 --format='%H')

# don't override user values
ifeq (,$(VERSION))
  # Remove 'v' prefix from git tag and assign to VERSION
  VERSION := $(shell git describe --tags 2>/dev/null | sed 's/^v//')
  # if VERSION is empty, then populate it with branch's name and raw commit hash
  ifeq (,$(VERSION))
    VERSION := $(BRANCH)-$(COMMIT)
  endif
endif

# Detect operating system and arch
OS := $(shell uname -s | tr A-Z a-z)
ARCH := $(shell uname -m)
ifeq ($(ARCH),x86_64)
	ARCH := amd64
endif
ifeq ($(ARCH),aarch64)
	ARCH := arm64
endif

# Set default commands, will potentially be overridden on macOS
SED := sed
GREP := grep

# macOS-specific adjustments
ifeq ($(OS),darwin)
    # Check for gsed and ggrep, suggest installation with Homebrew if not found
    FOUND_GSED := $(shell command -v gsed)
    FOUND_GGREP := $(shell command -v ggrep)
    ifeq ($(FOUND_GSED),)
        $(warning GNU sed (gsed) is not installed. Please install it using Homebrew by running: brew install gnu-sed)
        SED := gsed # Assuming the user will install it, setting the variable in advance
    else
        SED := gsed
    endif
    ifeq ($(FOUND_GGREP),)
        $(warning GNU grep (ggrep) is not installed. Please install it using Homebrew by running: brew install grep)
        GREP := ggrep # Assuming the user will install it, setting the variable in advance
    else
        GREP := ggrep
    endif
endif

####################
### Dependencies ###
####################

# TODO_IMPROVE(@okdas): Add other dependencies (ignite, docker, k8s, etc) here
.PHONY: install_ci_deps
install_ci_deps: ## Installs `mockgen` and other go tools
	go install "github.com/golang/mock/mockgen@v1.6.0" && mockgen --version
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.59.1 && golangci-lint --version
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/mikefarah/yq/v4@latest

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
	@grep -h -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-60s\033[0m %s\n", $$1, $$2}'

##############
### Checks ###
##############

# TODO_DOCUMENT: All of the `check_` helpers can be installed differently depending
# on the user's OS and environment.
# NB: For mac users, you may need to install with the proper linkers: https://github.com/golang/go/issues/65940

.PHONY: check_go_version
# Internal helper target - check go version
check_go_version:
	@# Extract the version number from the `go version` command.
	@GO_VERSION=$$(go version | cut -d " " -f 3 | cut -c 3-) && \
	MAJOR_VERSION=$$(echo $$GO_VERSION | cut -d "." -f 1) && \
	MINOR_VERSION=$$(echo $$GO_VERSION | cut -d "." -f 2) && \
	\
	if [ "$$MAJOR_VERSION" -ne 1 ] || [ "$$MINOR_VERSION" -le 20 ] ; then \
		echo "Invalid Go version. Expected 1.21.x or newer but found $$GO_VERSION"; \
		exit 1; \
	fi

.PHONY: check_ignite_version
# Internal helper target - check ignite version
check_ignite_version:
	@version=$$(ignite version 2>/dev/null | grep 'Ignite CLI version:' | awk '{print $$4}') ; \
	if [ "$$(printf "v28\n$$version" | sort -V | head -n1)" != "v28" ]; then \
		echo "Error: Version $$version is less than v28. Exiting with error." ; \
		exit 1 ; \
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
.PHONY: check_kind
# Internal helper target - check if kind is installed
check_kind:
	@if ! command -v kind >/dev/null 2>&1; then \
		echo "kind is not installed. Make sure you review build/localnet/README.md and docs/development/README.md  before continuing"; \
		exit 1; \
	fi

.PHONY: check_docker_ps
 ## Internal helper target - checks if Docker is running
check_docker_ps: check_docker
	@echo "Checking if Docker is running..."
	@docker ps > /dev/null 2>&1 || (echo "Docker is not running. Please start Docker and try again."; exit 1)

.PHONY: check_kind_context
## Internal helper target - checks if the kind-kind context exists and is set
check_kind_context: check_kind
	@if ! kubectl config get-contexts | grep -q 'kind-kind'; then \
		echo "kind-kind context does not exist. Please create it or switch to it."; \
		exit 1; \
	fi
	@if ! kubectl config current-context | grep -q 'kind-kind'; then \
		echo "kind-kind context is not currently set. Use 'kubectl config use-context kind-kind' to set it."; \
		exit 1; \
	fi


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

.PHONY: check_yq
# Internal helper target - check if `yq` is installed
check_yq:
	{ \
	if ( ! ( command -v yq >/dev/null )); then \
		echo "Seems like you don't have `yq` installed. Make sure you install it before continuing"; \
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

.PHONY: proto_ignite_gen
proto_ignite_gen: ## Generate protobuf artifacts using ignite
	ignite generate proto-go --yes

proto_fix_self_import: ## TODO_TECHDEBT(@bryanchriswhite): Add a proper explanation for this make target explaining why it's necessary
	@echo "Updating all instances of cosmossdk.io/api/poktroll to github.com/pokt-network/poktroll/api/poktroll..."
	@find ./api/poktroll/ -type f | while read -r file; do \
		$(SED) -i 's,cosmossdk.io/api/poktroll,github.com/pokt-network/poktroll/api/poktroll,g' "$$file"; \
	done
	@for dir in $(wildcard ./api/poktroll/*/); do \
			module=$$(basename $$dir); \
			echo "Further processing module $$module"; \
			$(GREP) -lRP '\s+'$$module' "github.com/pokt-network/poktroll/api/poktroll/'$$module'"' ./api/poktroll/$$module | while read -r file; do \
					echo "Modifying file: $$file"; \
					$(SED) -i -E 's,^[[:space:]]+'$$module'[[:space:]]+"github.com/pokt-network/poktroll/api/poktroll/'$$module'",,' "$$file"; \
					$(SED) -i 's,'$$module'\.,,g' "$$file"; \
			done; \
	done


.PHONY: proto_clean
proto_clean: ## Delete existing .pb.go or .pb.gw.go files
	find . \( -name "*.pb.go" -o -name "*.pb.gw.go" \) | xargs --no-run-if-empty rm

## TODO_TECHDEBT(@bryanchriswhite): Investigate if / how this can be integrated with `proto_regen`
.PHONY: proto_clean_pulsar
proto_clean_pulsar: ## TODO_TECHDEBT(@bryanchriswhite): Add a proper explanation for this make target explaining why it's necessary
	@find ./ -name "*.go" | xargs --no-run-if-empty $(SED) -i -E 's,(^[[:space:]_[:alnum:]]+"github.com/pokt-network/poktroll/api.+"),///\1,'
	find ./ -name "*.pulsar.go" | xargs --no-run-if-empty rm
	$(MAKE) proto_regen
	find ./ -name "*.go" | xargs --no-run-if-empty $(SED) -i -E 's,^///([[:space:]_[:alnum:]]+"github.com/pokt-network/poktroll/api.+"),\1,'

.PHONY: proto_regen
proto_regen: proto_clean proto_ignite_gen proto_fix_self_import ## Regenerate protobuf artifacts

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
localnet_up: check_docker_ps check_kind_context proto_regen localnet_regenesis ## Starts up a clean localnet
	tilt up

.PHONY: localnet_up_quick
localnet_up_quick: check_docker_ps check_kind_context ## Starts up a localnet without regenerating fixtures
	tilt up

.PHONY: localnet_down
localnet_down: ## Delete resources created by localnet
	tilt down

.PHONY: localnet_regenesis
localnet_regenesis: check_yq warn_message_acc_initialize_pubkeys ## Regenerate the localnet genesis file
# NOTE: intentionally not using --home <dir> flag to avoid overwriting the test keyring
	@echo "Initializing chain..."
	@set -e
	@ignite chain init --skip-proto
	AUTH_CONTENT=$$(cat ./tools/scripts/authz/dao_genesis_authorizations.json | jq -r tostring); \
	$(SED) -i -E 's!^(\s*)"authorization": (\[\]|null)!\1"authorization": '$$AUTH_CONTENT'!' ${HOME}/.poktroll/config/genesis.json;

	@cp -r ${HOME}/.poktroll/keyring-test $(POKTROLLD_HOME)
	@cp -r ${HOME}/.poktroll/config $(POKTROLLD_HOME)/

.PHONY: send_relay_sovereign_app_JSONRPC
send_relay_sovereign_app_JSONRPC: # Send a JSONRPC relay through the AppGateServer as a sovereign application
	curl -X POST -H "Content-Type: application/json" \
	--data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
	$(APPGATE_SERVER)/anvil

.PHONY: send_relay_delegating_app_JSONRPC
send_relay_delegating_app_JSONRPC: # Send a relay through the gateway as an application that's delegating to this gateway
	@appAddr=$$(poktrolld keys show app1 -a) && \
	curl -X POST -H "Content-Type: application/json" \
	--data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
	$(GATEWAY_URL)/anvil?applicationAddr=$$appAddr

.PHONY: send_relay_sovereign_app_REST
send_relay_sovereign_app_REST: # Send a REST relay through the AppGateServer as a sovereign application
	curl -X POST -H "Content-Type: application/json" \
	--data '{"model": "qwen:0.5b", "stream": false, "messages": [{"role": "user", "content":"count from 1 to 10"}]}' \
	$(APPGATE_SERVER)/ollama/api/chat

.PHONY: cosmovisor_start_node
cosmovisor_start_node: # Starts the node using cosmovisor that waits for an upgrade plan
	bash tools/scripts/upgrades/cosmovisor-start-node.sh

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

.PHONY: test_e2e_env
test_e2e_env: warn_message_acc_initialize_pubkeys ## Setup the default env vars for E2E tests
	export POCKET_NODE=$(POCKET_NODE) && \
	export APPGATE_SERVER=$(APPGATE_SERVER) && \
	export POKTROLLD_HOME=../../$(POKTROLLD_HOME)

.PHONY: test_e2e
test_e2e: test_e2e_env ## Run all E2E tests
	go test -count=1 -v ./e2e/tests/... -tags=e2e,test

.PHONY: test_e2e_relay
test_e2e_relay: test_e2e_env ## Run only the E2E suite that exercises the relay life-cycle
	go test -v ./e2e/tests/... -tags=e2e,test --features-path=relay.feature

.PHONY: test_e2e_app
test_e2e_app: test_e2e_env ## Run only the E2E suite that exercises the application life-cycle
	go test -v ./e2e/tests/... -tags=e2e,test --features-path=stake_app.feature

.PHONY: test_e2e_supplier
test_e2e_supplier: test_e2e_env ## Run only the E2E suite that exercises the supplier life-cycle
	go test -v ./e2e/tests/... -tags=e2e,test --features-path=stake_supplier.feature

.PHONY: test_e2e_gateway
test_e2e_gateway: test_e2e_env ## Run only the E2E suite that exercises the gateway life-cycle
	go test -v ./e2e/tests/... -tags=e2e,test --features-path=stake_gateway.feature

.PHONY: test_e2e_session
test_e2e_session: test_e2e_env ## Run only the E2E suite that exercises the session (i.e. claim/proof) life-cycle
	go test -v ./e2e/tests/... -tags=e2e,test --features-path=session.feature

.PHONY: test_e2e_settlement
test_e2e_settlement: test_e2e_env ## Run only the E2E suite that exercises the session & tokenomics settlement
	go test -v ./e2e/tests/... -tags=e2e,test --features-path=0_settlement.feature

.PHONY: test_e2e_params
test_e2e_params: test_e2e_env ## Run only the E2E suite that exercises parameter updates for all modules
	go test -v ./e2e/tests/... -tags=e2e,test --features-path=update_params.feature

.PHONY: test_load_relays_stress_custom
test_load_relays_stress_custom: ## Run the stress test for E2E relays using custom manifest. "loadtest_manifest_example.yaml" manifest is used by default. Set `LOAD_TEST_CUSTOM_MANIFEST` environment variable to use the different manifest.
	go test -v -count=1 ./load-testing/tests/... \
	-tags=load,test -run LoadRelays --log-level=debug --timeout=30m \
	--manifest ./load-testing/$(LOAD_TEST_CUSTOM_MANIFEST)

.PHONY: test_load_relays_stress_localnet
test_load_relays_stress_localnet: test_e2e_env warn_message_local_stress_test ## Run the stress test for E2E relays on LocalNet.
	go test -v -count=1 ./load-testing/tests/... \
	-tags=load,test -run LoadRelays --log-level=debug --timeout=30m \
	--manifest ./load-testing/loadtest_manifest_localnet.yaml

.PHONY: test_verbose
test_verbose: check_go_version ## Run all go tests verbosely
	go test -count=1 -v -race -tags test ./...

# NB: buildmode=pie is necessary to avoid linker errors on macOS.
# It is not compatible with `-race`, which is why it's omitted here.
# See ref for more details: https://github.com/golang/go/issues/54482#issuecomment-1251124908
.PHONY: test_all
test_all: warn_flaky_tests check_go_version ## Run all go tests showing detailed output only on failures
	go test -count=1 -buildmode=pie -tags test ./...

.PHONY: test_all_with_integration
test_all_with_integration: check_go_version ## Run all go tests, including those with the integration
	go test -count=1 -v -race -tags test,integration ./...

# We are explicitly using an env variable rather than a build tag to keep flaky
# tests in line with non flaky tests and use it as a way to easily turn them
# on and off without maintaining extra files.
.PHONY: test_all_with_integration_and_flaky
test_all_with_integration_and_flaky: check_go_version ## Run all go tests, including those with the integration and flaky tests
	INCLUDE_FLAKY_TESTS=true go test -count=1 -v -race -tags test,integration ./...

.PHONY: test_integration
test_integration: check_go_version ## Run only the in-memory integration "unit" tests
	go test -count=1 -v -race -tags test,integration ./tests/integration/...

.PHONY: itest
itest: check_go_version ## Run tests iteratively (see usage for more)
	./tools/scripts/itest.sh $(filter-out $@,$(MAKECMDGOALS))

.PHONY: go_mockgen
go_mockgen: ## Use `mockgen` to generate mocks used for testing purposes of all the modules.
	find . -name "*_mock.go" | xargs --no-run-if-empty rm
	go generate ./x/application/types/
	go generate ./x/gateway/types/
	go generate ./x/supplier/types/
	go generate ./x/session/types/
	go generate ./x/service/types/
	go generate ./x/proof/types/
	go generate ./x/tokenomics/types/
	find . -name interface.go | xargs -I {} go generate {}

.PHONY: go_testgen_fixtures
go_testgen_fixtures: ## Generate fixture data for unit tests
	go generate ./pkg/relayer/miner/miner_test.go

.PHONY: go_testgen_accounts
go_testgen_accounts: ## Generate test accounts for usage in test environments
	go generate ./testutil/testkeyring/keyring.go

.PHONY: go_develop
go_develop: check_ignite_version proto_regen go_mockgen ## Generate protos and mocks

.PHONY: go_develop_and_test
go_develop_and_test: go_develop test_all ## Generate protos, mocks and run all tests

#############
### TODOS ###
#############

# How do I use TODOs?
# 1. <KEYWORD>: <Description of follow up work>;
# 	e.g. TODO_HACK: This is a hack, we need to fix it later
# 2. If there's a specific issue, or specific person, add that in paranthesiss
#   e.g. TODO(@Olshansk): Automatically link to the Github user https://github.com/olshansk
#   e.g. TODO_INVESTIGATE(#420): Automatically link this to github issue https://github.com/pokt-network/poktroll/issues/420
#   e.g. TODO_DISCUSS(@Olshansk, #420): Specific individual should tend to the action item in the specific ticket
#   e.g. TODO_CLEANUP(core): This is not tied to an issue, or a person, but should only be done by the core team.
#   e.g. TODO_CLEANUP: This is not tied to an issue, or a person, and can be done by the core team or external contributors.
# 3. Feel free to add additional keywords to the list above.

# Inspired by @goldinguy_ in this post: https://goldin.io/blog/stop-using-todo ###
# TODO                        - General Purpose catch-all.
# TODO_COMMUNITY              - A TODO that may be a candidate for outsourcing to the community.
# TODO_DECIDE                 - A TODO indicating we need to make a decision and document it using an ADR in the future; https://github.com/pokt-network/pocket-network-protocol/tree/main/ADRs
# TODO_TECHDEBT               - Not a great implementation, but we need to fix it later.
# TODO_BLOCKER                - BEFORE MAINNET. Similar to TECHDEBT, but of higher priority, urgency & risk prior to the next release
# TODO_QOL                    - AFTER MAINNET. Similar to TECHDEBT, but of lower priority. Doesn't deserve a GitHub Issue but will improve everyone's life.
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
# TODO_ADDTEST / TODO_TEST    - Add more tests for a specific code section
# TODO_FLAKY                  - Signals that the test is flaky and we are aware of it. Provide an explanation if you know why.
# TODO_DEPRECATE              - Code that should be removed in the future
# TODO_RESEARCH               - A non-trivial action item that requires deep research and investigation being next steps can be taken
# TODO_DOCUMENT		          - A comment that involves the creation of a README or other documentation
# TODO_BUG                    - There is a known existing bug in this code
# TODO_NB                     - An important note to reference later
# TODO_DISCUSS_IN_THIS_COMMIT - SHOULD NEVER BE COMMITTED TO MASTER. It is a way for the reviewer of a PR to start / reply to a discussion.
# TODO_IN_THIS_COMMIT         - SHOULD NEVER BE COMMITTED TO MASTER. It is a way to start the review process while non-critical changes are still in progress


# Define shared variable for the exclude parameters
EXCLUDE_GREP = --exclude-dir={.git,vendor,./docusaurus,.vscode,.idea} --exclude={Makefile,reviewdog.yml,*.pb.go,*.pulsar.go}

.PHONY: todo_list
todo_list: ## List all the TODOs in the project (excludes vendor and prototype directories)
	grep -r $(EXCLUDE_GREP) TODO . | grep -v 'TODO()'

.PHONY: todo_count
todo_count: ## Print a count of all the TODOs in the project
	grep -r $(EXCLUDE_GREP) TODO . | grep -v 'TODO()' | wc -l

.PHONY: todo_this_commit
todo_this_commit: ## List all the TODOs needed to be done in this commit
	grep -r $(EXCLUDE_GREP) TODO_IN_THIS .| grep -v 'TODO()'


####################
###   Gateways   ###
####################

.PHONY: gateway_list
gateway_list: ## List all the staked gateways
	poktrolld --home=$(POKTROLLD_HOME) q gateway list-gateway --node $(POCKET_NODE)

.PHONY: gateway_stake
gateway_stake: ## Stake tokens for the gateway specified (must specify the gateway env var)
	poktrolld --home=$(POKTROLLD_HOME) tx gateway stake-gateway -y --config $(POKTROLLD_HOME)/config/$(STAKE) --keyring-backend test --from $(GATEWAY) --node $(POCKET_NODE)

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
	poktrolld --home=$(POKTROLLD_HOME) tx gateway unstake-gateway -y --keyring-backend test --from $(GATEWAY) --node $(POCKET_NODE)

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
	poktrolld --home=$(POKTROLLD_HOME) tx application stake-application -y --config $(POKTROLLD_HOME)/config/$(SERVICES) --keyring-backend test --from $(APP) --node $(POCKET_NODE)

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
	poktrolld --home=$(POKTROLLD_HOME) tx application unstake-application -y --keyring-backend test --from $(APP) --node $(POCKET_NODE)

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
	poktrolld --home=$(POKTROLLD_HOME) tx supplier stake-supplier -y --config $(POKTROLLD_HOME)/config/$(SERVICES) --keyring-backend test --from $(SUPPLIER) --node $(POCKET_NODE)

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

###############
### TestNet ###
###############

.PHONY: testnet_supplier_list
testnet_supplier_list: ## List all the staked supplier on TestNet
	poktrolld q supplier list-supplier --node=$(TESTNET_RPC)

.PHONY: testnet_gateway_list
testnet_gateway_list: ## List all the staked gateways on TestNet
	poktrolld q gateway list-gateway --node=$(TESTNET_RPC)

.PHONY: testnet_app_list
testnet_app_list: ## List all the staked applications on TestNet
	poktrolld q application list-application --node=$(TESTNET_RPC)

.PHONY: testnet_consensus_params
testnet_consensus_params: ## Output consensus parameters
	poktrolld q consensus params --node=$(TESTNET_RPC)

.PHONY: testnet_gov_params
testnet_gov_params: ## Output gov parameters
	poktrolld q gov params --node=$(TESTNET_RPC)

.PHONY: testnet_status
testnet_status: ## Output status of the RPC node (most likely a validator)
	poktrolld status --node=$(TESTNET_RPC) | jq

.PHONY: testnet_height
testnet_height: ## Height of the network from the RPC node point of view
	poktrolld status --node=$(TESTNET_RPC) | jq ".sync_info.latest_block_height"

################
### Accounts ###
################

.PHONY: acc_balance_query
acc_balance_query: ## Query the balance of the account specified (make acc_balance_query ACC=pokt...)
	@echo "~ Balances ~"
	poktrolld --home=$(POKTROLLD_HOME) q bank balances $(ACC) --node $(POCKET_NODE)
	@echo "~ Spendable Balances ~"
	@echo "Querying spendable balance for $(ACC)"
	poktrolld --home=$(POKTROLLD_HOME) q bank spendable-balances $(ACC) --node $(POCKET_NODE)

.PHONY: acc_balance_query_modules
acc_balance_query_modules: ## Query the balance of the network level module accounts
	@echo "### Application ###"
	make acc_balance_query ACC=$(APPLICATION_MODULE_ADDRESS)
	@echo "### Supplier ###"
	make acc_balance_query ACC=$(SUPPLIER_MODULE_ADDRESS)
	@echo "### Gateway ###"
	make acc_balance_query ACC=$(GATEWAY_MODULE_ADDRESS)
	@echo "### Service ###"
	make acc_balance_query ACC=$(SERVICE_MODULE_ADDRESS)

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
# TODO_TECHDEBT: One of the accounts involved in this command always errors
# so we need to understand why and fix it.
.PHONY: acc_initialize_pubkeys
acc_initialize_pubkeys: ## Make sure the account keeper has public keys for all available accounts
	$(eval ADDRESSES=$(shell make -s ignite_acc_list | grep pokt | awk '{printf "%s ", $$2}' | sed 's/.$$//'))
	$(foreach addr, $(ADDRESSES),\
		echo $(addr);\
		poktrolld tx bank send \
			$(addr) $(PNF_ADDRESS) 1000upokt \
			--yes \
			--home=$(POKTROLLD_HOME) \
			--node $(POCKET_NODE);)

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
	@echo "|     	DEVELOPER_TIP: If you're operating off defaults, you'll likely need to update to 3     |"
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
	@echo "|     	'make test_all' once or twice more                                                     |"
	@echo "|     3. If the same error persists, isolate it with 'go test -v ./path/to/failing/module       |"
	@echo "|                                                                                               |"
	@echo "+-----------------------------------------------------------------------------------------------+"

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

# TODO_CONSIDERATION: additional factoring (e.g. POKTROLLD_FLAGS).
PARAM_FLAGS = --home=$(POKTROLLD_HOME) --keyring-backend test --from $(PNF_ADDRESS) --node $(POCKET_NODE)

### Tokenomics Module Params ###
.PHONY: update_tokenomics_params_all
params_update_tokenomics_all: ## Update the tokenomics module params
	poktrolld tx authz exec ./tools/scripts/params/tokenomics_all.json $(PARAM_FLAGS)

.PHONY: params_update_tokenomics_compute_units_to_tokens_multiplier
params_update_tokenomics_compute_units_to_tokens_multiplier: ## Update the tokenomics module compute_units_to_tokens_multiplier param
	poktrolld tx authz exec ./tools/scripts/params/tokenomics_compute_units_to_tokens_multiplier.json $(PARAM_FLAGS)

### Proof Module Params ###
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

### Shared Module Params ###
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

.PHONY: params_query_all
params_query_all: check_jq ## Query the params from all available modules
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

.PHONY: ignite_poktrolld_build
ignite_poktrolld_build: check_go_version check_ignite_version ## Build the poktrolld binary using Ignite
	ignite chain build --skip-proto --debug -v -o $(shell go env GOPATH)/bin

.PHONY: ignite_openapi_gen
ignite_openapi_gen: ## Generate the OpenAPI spec for the Ignite API
	ignite generate openapi --yes

##################
### CI Helpers ###
##################

.PHONY: trigger_ci
trigger_ci: ## Trigger the CI pipeline by submitting an empty commit; See https://github.com/pokt-network/pocket/issues/900 for details
	git commit --allow-empty -m "Empty commit"
	git push

.PHONY: ignite_install
ignite_install: ## Install ignite. Used by CI and heighliner.
	# Determine if sudo is available and use it if it is
	if command -v sudo &>/dev/null; then \
		SUDO="sudo"; \
	else \
		SUDO=""; \
	fi; \
	echo "Downloading Ignite CLI..."; \
	wget https://github.com/ignite/cli/releases/download/v28.3.0/ignite_28.3.0_$(OS)_$(ARCH).tar.gz; \
	echo "Extracting Ignite CLI..."; \
	tar -xzf ignite_28.3.0_$(OS)_$(ARCH).tar.gz; \
	echo "Moving Ignite CLI to /usr/local/bin..."; \
	$$SUDO mv ignite /usr/local/bin/ignite; \
	echo "Cleaning up..."; \
	rm ignite_28.3.0_$(OS)_$(ARCH).tar.gz; \
	ignite version

.PHONY: ignite_update_ldflags
ignite_update_ldflags:
	yq eval '.build.ldflags = ["-X main.Version=$(VERSION)", "-X main.Date=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"]' -i config.yml

.PHONY: ignite_release
ignite_release: ## Builds production binaries
	ignite chain build --release -t linux:amd64 -t linux:arm64 -t darwin:amd64 -t darwin:arm64

.PHONY: ignite_release_extract_binaries
ignite_release_extract_binaries: ## Extracts binaries from the release archives
	mkdir -p release_binaries

	for archive in release/*.tar.gz; do \
		binary_name=$$(basename "$$archive" .tar.gz); \
		tar -zxvf "$$archive" -C release_binaries "poktrolld"; \
		mv release_binaries/poktrolld "release_binaries/$$binary_name"; \
	done

#####################
### Documentation ###
#####################

.PHONY: go_docs
go_docs: check_godoc ## Generate documentation for the project
	echo "Visit http://localhost:6060/pkg/github.com/pokt-network/poktroll/"
	godoc -http=:6060

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

#############################
### Grove Gateway Helpers ###
#############################

.PHONY: grove_staging_eth_block_height
grove_staging_eth_block_height: ## Sends a relay through the staging grove gateway to the eth-mainnet chain. Must have GROVE_STAGING_PORTAL_APP_ID environment variable set.
	curl $(GROVE_GATEWAY_STAGING_ETH_MAINNET)/v1/$(GROVE_STAGING_PORTAL_APP_ID) \
		-H 'Content-Type: application/json' \
		-H 'Protocol: shannon-testnet' \
		--data $(JSON_RPC_DATA_ETH_BLOCK_HEIGHT)

#################
### Catch all ###
#################

%:
	@echo "Error: target '$@' not found."
	@exit 1
