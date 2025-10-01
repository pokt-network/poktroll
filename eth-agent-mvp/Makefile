SHELL := /bin/sh

UV ?= uv
PYTHON_VERSION ?= 3.11
VENV ?= .venv
PYTHON := $(VENV)/bin/python
ENV_SCRIPT ?= env.sh
INDEX_ARTIFACTS := methods.index methods.meta.pkl

.DEFAULT_GOAL := help

################
### Commands ###
################

.PHONY: help quickstart env-file venv install build-index ask clean

help: ## Display grouped command list
	@echo ""
	@echo "=== Quickstart ==="
	@grep -h -E '^(quickstart):.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'
	@echo ""
	@echo "=== Setup ==="
	@grep -h -E '^(env-file|venv|install):.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'
	@echo ""
	@echo "=== Workflow ==="
	@grep -h -E '^(build-index|ask):.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'
	@echo ""
	@echo "=== Cleanup ==="
	@grep -h -E '^(clean):.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

quickstart: env-file build-index ## Create env script, install deps, and build index
	@echo ""
	@echo "Environment script ready: $(ENV_SCRIPT)"
	@echo "Run: source $(ENV_SCRIPT)"
	@echo "Then: make ask Q=\"What is the height of the ethereum blockchain?\""

################
### Setup     ###
################

# TODO_IN_THIS_PR(@assistant): replace hardcoded ETH_RPC_URL with pocketd-derived endpoint command

env-file: ## Create $(ENV_SCRIPT) exporting ETH_RPC_URL if absent
	@if [ ! -f $(ENV_SCRIPT) ]; then \
		echo 'export ETH_RPC_URL="https://eth.rpc.grove.city/v1/6c5de5ff"' > $(ENV_SCRIPT); \
		chmod +x $(ENV_SCRIPT); \
		echo "Created $(ENV_SCRIPT) with staging ETH_RPC_URL"; \
	else \
		echo "$(ENV_SCRIPT) already exists. Edit it to set ETH_RPC_URL."; \
	fi

venv: ## Create/refresh Python $(PYTHON_VERSION) virtual environment with uv
	@if [ -d $(VENV) ]; then \
		CURRENT_VER=`$(PYTHON) -c 'import sys; print("{}.{}".format(sys.version_info[0], sys.version_info[1]))' 2>/dev/null || echo ""`; \
		if [ "$$CURRENT_VER" != "$(PYTHON_VERSION)" ]; then \
			echo "Recreating $(VENV) with Python $(PYTHON_VERSION) (was $$CURRENT_VER)"; \
			rm -rf $(VENV); \
		fi; \
	fi
	$(UV) venv --python $(PYTHON_VERSION) $(VENV)

install: venv ## Install dependencies into the virtual environment
	$(UV) pip install --python $(PYTHON) -r requirements.txt

################
### Workflow ###
################

build-index: install ## Generate FAISS index from schema
	$(PYTHON) build_index.py

ask: install ## Retrieve best method for Q="question" and output curl command
	@if [ -z "$(Q)" ]; then \
		echo "Usage: make ask Q=\"your question\""; \
		exit 1; \
	fi
	@if [ -z "$$ETH_RPC_URL" ]; then \
		echo "Warning: ETH_RPC_URL is not set. Using placeholder endpoint."; \
	fi
	$(PYTHON) query_to_curl.py "$(Q)"

################
### Cleanup  ###
################

clean: ## Remove virtual environment, artifacts, and env script
	rm -rf $(VENV) $(INDEX_ARTIFACTS) $(ENV_SCRIPT)
