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

.PHONY: check_mockgen
# Internal helper target- Check if mockgen is installed
check_mockgen:
	{ \
	if ( ! ( command -v mockgen >/dev/null )); then \
		echo "Seems like you don't have `mockgen` installed. Please visit https://github.com/golang/mock#installation and follow the instructions to install `mockgen` before continuing"; \
		exit 1; \
	fi; \
	}


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

.PHONY: check_proto_unstable_marshalers
check_proto_unstable_marshalers: ## Check that all protobuf files have the `stable_marshalers_all` option set to true.
	go run ./tools/scripts/protocheck/cmd unstable

.PHONY: fix_proto_unstable_marshalers
fix_proto_unstable_marshalers: ## Ensure the `stable_marshaler_all` option is present on all protobuf files.
	go run ./tools/scripts/protocheck/cmd unstable --fix
	${MAKE} proto_regen


.PHONY: warn_destructive
warn_destructive: ## Print WARNING to the user
	@echo "This is a destructive action that will affect docker resources outside the scope of this repo!"
