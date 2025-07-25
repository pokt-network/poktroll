.SILENT:

SHELL = /bin/sh

POCKETD_HOME ?= ./localnet/pocketd
POCKET_NODE ?= tcp://127.0.0.1:26657 # The pocket node (validator in the localnet context)
DEFAULT_POCKET_NODE_GRPC_ADDR ?= "localhost:9090"
TESTNET_RPC ?= https://testnet-validated-validator-rpc.poktroll.com/ # TestNet RPC endpoint for validator maintained by Grove. Needs to be update if there's another "primary" testnet.
PATH_URL ?= http://localhost:3000
POCKET_ADDR_PREFIX = pokt
LOAD_TEST_CUSTOM_MANIFEST ?= loadtest_manifest_example.yaml

# The domain ending in ".town" is staging, ".city" is production
GROVE_PORTAL_STAGING_ETH_MAINNET = https://eth-mainnet.rpc.grove.town
# JSON RPC data for a test relay request
JSON_RPC_DATA_ETH_BLOCK_HEIGHT = '{"jsonrpc":"2.0","id":"0","method":"eth_blockNumber", "params": []}'

# Onchain module account addresses. Search for `func TestModuleAddress` in the
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

# TODO_TECHDEBT(@okdas): Add other dependencies (ignite, docker, k8s, etc) here
.PHONY: install_ci_deps
install_ci_deps: ## Installs `golangci-lint` and other go tools
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.1.6 && golangci-lint --version
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/mikefarah/yq/v4@latest

.PHONY: install_cosmovisor
install_cosmovisor: ## Installs `cosmovisor`
	go install cosmossdk.io/tools/cosmovisor/cmd/cosmovisor@v1.6.0 && cosmovisor version --cosmovisor-only

.PHONY: cosmovisor_cross_compile
cosmovisor_cross_compile: # Installs multiple cosmovisor binaries for different platforms (used by Dockerfile.release)
	@COSMOVISOR_VERSION="v1.6.0"; \
	PLATFORMS="linux/amd64 linux/arm64"; \
	mkdir -p ./tmp; \
	echo "Fetching Cosmovisor source..."; \
	temp_dir=$$(mktemp -d); \
	cd $$temp_dir; \
	go mod init temp; \
	go get cosmossdk.io/tools/cosmovisor/cmd/cosmovisor@$$COSMOVISOR_VERSION; \
	for platform in $$PLATFORMS; do \
		OS=$${platform%/*}; \
		ARCH=$${platform#*/}; \
		echo "Compiling for $$OS/$$ARCH..."; \
		GOOS=$$OS GOARCH=$$ARCH go build -o $(CURDIR)/tmp/cosmovisor-$$OS-$$ARCH cosmossdk.io/tools/cosmovisor/cmd/cosmovisor; \
	done; \
	cd $(CURDIR); \
	rm -rf $$temp_dir; \
	echo "Compilation complete. Binaries are in ./tmp/"; \
	ls -l ./tmp/cosmovisor-*

.PHONY: cosmovisor_clean
cosmovisor_clean:
	rm -f ./tmp/cosmovisor-*

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

#######################
### Proto  Helpers ####
#######################

.PHONY: proto_regen
proto_regen: ## Regenerate protobuf artifacts
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

###############
### Linting ###
###############

.PHONY: go_lint
go_lint: ## Run all go linters
	golangci-lint run --timeout 5m --build-tags test

go_imports: check_go_version ## Run goimports on all go files
	go run ./tools/scripts/goimports

# DEV_NOTE: Add `-v` flag to `go generate` to see which files are being generated.
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
	go generate ./x/migration/types/
	find . -name interface.go | xargs -I {} go generate {}

.PHONY: go_testgen_fixtures
go_testgen_fixtures: ## Generate fixture data for unit tests
	go generate ./pkg/relayer/miner/miner_test.go

.PHONY: go_testgen_accounts
go_testgen_accounts: ## Generate test accounts for usage in test environments
	go generate ./testutil/testkeyring/keyring.go

.PHONY: go_develop
go_develop: ignite_check_version proto_regen go_mockgen ## Generate protos and mocks

.PHONY: go_develop_and_test
go_develop_and_test: go_develop test_all ## Generate protos, mocks and run all tests

################
### Accounts ###
################

.PHONY: acc_balance_query
acc_balance_query: ## Query the balance of the account specified (make acc_balance_query ACC=pokt...)
	@echo "~ Balances ~"
	pocketd --home=$(POCKETD_HOME) q bank balances $(ACC) --node $(POCKET_NODE)
	@echo "~ Spendable Balances ~"
	@echo "Querying spendable balance for $(ACC)"
	pocketd --home=$(POCKETD_HOME) q bank spendable-balances $(ACC) --node $(POCKET_NODE)

.PHONY: acc_balance_query_modules
acc_balance_query_modules: ## Query the balance of the network level module accounts
	@echo "### Application Module ###\n"
	make acc_balance_query ACC=$(APPLICATION_MODULE_ADDRESS)
	@echo "### Supplier Module ###\n"
	make acc_balance_query ACC=$(SUPPLIER_MODULE_ADDRESS)
	@echo "### Gateway Module ###\n"
	make acc_balance_query ACC=$(GATEWAY_MODULE_ADDRESS)
	@echo "### Service Module ###\n"
	make acc_balance_query ACC=$(SERVICE_MODULE_ADDRESS)

.PHONY: acc_balance_query_app1
acc_balance_query_app1: ## Query the balance of app1
	APP1=$$(make pocketd_addr ACC_NAME=app1) && \
	make acc_balance_query ACC=$$APP1

.PHONY: acc_balance_total_supply
acc_balance_total_supply: ## Query the total supply of the network
	pocketd --home=$(POCKETD_HOME) q bank total --node $(POCKET_NODE)

# NB: Ignite does not populate `pub_key` in `accounts` within `genesis.json` leading
# to queries like this to fail: `pocketd query account pokt1<addr> --node $(POCKET_NODE).
# We attempted using a `tx multi-send` from the `faucet` to all accounts, but
# that also did not solve this problem because the account itself must sign the
# transaction for its public key to be populated in the account keeper. As such,
# the solution is to send funds from every account in genesis to some address
# (PNF was selected ambiguously) to make sure their public keys are populated.
.PHONY: acc_initialize_pubkeys
acc_initialize_pubkeys: ## Make sure the account keeper has public keys for all available accounts
	$(eval ADDRESSES=$(shell make -s ignite_acc_list | grep pokt | awk '{printf "%s ", $$2}' | sed 's/.$$//'))
	$(foreach addr, $(ADDRESSES),\
		echo $(addr);\
		pocketd tx bank send \
			$(addr) $(PNF_ADDRESS) 1000upokt \
			--yes \
			--home=$(POCKETD_HOME) \
			--node $(POCKET_NODE);)

######################
### Ignite Helpers ###
######################

# TODO_TECHDEBT(@olshansk): Change this to pocketd keys list
.PHONY: ignite_acc_list
ignite_acc_list: ## List all the accounts in LocalNet
	ignite account list --keyring-dir=$(POCKETD_HOME) --keyring-backend test --address-prefix $(POCKET_ADDR_PREFIX)

.PHONY: ignite_pocketd_build
ignite_pocketd_build: check_go_version ignite_check_version ## Build the pocketd binary using Ignite
	ignite chain build --skip-proto --debug -v -o $(shell go env GOPATH)/bin

.PHONY: ignite_openapi_gen
ignite_openapi_gen: ignite_check_version ## Generate the OpenAPI spec natively and process the output
	ignite generate openapi --yes
	$(MAKE) process_openapi

.PHONY: ignite_openapi_gen_docker
ignite_openapi_gen_docker: ## Generate the OpenAPI spec using Docker and process the output; workaround due to https://github.com/ignite/cli/issues/4495
	docker build -f ./proto/Dockerfile.ignite -t ignite-openapi .
	docker run --rm -v "$(PWD):/workspace" ignite-openapi
	$(MAKE) process_openapi

.PHONY: process_openapi
process_openapi: ## Ensure OpenAPI JSON and YAML files are properly formatted
	# The original command incorrectly outputs a JSON-formatted file with a .yml extension.
	# This fixes the issue by properly converting the JSON to a valid YAML format.
	mv docs/static/openapi.yml docs/static/openapi.json
	yq -o=json '.' docs/static/openapi.json -I=4 > docs/static/openapi.json.tmp && mv docs/static/openapi.json.tmp docs/static/openapi.json
	yq -P -o=yaml '.' docs/static/openapi.json > docs/static/openapi.yml

##################
### CI Helpers ###
##################


.PHONY: trigger_ci
trigger_ci: ## Trigger the CI pipeline by submitting an empty commit; See https://github.com/pokt-network/pocket/issues/900 for details
	git commit --allow-empty -m "Empty commit"
	git push

.PHONY: ignite_check_version
# Internal helper target - check ignite version
ignite_check_version:
	@version=$$(ignite version 2>&1 | awk -F':' '/Ignite CLI version/ {gsub(/^[ \t]+/, "", $$2); print $$2}'); \
	if [ "$$version" = "" ]; then \
		echo "Error: Ignite CLI not found."; \
		echo "Please install it via Homebrew (recommended) or make ignite_install." ; \
		echo "For Homebrew installation, follow: https://docs.ignite.com/welcome/install" ; \
		exit 1 ; \
	fi ; \
	if [ "$$(printf "v29\n$$version" | sort -V | head -n1)" != "v29" ]; then \
		echo "Error: Version $$version is less than v29. Please update Ignite via Homebrew or make ignite_install." ; \
		echo "For Homebrew installation, follow: https://docs.ignite.com/welcome/install" ; \
		exit 1 ; \
	fi

.PHONY: ignite_install
ignite_install: ## Install ignite. Used by CI and heighliner.
	# Determine if sudo is available and use it if it is
	if command -v sudo &>/dev/null; then \
		SUDO="sudo"; \
	else \
		SUDO=""; \
	fi; \
	echo "Downloading Ignite CLI..."; \
	wget https://github.com/ignite/cli/releases/download/v29.0.0-rc.1/ignite_29.0.0-rc.1_$(OS)_$(ARCH).tar.gz; \
	echo "Extracting Ignite CLI..."; \
	tar -xzf ignite_29.0.0-rc.1_$(OS)_$(ARCH).tar.gz; \
	echo "Moving Ignite CLI to /usr/local/bin..."; \
	$$SUDO mv ignite /usr/local/bin/ignite; \
	echo "Cleaning up..."; \
	rm ignite_29.0.0-rc.1_$(OS)_$(ARCH).tar.gz; \
	echo "Configuring ignite so it doesn't block CI by asking for tracking consent..."; \
	mkdir -p $(HOME)/.ignite; \
	echo '{"name":"doNotTrackMe","doNotTrack":true}' > $(HOME)/.ignite/anon_identity.json; \
	ignite version

#######################
### Keyring Helpers ###
#######################

.PHONY: pocketd_addr
pocketd_addr: ## Retrieve the address for an account by ACC_NAME
	@echo $(shell pocketd --home=$(POCKETD_HOME) keys show -a $(ACC_NAME))

.PHONY: pocketd_key
pocketd_key: ## Retrieve the private key for an account by ACC_NAME
	@echo $(shell pocketd --home=$(POCKETD_HOME) keys export --unsafe --unarmored-hex $(ACC_NAME))

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

############################
### Grove Portal Helpers ###
############################

.PHONY: grove_staging_eth_block_height
grove_staging_eth_block_height: ## Sends a relay through the staging grove gateway to the eth-mainnet chain. Must have GROVE_STAGING_PORTAL_APP_ID environment variable set.
	curl $(GROVE_PORTAL_STAGING_ETH_MAINNET)/v1/$(GROVE_STAGING_PORTAL_APP_ID) \
		-H 'Content-Type: application/json' \
		-H 'Protocol: shannon-testnet' \
		--data $(JSON_RPC_DATA_ETH_BLOCK_HEIGHT)

#################
### Catch all ###
#################

%:
	@echo "Error: target '$@' not found."
	@exit 1

###############
### Imports ###
###############

include ./makefiles/colors.mk
include ./makefiles/warnings.mk
include ./makefiles/todos.mk
include ./makefiles/checks.mk
include ./makefiles/tests.mk
include ./makefiles/localnet.mk
include ./makefiles/query.mk
include ./makefiles/testnet.mk
include ./makefiles/params.mk
include ./makefiles/applications.mk
include ./makefiles/suppliers.mk
include ./makefiles/gateways.mk
include ./makefiles/session.mk
include ./makefiles/claims.mk
include ./makefiles/relay.mk
include ./makefiles/ping.mk
include ./makefiles/migrate.mk
include ./makefiles/claudesync.mk
include ./makefiles/telegram.mk
include ./makefiles/docs.mk
include ./makefiles/release.mk
include ./makefiles/tools.mk