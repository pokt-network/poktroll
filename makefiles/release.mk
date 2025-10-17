#############################################
##          Configuration variables        ##
#############################################

GH_WORKFLOWS := .github/workflows

#####################################
##       CI/CD Workflow Testing    ##
#####################################

.PHONY: check_secrets
# Internal helper: Check if .secrets file exists with valid GITHUB_TOKEN
check_secrets:
	@if [ ! -f .secrets ]; then \
		echo "❌ .secrets file not found!"; \
		echo "Please create a .secrets file with your GitHub token:"; \
		echo "GITHUB_TOKEN=your_github_token"; \
		exit 1; \
	fi
	@if ! grep -q "GITHUB_TOKEN=" .secrets; then \
		echo "❌ GITHUB_TOKEN not found in .secrets file!"; \
		echo "Please add GITHUB_TOKEN to your .secrets file:"; \
		echo "GITHUB_TOKEN=your_github_token"; \
		echo "You can create a token at: https://github.com/settings/tokens"; \
		exit 1; \
	fi
	@if grep -q "GITHUB_TOKEN=$$" .secrets || grep -q "GITHUB_TOKEN=\"\"" .secrets || grep -q "GITHUB_TOKEN=''" .secrets; then \
		echo "❌ GITHUB_TOKEN is empty in .secrets file!"; \
		echo "Please set a valid GitHub token:"; \
		echo "GITHUB_TOKEN=your_github_token"; \
		echo "You can create a token at: https://github.com/settings/tokens"; \
		exit 1; \
	fi

.PHONY: install_act
install_act: ## Install act for local GitHub Actions testing
	@echo "Installing act..."
	@if [ "$$(uname)" = "Darwin" ]; then \
		brew install act; \
	else \
		curl -s https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash; \
	fi
	@echo "✅ act installed successfully"

########################
###   Act Triggers   ###
########################

.PHONY: workflow_test_release_artifacts
workflow_test_release_artifacts: check_act check_secrets ## Test the release artifacts GitHub workflow
	@echo "Testing release artifacts workflow..."
	@act -W $(GH_WORKFLOWS)/release-artifacts.yml workflow_dispatch $(ACT_ARCH_FLAG) -v --secret-file .secrets

###########################
###   Release Helpers   ###
###########################

# Common variables
GITHUB_REPO_URL := https://github.com/pokt-network/poktroll/releases/new
INFO_URL := https://dev.poktroll.com/explore/account_management/pocketd_cli?_highlight=cli

define print_next_steps
	$(call print_info_section,Next Steps)
	@echo "  $(BOLD)1.$(RESET) Push the new tag: $(CYAN)git push origin $(1)$(RESET)"
	@echo "  $(BOLD)2.$(RESET) Draft the release with gh: $(CYAN)gh release create $(1) $(if $(2),--prerelease,) --generate-notes$(RESET)"
	@echo ""
	@repo_url=$$(gh repo view --json url -q .url); \
		echo "$(BOLD)Release URL:$(RESET) $(CYAN)$${repo_url}/releases/tag/$(1)$(RESET)"
	@echo ""
endef

define print_cleanup_commands
	$(call print_info_section,If you need to delete the tag)
	@echo "  $(BOLD)Local:$(RESET) $(CYAN)git tag -d $(1)$(RESET)"
	@echo "  $(BOLD)Remote:$(RESET) $(CYAN)git push origin --delete $(1)$(RESET)"
	@echo ""
endef

define print_additional_info
	$(call print_info_section,Additional Information)
	$(call print_url,$(INFO_URL))
	@echo ""
endef

.PHONY: release_tag_local_testing
release_tag_local_testing: ## Tag a new local testing release (e.g. v1.0.1 -> v1.0.2-test1, v1.0.2-test1 -> v1.0.2-test2)
	@$(eval LATEST_TAG=$(shell git tag --sort=-v:refname | head -n 1))
	@$(eval NEW_TAG=$(shell \
		if [ -z "$(LATEST_TAG)" ]; then \
			echo "v0.1.0-test1"; \
		elif echo "$(LATEST_TAG)" | grep -q -- '-test'; then \
			BASE_TAG=$$(echo "$(LATEST_TAG)" | sed 's/-test[0-9]*//'); \
			LAST_TEST_NUM=$$(echo "$(LATEST_TAG)" | sed -E 's/.*-test([0-9]+)/\1/'); \
			NEXT_TEST_NUM=$$(($$LAST_TEST_NUM + 1)); \
			echo "$${BASE_TAG}-test$${NEXT_TEST_NUM}"; \
		else \
			BASE_TAG=$$(echo "$(LATEST_TAG)" | awk -F. -v OFS=. '{$$NF = sprintf("%d", $$NF + 1); print}'); \
			echo "$${BASE_TAG}-test1"; \
		fi))
	@git tag $(NEW_TAG)
	$(call print_success,Local testing version tagged: $(NEW_TAG))
	$(call print_next_steps,$(NEW_TAG))
	$(call print_additional_info)

.PHONY: release_tag_dev
release_tag_dev: ## Tag a new dev release for unmerged PRs (e.g. v1.0.1-dev-feat-xyz, v1.0.1-dev-pr-123)
	@$(eval LATEST_TAG=$(shell git tag --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$$' | head -n 1))
	@$(eval CURRENT_BRANCH=$(shell git rev-parse --abbrev-ref HEAD))
	@$(eval SHORT_COMMIT=$(shell git rev-parse --short HEAD))
	@if [ "$(CURRENT_BRANCH)" = "main" ] || [ "$(CURRENT_BRANCH)" = "master" ]; then \
		$(call print_warning,Cannot create dev tag from main/master branch. Switch to a feature branch first.); \
		exit 1; \
	fi
	@if [ -n "$$(git status --porcelain)" ]; then \
		$(call print_warning,Working directory has uncommitted changes.); \
		read -p "Continue anyway? (y/N): " confirm; \
		if [ "$$confirm" != "y" ] && [ "$$confirm" != "Y" ]; then \
			echo "Aborted."; \
			exit 1; \
		fi; \
	fi
	@$(eval BRANCH_CLEAN=$(shell echo $(CURRENT_BRANCH) | sed 's/[^a-zA-Z0-9-]/-/g' | sed 's/--*/-/g' | sed 's/^-\|-$$//g'))
	@$(eval NEW_TAG=$(LATEST_TAG)-dev-$(BRANCH_CLEAN)-$(SHORT_COMMIT))
	@git tag $(NEW_TAG)
	$(call print_success,Dev version tagged: ${CYAN}$(NEW_TAG)${RESET})
	@echo "$(BOLD)Branch:$(RESET) $(CYAN)$(CURRENT_BRANCH)$(RESET)"
	@echo "$(BOLD)Commit:$(RESET) $(CYAN)$(SHORT_COMMIT)$(RESET)"
	@echo ""
	$(call print_next_steps,$(NEW_TAG),pre-release)
	$(call print_cleanup_commands,$(NEW_TAG))
	$(call print_additional_info)

.PHONY: release_tag_rc
release_tag_rc: ## Tag a new rc release (e.g. v0.1.28 -> v0.1.29-rc1, v0.1.29-rc1 -> v0.1.29-rc2)
	@$(eval LATEST_TAG=$(shell git tag --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$$' | head -n 1))
	@$(eval NEXT_VERSION=$(shell echo $(LATEST_TAG) | awk -F. -v OFS=. '{ $$NF = sprintf("%d", $$NF + 1); print }'))
	@$(eval EXISTING_RC_TAG=$(shell git tag --sort=-v:refname | grep "^$(NEXT_VERSION)-rc[0-9]*$$" | head -n 1))
	@$(eval NEW_TAG=$(shell \
		if [ -z "$(LATEST_TAG)" ]; then \
			echo "No stable version tags found" >&2; \
			exit 1; \
		elif [ -z "$(EXISTING_RC_TAG)" ]; then \
			echo "$(NEXT_VERSION)-rc1"; \
		else \
			RC_NUM=$$(echo "$(EXISTING_RC_TAG)" | sed 's/.*-rc\([0-9]*\)$$/\1/'); \
			NEW_RC_NUM=$$((RC_NUM + 1)); \
			echo "$(NEXT_VERSION)-rc$$NEW_RC_NUM"; \
		fi))
	@git tag $(NEW_TAG)
	$(call print_success,RC version tagged: ${CYAN}$(NEW_TAG)${RESET})
	$(call print_next_steps,$(NEW_TAG))
	$(call print_cleanup_commands,$(NEW_TAG))
	$(call print_additional_info)

.PHONY: release_tag_minor
release_tag_minor: ## Tag a new minor release (e.g. v1.0.1 -> v1.0.2)
	@$(eval LATEST_TAG=$(shell git tag --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$$' | head -n 1))
	@$(eval NEW_TAG=$(shell echo $(LATEST_TAG) | awk -F. -v OFS=. '{ $$NF = sprintf("%d", $$NF + 1); print }'))
	@git tag $(NEW_TAG)
	$(call print_success,Bug fix version tagged: ${CYAN}$(NEW_TAG)${RESET})
	$(call print_next_steps,$(NEW_TAG))
	$(call print_cleanup_commands,$(NEW_TAG))
	$(call print_additional_info)

.PHONY: release_tag_major
release_tag_major: ## Tag a new major release (e.g. v1.0.0 -> v2.0.0)
	@$(eval LATEST_TAG=$(shell git tag --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$$' | head -n 1))
	@$(eval NEW_TAG=$(shell echo $(LATEST_TAG) | awk -F. '{$$2 += 1; $$3 = 0; print $$1 "." $$2 "." $$3}'))
	@git tag $(NEW_TAG)
	$(call print_success,Minor release version tagged: ${CYAN}$(NEW_TAG)${RESET})
	$(call print_next_steps,$(NEW_TAG))
	$(call print_cleanup_commands,$(NEW_TAG))
	$(call print_additional_info)

.PHONY: release_artifacts_current_branch
release_artifacts_current_branch: ## Trigger the release-artifacts workflow using the current branch to build artifacts for all environments
	@echo "Triggering release-artifacts workflow for current branch..."
	@BRANCH=$$(git rev-parse --abbrev-ref HEAD) && \
	gh workflow run release-artifacts.yml --ref $$BRANCH
	@echo "Workflow triggered for branch: ${CYAN} $$(git rev-parse --abbrev-ref HEAD)${RESET}"
	@echo "Check the workflow status at: ${BLUE}https://github.com/$(shell git config --get remote.origin.url | sed 's/.*github.com[:/]\([^/]*\/[^.]*\).*/\1/')/actions/workflows/release-artifacts.yml${RESET}"

###################################
### Local Docker Build Testing ###
###################################

.PHONY: docker_test_build_local
docker_test_build_local: ## Test Docker build locally with current architecture binaries only
	@echo "$(CYAN)Building binaries for local testing...$(RESET)"
	$(MAKE) ignite_release_local
	$(MAKE) ignite_release_extract_binaries
	$(MAKE) cosmovisor_cross_compile
	@echo "$(CYAN)Testing Docker build (CGO disabled)...$(RESET)"
	docker build -f Dockerfile.release -t pocketd-test:nocgo .
	# TODO_INVESTIGATE: CGO Docker build test disabled - https://github.com/pokt-network/poktroll/discussions/1822
	@echo "$(YELLOW)Skipping CGO-enabled Docker build test (disabled).$(RESET)"
	# docker build -f Dockerfile.release.cgo -t pocketd-test:cgo .
	$(call print_success,Docker build test successful!)

.PHONY: docker_test_build_multiplatform
docker_test_build_multiplatform: ## Test multi-platform Docker build locally (requires Docker buildx)
	@echo "$(CYAN)Building all platform binaries...$(RESET)"
	$(MAKE) ignite_release
	$(MAKE) ignite_release_repackage
	$(MAKE) ignite_release_extract_binaries
	$(MAKE) cosmovisor_cross_compile
	@echo "$(CYAN)Setting up Docker buildx...$(RESET)"
	@docker buildx create --name poktroll-builder --use 2>/dev/null || docker buildx use poktroll-builder
	@echo "$(CYAN)Testing multi-platform Docker build (CGO disabled)...$(RESET)"
	docker buildx build --platform linux/amd64,linux/arm64 \
		-f Dockerfile.release -t pocketd-test:nocgo-multi . --progress=plain
	# TODO_INVESTIGATE: CGO multiplatform Docker build test disabled - https://github.com/pokt-network/poktroll/discussions/1822
	@echo "$(YELLOW)Skipping CGO-enabled multi-platform Docker build test (disabled).$(RESET)"
	# docker buildx build --platform linux/amd64,linux/arm64 \
	# 	-f Dockerfile.release.cgo -t pocketd-test:cgo-multi . --progress=plain
	$(call print_success,Multi-platform Docker build test successful!)

.PHONY: docker_test_run
docker_test_run: ## Run the locally built Docker image to verify it works
	@echo "$(CYAN)Running Docker image test...$(RESET)"
	@docker run --rm pocketd-test:nocgo version || \
		(echo "$(RED)❌ Failed to run pocketd version$(RESET)" && exit 1)
	$(call print_success,Docker runtime test successful!)

.PHONY: docker_test_quick
docker_test_quick: ## Quick Docker build test - builds minimal binaries if needed (fastest)
	@# Check if we have Linux binaries (required for Docker)
	@if [ ! -d "release_binaries" ] || ! ls release_binaries/pocket*linux* >/dev/null 2>&1; then \
		echo "$(YELLOW)⚠️  No Linux binaries found. Building minimal set for Docker testing...$(RESET)"; \
		echo "$(CYAN)Building Linux binaries (CGO disabled)...$(RESET)"; \
		$(MAKE) ignite_release_cgo_disabled; \
		$(MAKE) ignite_release_extract_binaries; \
	fi
	@if [ ! -d "tmp" ] || [ ! -f "tmp/cosmovisor-linux-amd64" -a ! -f "tmp/cosmovisor-linux-arm64" ]; then \
		echo "$(YELLOW)⚠️  Building cosmovisor for testing...$(RESET)"; \
		$(MAKE) cosmovisor_cross_compile; \
	fi
	@echo "$(CYAN)Testing Docker build with Linux binaries...$(RESET)"
	@echo "$(CYAN)Available binaries:$(RESET)"
	@ls -la release_binaries/pocket*linux* 2>/dev/null || echo "No Linux binaries found"
	docker build -f Dockerfile.release -t pocketd-test:quick . --progress=plain
	$(call print_success,Quick Docker build test successful!)

.PHONY: docker_test_clean
docker_test_clean: ## Clean up test Docker images and build cache
	@echo "$(CYAN)Cleaning up test Docker images...$(RESET)"
	@docker rmi -f pocketd-test:nocgo \
		pocketd-test:nocgo-multi \
		pocketd-test:quick 2>/dev/null || true
	@docker builder prune -f
	@docker buildx rm poktroll-builder 2>/dev/null || true
	$(call print_success,Docker test cleanup complete!)
