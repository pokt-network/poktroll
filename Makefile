.SILENT:

SHELL = /bin/sh

POKTROLLD_HOME ?= ./localnet/poktrolld
POCKET_NODE ?= tcp://127.0.0.1:26657 # The pocket node (validator in the localnet context)
TESTNET_RPC ?= https://testnet-validated-validator-rpc.poktroll.com/ # TestNet RPC endpoint for validator maintained by Grove. Needs to be update if there's another "primary" testnet.
PATH_URL ?= http://localhost:3000
POCKET_ADDR_PREFIX = pokt
LOAD_TEST_CUSTOM_MANIFEST ?= loadtest_manifest_example.yaml

# The domain ending in ".town" is staging, ".city" is production
GROVE_PORTAL_STAGING_ETH_MAINNET = https://eth-mainnet.rpc.grove.town
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
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.60.3 && golangci-lint --version
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

###############
### Linting ###
###############

.PHONY: go_lint
go_lint: ## Run all go linters
	golangci-lint run --timeout 5m --build-tags test

go_imports: check_go_version ## Run goimports on all go files
	go run ./tools/scripts/goimports

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

.PHONY: docs_update_gov_params_page
docs_update_gov_params_page: ## Update the page in Docusaurus documenting all the governance parameters
	go run tools/scripts/docusaurus/generate_docs_params.go

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


###########################
###   Release Helpers   ###
###########################

# List tags: git tag
# Delete tag locally: git tag -d v1.2.3
# Delete tag remotely: git push --delete origin v1.2.3

.PHONY: release_tag_bug_fix
release_tag_bug_fix: ## Tag a new bug fix release (e.g. v1.0.1 -> v1.0.2)
	@$(eval LATEST_TAG=$(shell git tag --sort=-v:refname | head -n 1))
	@$(eval NEW_TAG=$(shell echo $(LATEST_TAG) | awk -F. -v OFS=. '{ $$NF = sprintf("%d", $$NF + 1); print }'))
	@git tag $(NEW_TAG)
	@echo "New bug fix version tagged: $(NEW_TAG)"
	@echo "Run the following commands to push the new tag:"
	@echo "  git push origin $(NEW_TAG)"
	@echo "And draft a new release at https://github.com/pokt-network/poktroll/releases/new"


.PHONY: release_tag_minor_release
release_tag_minor_release: ## Tag a new minor release (e.g. v1.0.0 -> v1.1.0)
	@$(eval LATEST_TAG=$(shell git tag --sort=-v:refname | head -n 1))
	@$(eval NEW_TAG=$(shell echo $(LATEST_TAG) | awk -F. '{$$2 += 1; $$3 = 0; print $$1 "." $$2 "." $$3}'))
	@git tag $(NEW_TAG)
	@echo "New minor release version tagged: $(NEW_TAG)"
	@echo "Run the following commands to push the new tag:"
	@echo "  git push origin $(NEW_TAG)"
	@echo "And draft a new release at https://github.com/pokt-network/poktroll/releases/new"

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
