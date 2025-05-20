######################
### Claude targets ###
######################

.PHONY: claudesync_check
# Internal helper: Checks if claudesync is installed locally
claudesync_check:
	@if ! command -v claudesync >/dev/null 2>&1; then \
		echo "claudesync is not installed. Make sure you review this file: docusaurus/docs/develop/tips/claude.md"; \
		exit 1; \
	fi

#############################
### Documentation targets ###
#############################

.PHONY: claudesync_init_docs
claudesync_init_docs: claudesync_check ## Initializes a new ClaudeSync project for documentation
	@echo "###############################################"
	@echo "Initializing a new ClaudeSync project for Pocket's documentation"
	@echo "Follow the Remote URL outputted and copy-paste the recommended system prompt"
	@echo "###############################################"
	@make claudesync_category_add_docs
	@claudesync project init --new --name pocket_docs --description "Pocket Documentation" --local-path .
	@echo "See this file for the Pocket Docs recommended system prompt: docusaurus/docs/develop/tips/claude.md"

.PHONY: claudesync_set_docs
claudesync_set_docs: claudesync_check ## Sets the current ClaudeSync project to the pocket_docs project
	@echo "Updating the .claudeignore file for documentation"
	@cp .claudeignore_docs .claudeignore
	@echo "Select 'pocket_docs' from the list of projects"
	@claudesync project set

.PHONY: claudesync_push_docs
claudesync_push_docs: claudesync_check ## Pushes only markdown documentation to Claude
	@echo "Pushing documentation to Claude..."
	@claudesync push --category docs

.PHONY: claudesync_category_add_docs
# Internal Helper: Adds a new category for documentation
claudesync_category_add_docs:
	@echo "Adding a new category for documentation"
	@claudesync config category add docs \
	--description "Documentation including markdown files" \
	--patterns "*.md"

###################
### CLI targets ###
###################

.PHONY: claudesync_init_cli
claudesync_init_cli: claudesync_check ## Initializes a new ClaudeSync project for CLI
	@echo "###############################################"
	@echo "Initializing a new ClaudeSync project for Pocket's CLI"
	@echo "Follow the Remote URL outputted and copy-paste the recommended system prompt"
	@echo "###############################################"
	@make claudesync_category_add_cli
	@claudesync project init --new --name pocket_cli --description "Pocket CLI" --local-path .
	@echo "See this file for the Pocket CLI recommended system prompt: docusaurus/docs/develop/tips/claude.md"

.PHONY: claudesync_set_cli
claudesync_set_cli: claudesync_check ## Sets the current ClaudeSync project to the pocket_cli project
	@echo "Updating the .claudeignore file for CLI"
	@cp .claudeignore_cli .claudeignore
	@echo "Select 'pocket_cli' from the list of projects"
	@claudesync project set

.PHONY: claudesync_push_cli
claudesync_push_cli: claudesync_check ## Pushes only CLI source to Claude
	@echo "Pushing CLI source to Claude..."
	@claudesync push --category cli

.PHONY: claudesync_category_add_cli
# Internal Helper: Adds a new category for CLI commands
claudesync_category_add_cli:
	@echo "Adding a new category for CLI commands"
	@claudesync config category add cli \
	--description "CLI commands source files" \
	--patterns "cmd/pocketd/cmd/root.go" \
	--patterns "pkg/relayer/cmd/cmd.go" \
	--patterns "x/application/module/query_*.go" \
	--patterns "x/application/module/tx_*.go" \
	--patterns "x/gateway/module/query_*.go" \
	--patterns "x/gateway/module/tx_*.go" \
	--patterns "x/migration/module/cmd/*.go" \
	--patterns "x/migration/module/tx.go" \
	--patterns "x/proof/module/query_*.go" \
	--patterns "x/proof/module/tx_*.go" \
	--patterns "x/service/module/tx_*.go" \
	--patterns "x/session/module/query_*.go" \
	--patterns "x/supplier/module/tx_*.go" \
	--patterns "x/tokenomics/module/query_*.go"