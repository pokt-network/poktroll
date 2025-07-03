########################
### Telegram Helpers ###
########################

.PHONY: telegram_broadcast
telegram_broadcast: ## Broadcast a custom message to all Telegram groups. Usage: make telegram_broadcast MSG="Your message here"
	@if [ -z "$(MSG)" ]; then \
		echo "Error: MSG parameter is required. Usage: make telegram_broadcast MSG=\"Your message here\""; \
		exit 1; \
	fi
	@echo "Broadcasting message to all Telegram groups..."
	@gh workflow run telegram-broadcast.yml -f message="$(MSG)"
	@echo "Broadcast initiated. Check the workflow status at: https://github.com/pokt-network/poktroll/actions/workflows/telegram-broadcast.yml"

.PHONY: telegram_release_notify
telegram_release_notify: ## Notify all Telegram groups of the latest release
	@echo "Notifying Telegram groups of the latest release..."
	@gh workflow run telegram-notify-release.yml
	@echo "Release notification initiated. Check the workflow status at: https://github.com/pokt-network/poktroll/actions/workflows/telegram-notify-release.yml"