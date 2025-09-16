########################
### Telegram Helpers ###
########################

.PHONY: telegram_broadcast_msg
telegram_broadcast_msg: ## Broadcast a custom message to all Telegram groups. Usage: make telegram_broadcast_msg MSG_FILE=message.txt
	@if [ -z "$(MSG_FILE)" ]; then \
		echo "Error: MSG_FILE parameter is required. Usage: make telegram_broadcast_msg MSG_FILE=message.txt"; \
		exit 1; \
	fi
	@echo "Broadcasting message to all Telegram groups...\n"
	@MSG="$$(cat $(MSG_FILE))"; \
	gh workflow run telegram-broadcast.yml \
		--raw-field message="$$MSG" \
		--raw-field test_mode=false \
	@echo "\nBroadcast initiated. Check the workflow status at: $(CYAN)https://github.com/pokt-network/poktroll/actions/workflows/telegram-broadcast.yml$(RESET)"

.PHONY: telegram_test_broadcast_msg
telegram_test_broadcast_msg: ## Test broadcast message from file. Usage: make telegram_test_broadcast_msg MSG_FILE=message.html
	@if [ -z "$(MSG_FILE)" ]; then \
		echo "Error: MSG_FILE parameter is required. Usage: make telegram_test_broadcast_msg MSG_FILE=message.html"; \
		exit 1; \
	fi
	@echo "Sending test message to Telegram testing group...\n"
	@MSG="$$(cat $(MSG_FILE))"; \
	gh workflow run telegram-broadcast.yml \
		--raw-field message="$$MSG" \
		--raw-field test_mode=true \
		--ref="$(shell git rev-parse --abbrev-ref HEAD)"
	@echo "\nTest message initiated. Check the workflow status at: $(CYAN)https://github.com/pokt-network/poktroll/actions/workflows/telegram-broadcast.yml$(RESET)"