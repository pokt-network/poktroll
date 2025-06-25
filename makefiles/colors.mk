
# Color definitions for better readability
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
CYAN := \033[0;36m
RED := \033[0;31m
BOLD := \033[1m
RESET := \033[0m

# Reusable function to print formatted messages
define print_success
	echo ""
	echo "$(GREEN)########################################$(RESET)"
	echo "$(GREEN)$(BOLD) ✓ $(1)$(RESET)"
	echo "$(GREEN)########################################$(RESET)"
	echo ""
endef

define print_info_section
	echo "$(CYAN)$(BOLD)$(1):$(RESET)"
endef

define print_command
	echo "  $(YELLOW)$(1)$(RESET)"
endef

define print_url
	echo "$(BLUE)$(1)$(RESET)"
endef

define print_warning
	echo "$(RED)⚠️  $(1)$(RESET)"
endef