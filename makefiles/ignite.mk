##########################
### Ignite Configuration ###
##########################

# Build configuration
BUILD_TAGS ?= ethereum_secp256k1
IGNITE_ENV ?= CGO_ENABLED=1 CGO_CFLAGS="-Wno-implicit-function-declaration"
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

###############################
### Cosmovisor Dependencies ###
###############################

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