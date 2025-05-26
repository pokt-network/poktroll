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

###########################
###   Release Helpers   ###
###########################

# List tags: git tag
# Delete tag locally: git tag -d v1.2.3
# Delete tag remotely: git push --delete origin v1.2.3

.PHONY: release_tag_local_testing
release_tag_local_testing: ## Tag a new local testing release (e.g. v1.0.1 -> v1.0.2-test1, v1.0.2-test1 -> v1.0.2-test2)
	@LATEST_TAG=$$(git tag --sort=-v:refname | head -n 1 | xargs); \
	if [ -z "$$LATEST_TAG" ]; then \
	  NEW_TAG=v0.1.0-test1; \
	else \
	  if echo "$$LATEST_TAG" | grep -q -- '-test'; then \
	    BASE_TAG=$$(echo "$$LATEST_TAG" | sed 's/-test[0-9]*//'); \
	    LAST_TEST_NUM=$$(echo "$$LATEST_TAG" | sed -E 's/.*-test([0-9]+)/\1/'); \
	    NEXT_TEST_NUM=$$(($$LAST_TEST_NUM + 1)); \
	    NEW_TAG=$${BASE_TAG}-test$${NEXT_TEST_NUM}; \
	  else \
	    BASE_TAG=$$(echo "$$LATEST_TAG" | awk -F. -v OFS=. '{$$NF = sprintf("%d", $$NF + 1); print}'); \
	    NEW_TAG=$${BASE_TAG}-test1; \
	  fi; \
	fi; \
	git tag $$NEW_TAG; \
	echo "New local testing version tagged: $$NEW_TAG"; \
	echo "Run the following commands to push the new tag:"; \
	echo "  git push origin $$NEW_TAG"; \
	echo "And draft a new release at https://github.com/pokt-network/poktroll/releases/new";


.PHONY: release_tag_dev
release_tag_dev: ## Tag a new dev release (e.g. v1.0.1 -> v1.0.1-dev1, v1.0.1-dev1 -> v1.0.1-dev2)
	@$(eval LATEST_TAG=$(shell git tag --sort=-v:refname | head -n 1))
	@$(eval BASE_VERSION=$(shell echo $(LATEST_TAG) | sed 's/-dev[0-9]*$$//' ))
	@$(eval EXISTING_DEV_TAGS=$(shell git tag --sort=-v:refname | grep "^$(BASE_VERSION)-dev[0-9]*$$" | head -n 1))
	@if [ -z "$(EXISTING_DEV_TAGS)" ]; then \
		NEW_TAG="$(BASE_VERSION)-dev1"; \
	else \
		DEV_NUM=$$(echo $(EXISTING_DEV_TAGS) | sed 's/.*-dev\([0-9]*\)$$/\1/'); \
		NEW_DEV_NUM=$$((DEV_NUM + 1)); \
		NEW_TAG="$(BASE_VERSION)-dev$$NEW_DEV_NUM"; \
	fi; \
	git tag $$NEW_TAG; \
	echo "########"; \
	echo "New dev version tagged: $$NEW_TAG"; \
	echo ""; \
	echo "If you need to delete a tag, run:"; \
	echo "  git tag -d $$NEW_TAG"; \
	echo ""; \
	echo "If you need to delete a tag remotely, run:"; \
	echo "  git push origin --delete $$NEW_TAG"; \
	echo ""; \
	echo "Next, do the following:"; \
	echo "1. Run the following commands to push the new tag:"; \
	echo "   git push origin $$NEW_TAG"; \
	echo "2. And draft a new release at https://github.com/pokt-network/poktroll/releases/new"
	echo ""; \
	echo "Visit this URL for more info: https://dev.poktroll.com/explore/account_management/pocketd_cli?_highlight=cli"
	echo "########"

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


.PHONY: ignite_update_ldflags
## Artifact release helper - sets version/datetime of the build
ignite_update_ldflags:
	yq eval '.build.ldflags = ["-X main.Version=$(VERSION)", "-X main.Date=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"]' -i config.yml

.PHONY: ignite_release
ignite_release: ignite_check_version ## Builds production binaries for all architectures and outputs them in the
# TODO_WIP: @Olshansky testing things manually.
# ignite chain build --release -t linux:amd64 -t linux:arm64 -t darwin:amd64 -t darwin:arm64 -o release
	ignite chain build --release -t darwin:arm64 -o release
	cd release && for f in poktroll_*.tar.gz; do mv "$$f" "pocket_$${f#poktroll_}"; done
	# The existing release_checksum file generated by 'ignite' is using 'poktroll' in the filename - we need to update it to use 'pocket'
	cd release && sed 's/poktroll/pocket/g' release_checksum > release_checksum.tmp && mv release_checksum.tmp release_checksum

.PHONY: ignite_release_local
ignite_release_local: ignite_check_version ## Builds a production binary for the current architecture only and outputs it in the release directory
	ignite chain build --release -o release
	cd release && for f in poktroll_*.tar.gz; do mv "$$f" "pocket_$${f#poktroll_}"; done
	# The existing release_checksum file generated by 'ignite' is using 'poktroll' in the filename - we need to update it to use 'pocket'
	cd release && sed 's/poktroll/pocket/g' release_checksum > release_checksum.tmp && mv release_checksum.tmp release_checksum

.PHONY: ignite_release_extract_binaries
ignite_release_extract_binaries: ## Extracts binaries from the release archives
	mkdir -p release_binaries
	for archive in release/*.tar.gz; do \
		binary_name=$$(basename "$$archive" .tar.gz); \
		temp_dir=$$(mktemp -d); \
		tar -zxf "$$archive" -C "$$temp_dir"; \
		find "$$temp_dir" -name "pocketd" -type f -exec cp {} "release_binaries/$$binary_name" \; ; \
		rm -rf "$$temp_dir"; \
	done

########################
###   Act Triggers   ###
########################

.PHONY: workflow_test_release_artifacts
workflow_test_release_artifacts: check_act check_secrets ## Test the release artifacts GitHub workflow
	@echo "Testing release artifacts workflow..."
	@act -W $(GH_WORKFLOWS)/release-artifacts.yml workflow_dispatch $(ACT_ARCH_FLAG) -v --secret-file .secrets
