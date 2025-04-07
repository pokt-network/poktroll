#####################
### Claude targets ###
#####################

.PHONY: claudesync_check
# Internal helper: Checks if claudesync is installed locally
claudesync_check:
	@if ! command -v claudesync >/dev/null 2>&1; then \
		echo "claudesync is not installed. Make sure you review this file: docusaurus/docs/develop/developer_guide/claude_sync.md"; \
		exit 1; \
	fi

.PHONY: claudesync_push
claudesync_push: claudesync_check ## Pushes the current project to the ClaudeSync project
	@claudesync push

##############################
### Claude onchain targets ###
##############################

.PHONY: claude_init_onchain
claude_init_onchain: claudesync_check ## Initializes a new ClaudeSync project for onchain code
	@echo "###############################################"
	@echo "Initializing a new ClaudeSync project for onchain code"
	@echo "When prompted, enter the following name: pocket_onchain"
	@echo "When prompted, enter the following description: Pocket Network Onchain Code (app, x, proto)"
	@echo "When prompted for an absolute path, press enter"
	@echo "Follow the Remote URL outputted and copy-paste the recommended system prompt from the README"
	@echo "###############################################"
	@claudesync project init --new --name pocket_onchain --description "Pocket Network Onchain Code (app, x, proto)" --local-path .

.PHONY: claude_set_onchain
claude_set_onchain: claudesync_check ## Sets the current ClaudeSync project to onchain code
	@echo "Updating the .claudeignore file for onchain code"
	@cp .claudeignore_onchain .claudeignore
	@echo "Select 'pocket-onchain' from the list of projects"
	@claudesync project set


# TODO:
# - Have one base .claudeignore file
# - Have multiple .claudeignore files
# - Create multiple "claudesync_push_*"  targets that updates the .claudeignore file and pushes  the appropriat one
# - Need to tailor it specifically to different things:
# 	- Testing
# 	- Onchain
# 	- Offchain
# 	- Devops
# 	- Etc...
