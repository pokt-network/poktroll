############################
### Ignite Configuration ###
############################

# ⚠️The crypto backend is a BUILD-TIME configuration ⚠️
#
# The crypto stack is a complex system that involves multiple dependencies:
# - go-dleq: https://github.com/pokt-network/go-dleq
# - ring-go: https://github.com/pokt-network/ring-go
# - shannon-sdk: https://github.com/pokt-network/shannon-sdk
#
# These repos can choose between:
# - CGO enabled cryptography (Decred implementation)
# - CGO disabled cryptography (Ethereum implementation)


# ⚠️ The crypto backend is a BUILD-TIME configuration ⚠️
# CGO=0 uses pure-Go secp256k1 (portable). CGO=1 uses Decred (C-backed).

IGNITE_CMD ?= ignite chain build
IGNITE_BASE_CGO_ENABLED   := CGO_ENABLED=1 CGO_CFLAGS="-Wno-implicit-function-declaration" $(IGNITE_CMD) --build.tags="ethereum_secp256k1"
IGNITE_BASE_CGO_DISABLED  := CGO_ENABLED=0 $(IGNITE_CMD)

# Release targets
LINUX_TARGETS  := -t linux:amd64 -t linux:arm64
DARWIN_TARGETS := -t darwin:amd64 -t darwin:arm64

# On Ubuntu:
# - CGO=0: linux + darwin are OK
# - CGO=1: build linux per-arch with a proper cross-compiler
RELEASE_TARGETS_NOCGO := $(LINUX_TARGETS) $(DARWIN_TARGETS)

# Cross C compilers on Ubuntu runners (install via apt)
CC_LINUX_AMD64 ?= x86_64-linux-gnu-gcc
CC_LINUX_ARM64 ?= aarch64-linux-gnu-gcc

##########################
### Ignite Build Tasks ###
##########################

.PHONY: ignite_build
ignite_build: ignite_check_version ## Build the pocketd binary using Ignite (development mode)
	$(IGNITE_BASE_LOCAL) --skip-proto --debug -v -o ./bin

.PHONY: ignite_pocketd_build
ignite_pocketd_build: check_go_version ignite_check_version ## Build the pocketd binary to GOPATH/bin
	$(IGNITE_BASE_LOCAL) --skip-proto --debug -v -o $(shell go env GOPATH)/bin

.PHONY: ignite_release
ignite_release: ignite_check_version ## Build production binaries for all architectures
	$(IGNITE_BASE_RELEASE) --release $(RELEASE_TARGETS) -o release
	$(MAKE) _ignite_rename_archives

.PHONY: ignite_release_local
ignite_release_local: ignite_check_version ## Build production binary for current architecture only
	$(IGNITE_BASE_LOCAL) --release -o release
	$(MAKE) _ignite_rename_archives

.PHONY: ignite_release_cgo_disabled
ignite_release_cgo_disabled: ignite_check_version ## CGO=0 release with default names (linux + darwin)
	$(IGNITE_BASE_CGO_DISABLED) \
		--release $(RELEASE_TARGETS_NOCGO) \
		-o release
	$(MAKE) _ignite_rename_archives
	# Optional: $(MAKE) ignite_release_repackage

# CGO=1: build per linux arch to set CC explicitly; keep artifacts with "_cgo" suffix.

.PHONY: ignite_release_cgo_enabled_linux_amd64
ignite_release_cgo_enabled_linux_amd64: ignite_check_version ## CGO=1 release for linux/amd64 (_cgo suffix)
	CC=$(CC_LINUX_AMD64) $(IGNITE_BASE_CGO_ENABLED) \
		--release -t linux:amd64 \
		--release.prefix cgo_ \
		-o release
	$(MAKE) _ignite_suffix_cgo

.PHONY: ignite_release_cgo_enabled_linux_arm64
ignite_release_cgo_enabled_linux_arm64: ignite_check_version ## CGO=1 release for linux/arm64 (_cgo suffix)
	CC=$(CC_LINUX_ARM64) $(IGNITE_BASE_CGO_ENABLED) \
		--release -t linux:arm64 \
		--release.prefix cgo_ \
		-o release
	$(MAKE) _ignite_suffix_cgo

.PHONY: ignite_release_cgo_enabled
ignite_release_cgo_enabled: ignite_release_cgo_enabled_linux_amd64 ignite_release_cgo_enabled_linux_arm64

# Aggregate: build both families.
.PHONY: ignite_release
ignite_release: ignite_release_cgo_disabled ignite_release_cgo_enabled

######################################
### Ignite Release Post-Processing ###
######################################

# Rename CGO-enabled tarballs from cgo_poktroll_* to pocket_*_cgo
.PHONY: _ignite_suffix_cgo
_ignite_suffix_cgo:
	@cd release && \
	for f in cgo_poktroll_*.tar.gz; do \
		base_no_pref=$${f#cgo_}; \
		swapped=$${base_no_pref/poktroll/pocket}; \
		mv "$$f" "$${swapped%.tar.gz}_cgo.tar.gz"; \
	done; \
	sha256sum pocket_*.tar.gz > release_checksum || true

# Rename poktroll_* to pocket_* (CGO=0 path and any others that slipped through)
.PHONY: _ignite_rename_archives
_ignite_rename_archives:
	@cd release && for f in poktroll_*.tar.gz; do [ -f "$$f" ] && mv "$$f" "pocket_$${f#poktroll_}" || true; done
	@cd release && if [ -f release_checksum ]; then \
		sed 's/poktroll/pocket/g' release_checksum > release_checksum.tmp && \
		mv release_checksum.tmp release_checksum; \
	fi

# Repackage to contain only pocketd at root, then refresh checksums
.PHONY: ignite_release_repackage
ignite_release_repackage:
	@for archive in release/pocket_*.tar.gz; do \
		if [ -f "$$archive" ]; then \
			tmp=$$(mktemp -d); \
			tar -zxf "$$archive" -C "$$tmp"; \
			find "$$tmp" -name "pocketd" -type f -exec cp {} "$$tmp/pocketd" \; ; \
			tar -czf "$$archive.new" -C "$$tmp" pocketd; \
			mv "$$archive.new" "$$archive"; \
			rm -rf "$$tmp"; \
		fi; \
	done
	@cd release && sha256sum pocket_*.tar.gz > release_checksum

# Extract all archives to release_binaries/<archive base> (Dockerfile.release expects pocket_linux_$ARCH)
.PHONY: ignite_release_extract_binaries
ignite_release_extract_binaries:
	@mkdir -p release_binaries
	@for archive in release/*.tar.gz; do \
		bname=$$(basename "$$archive" .tar.gz); \
		tmp=$$(mktemp -d); \
		tar -zxf "$$archive" -C "$$tmp"; \
		find "$$tmp" -name "pocketd" -type f -exec cp {} "release_binaries/$$bname" \; ; \
		rm -rf "$$tmp"; \
	done

#################################
### Ignite Version Management ###
#################################

.PHONY: ignite_update_ldflags
ignite_update_ldflags:
	@yq eval '.build.ldflags = ["-X main.Version=$(VERSION)", "-X main.Date=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"]' -i config.yml

.PHONY: ignite_check_version
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
ignite_install:
	@if command -v sudo &>/dev/null; then SUDO="sudo"; else SUDO=""; fi; \
	wget https://github.com/ignite/cli/releases/download/v29.0.0-rc.1/ignite_29.0.0-rc.1_$(OS)_$(ARCH).tar.gz; \
	tar -xzf ignite_29.0.0-rc.1_$(OS)_$(ARCH).tar.gz; \
	$$SUDO mv ignite /usr/local/bin/ignite; \
	rm ignite_29.0.0-rc.1_$(OS)_$(ARCH).tar.gz; \
	mkdir -p $(HOME)/.ignite; echo '{"name":"doNotTrackMe","doNotTrack":true}' > $(HOME)/.ignite/anon_identity.json; \
	ignite version

###############################
### Cosmovisor Dependencies ###
###############################

.PHONY: install_cosmovisor
install_cosmovisor:
	go install cosmossdk.io/tools/cosmovisor/cmd/cosmovisor@v1.6.0 && cosmovisor version --cosmovisor-only

.PHONY: cosmovisor_cross_compile
cosmovisor_cross_compile:
	@COSMOVISOR_VERSION="v1.6.0"; \
	PLATFORMS="linux/amd64 linux/arm64"; \
	mkdir -p ./tmp; \
	tmpd=$$(mktemp -d); cd $$tmpd; \
	go mod init temp; \
	go get cosmossdk.io/tools/cosmovisor/cmd/cosmovisor@$$COSMOVISOR_VERSION; \
	for platform in $$PLATFORMS; do \
		OS=$${platform%/*}; ARCH=$${platform#*/}; \
		GOOS=$$OS GOARCH=$$ARCH go build -o $(CURDIR)/tmp/cosmovisor-$$OS-$$ARCH cosmossdk.io/tools/cosmovisor/cmd/cosmovisor; \
	done; \
	cd $(CURDIR); rm -rf $$tmpd; \
	ls -l ./tmp/cosmovisor-*

.PHONY: cosmovisor_clean
cosmovisor_clean:
	rm -f ./tmp/cosmovisor-*