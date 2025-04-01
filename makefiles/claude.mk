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

.PHONY: claudesync_init
claudesync_init: claudesync_check ## Initializes a new ClaudeSync project
	@echo "###############################################"
	@echo "Initializing a new ClaudeSync project"
	@echo "When prompted, enter the following name: poktroll"
	@echo "When prompted, enter the following description: Pocket Network Source Code"
	@echo "When prompted for an absolute path, press enter"
	@echo "Follow the Remote URL outputted and copy-paste the recommended system prompt from the README"
	@echo "###############################################"
	@claudesync project init --new

.PHONY: claudesync_push
claudesync_push: claudesync_check ## Pushes the current project to the ClaudeSync project
	@claudesync push


# TODO:
- Have one base .claudeignore file
- Have multiple .claudeignore files
- Create multiple "claudesync_push_*"  targets that updates the .claudeignore file and pushes  the appropriat one
- Need to tailor it specifically to different things:
	- Testing
	- Onchain
	- Offchain
	- Devops
	- Etc...
