########################
### Telegram Helpers ###
########################

.PHONY: telegram_broadcast
telegram_broadcast: ## Broadcast a custom message to all Telegram groups. Usage: make telegram_broadcast MSG="Your message here"
	@if [ -z "$(MSG)" ]; then \
		echo "Error: MSG parameter is required. Usage: make telegram_broadcast MSG=\"Your message here\""; \
		exit 1; \
	fi
	@echo "Broadcasting message to all Telegram groups...\n"
	@gh workflow run telegram-broadcast.yml -f message="$(MSG)"
	@echo "\nBroadcast initiated. Check the workflow status at: $(CYAN)https://github.com/pokt-network/poktroll/actions/workflows/telegram-broadcast.yml$(RESET)"

.PHONY: telegram_release_notify
telegram_release_notify: ## Notify all Telegram groups of the latest release
	@echo "Notifying Telegram groups of the latest release...\n"
	@gh workflow run telegram-notify-release.yml
	@echo "\nRelease notification initiated. Check the workflow status at: $(CYAN)https://github.com/pokt-network/poktroll/actions/workflows/telegram-notify-release.yml$(RESET)"

.PHONY: telegram_test_message
telegram_test_message: ## Test broadcast message to testing group only. Usage: make telegram_test_message MSG="Your test message"
	@if [ -z "$(MSG)" ]; then \
		echo "Error: MSG parameter is required. Usage: make telegram_test_message MSG=\"Your test message\""; \
		exit 1; \
	fi
	@echo "Sending test message to Telegram testing group...\n"
	@gh workflow run telegram-broadcast.yml -f message="$(MSG)" -f test_mode=true
	@echo "\nTest message sent. Check the workflow status at: $(CYAN)https://github.com/pokt-network/poktroll/actions/workflows/telegram-broadcast.yml$(RESET)"

.PHONY: telegram_test_release
telegram_test_release: ## Test release notification to testing group only
	@echo "Sending release notification to Telegram testing group...\n"
	@gh workflow run telegram-notify-release.yml -f test_mode=true
	@echo "\nRelease notification sent. Check the workflow status at: $(CYAN)https://github.com/pokt-network/poktroll/actions/workflows/telegram-notify-release.yml$(RESET)"

.PHONY: telegram_test_release_from_branch
telegram_test_release_from_branch: ## Test release notification to testing group only from the current branch
	@echo "Sending release notification to Telegram testing group from branch $(shell git rev-parse --abbrev-ref HEAD)...\n"
	@gh workflow run telegram-notify-release.yml -f test_mode=true -f branch="$(shell git rev-parse --abbrev-ref HEAD)"
	@echo "\nRelease notification sent. Check the workflow status at: $(CYAN)https://github.com/pokt-network/poktroll/actions/workflows/telegram-notify-release.yml$(RESET)"
