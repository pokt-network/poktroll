##########################
### Ignite Configuration ###
##########################

# Build configuration
BUILD_TAGS ?= ethereum_secp256k1
IGNITE_ENV ?= CGO_ENABLED=1
IGNITE_CMD ?= ignite chain build
IGNITE_BASE := $(IGNITE_ENV) $(IGNITE_CMD) --build.tags="$(BUILD_TAGS)"

# Build targets (for release builds)
LINUX_TARGETS := -t linux:amd64 -t linux:arm64
DARWIN_TARGETS := -t darwin:amd64 -t darwin:arm64
RELEASE_TARGETS := $(LINUX_TARGETS) $(DARWIN_TARGETS)

##########################
### Ignite Build Tasks ###
##########################

.PHONY: ignite_build
ignite_build: ignite_check_version ## Build the pocketd binary using Ignite (development mode)
	$(IGNITE_BASE) --skip-proto --debug -v -o .

.PHONY: ignite_pocketd_build
ignite_pocketd_build: check_go_version ignite_check_version ## Build the pocketd binary to GOPATH/bin
	$(IGNITE_BASE) --skip-proto --debug -v -o $(shell go env GOPATH)/bin

.PHONY: ignite_release
ignite_release: ignite_check_version ## Build production binaries for all architectures
	$(IGNITE_BASE) --release $(RELEASE_TARGETS) -o release
	$(MAKE) _ignite_rename_archives

.PHONY: ignite_release_local
ignite_release_local: ignite_check_version ## Build production binary for current architecture only
	$(IGNITE_BASE) --release -o release
	$(MAKE) _ignite_rename_archives

##################################
### Ignite Release Post-Processing ###
##################################

.PHONY: _ignite_rename_archives
# Internal helper: Rename poktroll archives to pocket and update checksums
_ignite_rename_archives:
	@cd release && for f in poktroll_*.tar.gz; do [ -f "$$f" ] && mv "$$f" "pocket_$${f#poktroll_}" || true; done
	@cd release && if [ -f release_checksum ]; then \
		sed 's/poktroll/pocket/g' release_checksum > release_checksum.tmp && \
		mv release_checksum.tmp release_checksum; \
	fi

.PHONY: ignite_release_repackage
ignite_release_repackage: ## Repackage release archives to contain only pocketd binary at root level
	@for archive in release/pocket_*.tar.gz; do \
		if [ -f "$$archive" ]; then \
			binary_name=$$(basename "$$archive" .tar.gz); \
			temp_dir=$$(mktemp -d); \
			tar -zxf "$$archive" -C "$$temp_dir"; \
			find "$$temp_dir" -name "pocketd" -type f -exec cp {} "$$temp_dir/pocketd" \; ; \
			tar -czf "$$archive.new" -C "$$temp_dir" pocketd; \
			mv "$$archive.new" "$$archive"; \
			rm -rf "$$temp_dir"; \
		fi; \
	done
	@cd release && sha256sum pocket_*.tar.gz > release_checksum

.PHONY: ignite_release_extract_binaries
ignite_release_extract_binaries: ## Extract binaries from release archives to release_binaries/
	@mkdir -p release_binaries
	@for archive in release/*.tar.gz; do \
		binary_name=$$(basename "$$archive" .tar.gz); \
		temp_dir=$$(mktemp -d); \
		tar -zxf "$$archive" -C "$$temp_dir"; \
		find "$$temp_dir" -name "pocketd" -type f -exec cp {} "release_binaries/$$binary_name" \; ; \
		rm -rf "$$temp_dir"; \
	done

#################################
### Ignite Version Management ###
#################################

.PHONY: ignite_update_ldflags
ignite_update_ldflags: ## Update build ldflags with version and build date
	@yq eval '.build.ldflags = ["-X main.Version=$(VERSION)", "-X main.Date=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"]' -i config.yml

.PHONY: ignite_check_version
# Internal helper: Check ignite version compatibility
ignite_check_version:
	@version=$$(ignite version 2>&1 | awk -F':' '/Ignite CLI version/ {gsub(/^[ \t]+/, "", $$2); print $$2}'); \
	if [ "$$version" = "" ]; then \
		echo "Error: Ignite CLI not found."; \
		echo "Please install it via Homebrew (recommended) or make ignite_install." ; \
		echo "For Homebrew installation, follow: https://docs.ignite.com/welcome/install" ; \
		exit 1 ; \
	fi ; \
	if [ "$$(printf "v29\\n$$version" | sort -V | head -n1)" != "v29" ]; then \
		echo "Error: Version $$version is less than v29. Please update Ignite via Homebrew or make ignite_install." ; \
		echo "For Homebrew installation, follow: https://docs.ignite.com/welcome/install" ; \
		exit 1 ; \
	fi

.PHONY: ignite_install
ignite_install: ## Install Ignite CLI (used by CI and heighliner)
	@if command -v sudo &>/dev/null; then \
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

##########################
### Ignite Development ###
##########################

.PHONY: ignite_acc_list
ignite_acc_list: ## List all accounts in LocalNet
	@ignite account list --keyring-dir=$(POCKETD_HOME) --keyring-backend test --address-prefix $(POCKET_ADDR_PREFIX)

.PHONY: ignite_openapi_gen
ignite_openapi_gen: ignite_check_version ## Generate OpenAPI spec natively and process output
	@ignite generate openapi --yes
	@$(MAKE) process_openapi

.PHONY: ignite_openapi_gen_docker
ignite_openapi_gen_docker: ## Generate OpenAPI spec using Docker (workaround for ignite/cli#4495)
	@docker build -f ./proto/Dockerfile.ignite -t ignite-openapi .
	@docker run --rm -v "$(PWD):/workspace" ignite-openapi
	@$(MAKE) process_openapi

.PHONY: process_openapi
# Internal helper: Process OpenAPI output to proper JSON/YAML format
process_openapi:
	@# Fix incorrectly named .yml file that contains JSON
	@mv docs/static/openapi.yml docs/static/openapi.json
	@yq -o=json '.' docs/static/openapi.json -I=4 > docs/static/openapi.json.tmp && mv docs/static/openapi.json.tmp docs/static/openapi.json
	@yq -P -o=yaml '.' docs/static/openapi.json > docs/static/openapi.yml