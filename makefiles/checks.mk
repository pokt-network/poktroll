##############
### Checks ###
##############

# TODO_DOCUMENT: All of the 'check_' helpers can be installed differently depending
# on the user's OS and environment.
# NB: For mac users, you may need to install with the proper linkers: https://github.com/golang/go/issues/65940

.PHONY: check_go_version
check_go_version:
	@GO_VERSION=$$(go version | cut -d " " -f 3 | cut -c 3-) && \
	MAJOR_VERSION=$$(echo $$GO_VERSION | cut -d "." -f 1) && \
	MINOR_VERSION=$$(echo $$GO_VERSION | cut -d "." -f 2) && \
	\
	if [ "$$MAJOR_VERSION" -lt 1 ] || { [ "$$MAJOR_VERSION" -eq 1 ] && [ "$$MINOR_VERSION" -lt 24 ]; }; then \
		echo "Invalid Go version. Expected 1.24.x or newer but found $$GO_VERSION"; \
		exit 1; \
	fi

.PHONY: check_act
# Internal helper: Check if act is installed
check_act:
	@if ! command -v act >/dev/null 2>&1; then \
		echo "❌ Please install act first with 'make install_act'"; \
		exit 1; \
	fi;

.PHONY: check_gh
# Internal helper target - check if 'gh' is installed
check_gh:
	{ \
	if ( ! ( command -v gh >/dev/null )); then \
		echo "Seems like you don't have 'gh' installed. Please visit https://cli.github.com/ before continuing"; \
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

.PHONY: check_kubectl
# Internal helper target - check if kubectl is installed
check_kubectl:
	@if ! command -v kubectl >/dev/null 2>&1; then \
		echo "kubectl is not installed. Make sure you review build/localnet/README.md and docs/development/README.md  before continuing"; \
		exit 1; \
	fi

.PHONY: check_docker_ps
 ## Internal helper target - checks if Docker is running
check_docker_ps: check_docker
	@echo "Checking if Docker is running..."
	@docker ps > /dev/null 2>&1 || (echo "Docker is not running. Please start Docker and try again."; exit 1)

.PHONY: check_godoc
# Internal helper target - check if godoc is installed
check_godoc:
	{ \
	if ( ! ( command -v godoc >/dev/null )); then \
		echo "Seems like you don't have godoc installed. Make sure you install it via 'go install golang.org/x/tools/cmd/godoc@latest' before continuing"; \
		exit 1; \
	fi; \
	}

.PHONY: check_yarn
# Internal helper target - check if yarn is installed
check_yarn:
	{ \
	if ( ! ( command -v yarn >/dev/null )); then \
		echo "Seems like you don't have yarn installed. Make sure you install it before continuing"; \
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
# Internal helper target - check if 'yq' is installed
check_yq:
	{ \
	if ( ! ( command -v yq >/dev/null )); then \
		echo "Seems like you don't have 'yq' installed. Make sure you install it before continuing"; \
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

.PHONY: check_path_up
# Internal helper: Checks if PATH is running at localhost:3069
check_path_up:
	@if ! nc -z localhost 3069 2>/dev/null; then \
		echo "########################################################################"; \
		echo "ERROR: PATH is not running on port 3069"; \
		echo "Please make sure localnet_config.yaml contains at least 1 path gateway and start localnet with:"; \
		echo "  make localnet_up"; \
		echo "########################################################################"; \
		exit 1; \
	else \
		echo "✅ PATH is up and running on port 3069"; \
	fi

.PHONY: check_relay_util
# Internal helper: Checks if relay-util is installed locally
check_relay_util:
	@if ! command -v relay-util &> /dev/null; then \
		echo "####################################################################################################"; \
		echo "Relay Util is not installed." \
		echo "To use any Relay Util make targets to send load testing requests please install Relay Util with:"; \
		echo "go install github.com/commoddity/relay-util/v2@latest"; \
		echo "####################################################################################################"; \
	fi

.PHONY: check_proto_unstable_marshalers
check_proto_unstable_marshalers: ## Check that all protobuf files have the 'stable_marshalers_all' option set to true.
	go run ./tools/scripts/protocheck/cmd unstable

.PHONY: fix_proto_unstable_marshalers
fix_proto_unstable_marshalers: ## Ensure the 'stable_marshaler_all' option is present on all protobuf files.
	go run ./tools/scripts/protocheck/cmd unstable --fix
	${MAKE} proto_regen

.PHONY: check_proto_event_fields
check_proto_event_fields: ## Check that all Event messages only contain primitive fields for optimal disk utilization.
	go run ./tools/scripts/protocheck/cmd event-fields
