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

# Patterns for classified help categories
HELP_PATTERNS := \
	'^(help|help-params|help-unclassified|list):' \
	'^(ignite_build|ignite_pocketd_build|ignite_serve|ignite_serve_reset|ignite_release.*|cosmovisor_start_node):' \
	'^(go_develop|go_develop_and_test|proto_regen|go_mockgen|go_testgen_fixtures|go_testgen_accounts|go_imports):' \
	'^(test_all|test_unit|test_e2e|test_integration|test_timing|test_govupgrade|test_e2e_relay|go_test_verbose|go_test):' \
	'^(go_lint|go_vet|go_sec|gosec_version_fix|check_todos):' \
	'^(localnet_up|localnet_up_quick|localnet_down|localnet_regenesis|localnet_cancel_upgrade|localnet_show_upgrade_plan):' \
	'^testnet_.*:' \
	'^(acc_.*|pocketd_addr|pocketd_key):' \
	'^query_.*:' \
	'^app_.*:' \
	'^supplier_.*:' \
	'^gateway_.*:' \
	'^(relay_.*|claim_.*|ping_.*):' \
	'^session_.*:' \
	'^ibc_.*:' \
	'^release_.*:' \
	'^docker_test_.*:' \
	'^(go_docs|docusaurus_.*|gen_.*_docs):' \
	'^(install_.*|check_.*|grove_.*|act_.*|trigger_ci|docker_wipe):' \
	'^telegram_.*:' \
	'^claudesync_.*:' \
	'^params_.*:'

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

########################
### Makefile Helpers ###
########################

.PHONY: prompt_user
# Internal helper target - prompt the user before continuing
prompt_user:
	@echo "Are you sure? [y/N] " && read ans && [ $${ans:-N} = y ]

.PHONY: help
.DEFAULT_GOAL := help
help: ## Prints all the targets in all the Makefiles
	@echo ""
	@echo "$(BOLD)$(CYAN)🚀 Poktroll Makefile Targets$(RESET)"
	@echo ""
	@echo "$(BOLD)=== 📋 Information & Discovery ===$(RESET)"
	@grep -h -E '^(help|help-params|help-unclassified|list):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 🔨 Build & Run ===$(RESET)"
	@grep -h -E '^(ignite_build|ignite_pocketd_build|ignite_serve|ignite_serve_reset|ignite_release.*|cosmovisor_start_node):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== ⚙️ Development ===$(RESET)"
	@grep -h -E '^(go_develop|go_develop_and_test|proto_regen|go_mockgen|go_testgen_fixtures|go_testgen_accounts|go_imports):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 🧪 Testing ===$(RESET)"
	@grep -h -E '^(test_all|test_unit|test_e2e|test_integration|test_timing|test_govupgrade|test_e2e_relay|go_test_verbose|go_test):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== ✅ Linting & Quality ===$(RESET)"
	@grep -h -E '^(go_lint|go_vet|go_sec|gosec_version_fix|check_todos):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 🌐 LocalNet Operations ===$(RESET)"
	@grep -h -E '^(localnet_up|localnet_up_quick|localnet_down|localnet_regenesis|localnet_cancel_upgrade|localnet_show_upgrade_plan):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 🔗 TestNet Operations ===$(RESET)"
	@grep -h -E '^(testnet_.*):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 💰 Accounts & Balances ===$(RESET)"
	@grep -h -E '^(acc_.*|pocketd_addr|pocketd_key):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 📊 Query Commands ===$(RESET)"
	@grep -h -E '^(query_.*):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 🛡️ Applications ===$(RESET)"
	@grep -h -E '^(app_.*):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 🏭 Suppliers ===$(RESET)"
	@grep -h -E '^(supplier_.*):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 🌉 Gateways ===$(RESET)"
	@grep -h -E '^(gateway_.*):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 🔧 Services ===$(RESET)"
	@grep -h -E '^(service_.*):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 📡 Relays & Claims ===$(RESET)"
	@grep -h -E '^(relay_.*|claim_.*|ping_.*):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 📜 Sessions ===$(RESET)"
	@grep -h -E '^(session_.*):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 🔄 IBC ===$(RESET)"
	@grep -h -E '^(ibc_.*):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 🚢 Release Management ===$(RESET)"
	@grep -h -E '^(release_.*):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 🐳 Docker Testing ===$(RESET)"
	@grep -h -E '^(docker_test_.*):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 📚 Documentation ===$(RESET)"
	@grep -h -E '^(go_docs|docusaurus_.*|gen_.*_docs):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 🔧 Tools & Utilities ===$(RESET)"
	@grep -h -E '^(install_.*|check_.*|grove_.*|act_.*|trigger_ci|docker_wipe):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 📱 Telegram ===$(RESET)"
	@grep -h -E '^(telegram_.*):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 🤖 AI & Sync ===$(RESET)"
	@grep -h -E '^(claudesync_.*):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)💡 More Commands:$(RESET)"
	@echo "   $(CYAN)make help-params$(RESET)         - Parameter management commands"
	@echo "   $(CYAN)make help-unclassified$(RESET)   - Show unclassified targets"
	@echo ""

.PHONY: help-params
help-params: ## Show parameter management commands
	@echo ""
	@echo "$(BOLD)$(CYAN)⚙️ Parameter Management Commands$(RESET)"
	@echo ""
	@echo "$(BOLD)=== 🌐 All Modules ===$(RESET)"
	@grep -h -E '^(params_query_all):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 🏛️ Cosmos Modules ===$(RESET)"
	@grep -h -E '^(params_(auth|bank|consensus|crisis|distribution|gov|mint|protocolpool|slashing|staking)_.*):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 💰 Tokenomics ===$(RESET)"
	@grep -h -E '^(params_tokenomics_.*):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 🔧 Service ===$(RESET)"
	@grep -h -E '^(params_service_.*):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 🔍 Proof ===$(RESET)"
	@grep -h -E '^(params_proof_.*):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 🤝 Shared ===$(RESET)"
	@grep -h -E '^(params_shared_.*):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 🌉 Gateway ===$(RESET)"
	@grep -h -E '^(params_gateway_.*):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 🛡️ Application ===$(RESET)"
	@grep -h -E '^(params_application_.*):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 🏭 Supplier ===$(RESET)"
	@grep -h -E '^(params_supplier_.*):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 📜 Session ===$(RESET)"
	@grep -h -E '^(params_session_.*):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 🔄 Migration ===$(RESET)"
	@grep -h -E '^(params_migration_.*):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD)=== 🏛️ Consensus ===$(RESET)"
	@grep -h -E '^(params_consensus_.*):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'
	@echo ""

.PHONY: help-unclassified
help-unclassified: ## Show all unclassified targets
	@echo ""
	@echo "$(BOLD)$(CYAN)📦 Unclassified Targets$(RESET)"
	@echo ""
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) ./makefiles/*.mk 2>/dev/null | sed 's/:.*//g' | sort -u > /tmp/all_targets.txt
	@grep -h -E '^(help|help-params|help-unclassified|list|ignite_build|ignite_pocketd_build|ignite_serve|ignite_serve_reset|ignite_release.*|cosmovisor_start_node|go_develop|go_develop_and_test|proto_regen|go_mockgen|go_testgen_fixtures|go_testgen_accounts|go_imports|test_all|test_unit|test_e2e|test_integration|test_timing|test_govupgrade|test_e2e_relay|go_test_verbose|go_test|go_lint|go_vet|go_sec|gosec_version_fix|check_todos|localnet_up|localnet_up_quick|localnet_down|localnet_regenesis|localnet_cancel_upgrade|localnet_show_upgrade_plan|testnet_.*|acc_.*|pocketd_addr|pocketd_key|query_.*|app_.*|supplier_.*|gateway_.*|service_.*|relay_.*|claim_.*|ping_.*|session_.*|ibc_.*|release_.*|docker_test_.*|go_docs|docusaurus_.*|gen_.*_docs|install_.*|check_.*|grove_.*|act_.*|trigger_ci|docker_wipe|telegram_.*|claudesync_.*|params_.*):.*?## .*$$' $(MAKEFILE_LIST) ./makefiles/*.mk 2>/dev/null | sed 's/:.*//g' | sort -u > /tmp/classified_targets.txt
	@comm -23 /tmp/all_targets.txt /tmp/classified_targets.txt | while read target; do \
		grep -h -E "^$$target:.*?## .*\$$" $(MAKEFILE_LIST) ./makefiles/*.mk 2>/dev/null | head -1 | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-40s$(RESET) %s\n", $$1, $$2}'; \
	done
	@rm -f /tmp/all_targets.txt /tmp/classified_targets.txt
	@echo ""

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
	APP1=$$(make -s pocketd_addr ACC_NAME=app1) && \
	make -s acc_balance_query ACC=$$APP1

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

##################
### CI Helpers ###
##################

.PHONY: trigger_ci
trigger_ci: ## Trigger the CI pipeline by submitting an empty commit; See https://github.com/pokt-network/pocket/issues/900 for details
	git commit --allow-empty -m "Empty commit"
	git push



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

###############################
###  Global Error Handling  ###
###############################

# Catch-all for undefined targets - MUST be at END after all includes
%:
	@echo ""
	@echo "$(RED)❌ Error: Unknown target '$(BOLD)$@$(RESET)$(RED)'$(RESET)"
	@echo ""
	@echo "$(YELLOW)💡 Available targets:$(RESET)"
	@echo "   Run $(CYAN)make help$(RESET) to see all available targets"
	@echo "   Run $(CYAN)make help-unclassified$(RESET) to see unclassified targets"
	@echo ""
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
include ./makefiles/services.mk
include ./makefiles/session.mk
include ./makefiles/claims.mk
include ./makefiles/relay.mk
include ./makefiles/ping.mk
include ./makefiles/migrate.mk
include ./makefiles/claudesync.mk
include ./makefiles/telegram.mk
include ./makefiles/docs.mk
include ./makefiles/ignite.mk
include ./makefiles/release.mk
include ./makefiles/tools.mk
include ./makefiles/ibc.mk

###############################
###  Global Error Handling  ###
###############################

# Catch-all rule for undefined targets
# This must be defined AFTER includes so color variables are available
# and it acts as a fallback for any undefined target
%:
	@echo ""
	@echo "$(RED)❌ Error: Unknown target '$(BOLD)$@$(RESET)$(RED)'$(RESET)"
	@echo ""
	@if echo "$@" | grep -q "^localnet"; then \
		echo "$(YELLOW)💡 Hint: LocalNet targets available:$(RESET)"; \
		echo "   Run: $(CYAN)make help$(RESET) and check the '🌐 LocalNet Operations' section"; \
	elif echo "$@" | grep -q "^testnet"; then \
		echo "$(YELLOW)💡 Hint: TestNet targets available:$(RESET)"; \
		echo "   Run: $(CYAN)make help$(RESET) and check the '🔗 TestNet Operations' section"; \
	elif echo "$@" | grep -q "^app"; then \
		echo "$(YELLOW)💡 Hint: Application targets available:$(RESET)"; \
		echo "   Run: $(CYAN)make help$(RESET) and check the '🛡️ Applications' section"; \
	elif echo "$@" | grep -q "^supplier"; then \
		echo "$(YELLOW)💡 Hint: Supplier targets available:$(RESET)"; \
		echo "   Run: $(CYAN)make help$(RESET) and check the '🏭 Suppliers' section"; \
	elif echo "$@" | grep -q "^gateway"; then \
		echo "$(YELLOW)💡 Hint: Gateway targets available:$(RESET)"; \
		echo "   Run: $(CYAN)make help$(RESET) and check the '🌉 Gateways' section"; \
	elif echo "$@" | grep -q "^test"; then \
		echo "$(YELLOW)💡 Hint: Testing targets available:$(RESET)"; \
		echo "   Run: $(CYAN)make help$(RESET) and check the '🧪 Testing' section"; \
	elif echo "$@" | grep -q "^params"; then \
		echo "$(YELLOW)💡 Hint: Parameter management targets available:$(RESET)"; \
		echo "   Run: $(CYAN)make help-params$(RESET) to see all parameter commands"; \
	elif echo "$@" | grep -q "^ignite"; then \
		echo "$(YELLOW)💡 Hint: Ignite/build targets available:$(RESET)"; \
		echo "   Run: $(CYAN)make help$(RESET) and check the '🔨 Build & Run' section"; \
	elif echo "$@" | grep -q "^go_"; then \
		echo "$(YELLOW)💡 Hint: Go development targets available:$(RESET)"; \
		echo "   Run: $(CYAN)make help$(RESET) and check the '⚙️ Development' or '✅ Linting & Quality' sections"; \
	elif echo "$@" | grep -q "^docker"; then \
		echo "$(YELLOW)💡 Hint: Docker targets available:$(RESET)"; \
		echo "   Run: $(CYAN)make help$(RESET) and check the '🐳 Docker Testing' section"; \
	else \
		echo "$(YELLOW)💡 Available help commands:$(RESET)"; \
		echo "   $(CYAN)make help$(RESET)              - See all available targets"; \
		echo "   $(CYAN)make help-params$(RESET)       - See parameter management commands"; \
		echo "   $(CYAN)make help-unclassified$(RESET) - See uncategorized targets"; \
	fi
	@echo ""
	@exit 1
