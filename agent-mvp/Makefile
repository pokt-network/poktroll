SHELL := /bin/sh

include colors.mk

UV ?= uv
PYTHON_VERSION ?= 3.11
VENV ?= .venv
PYTHON := $(VENV)/bin/python
ENV_SCRIPT ?= env.sh
CHROMA_DATA_DIR := chroma_data
OPENRPC_SPEC_URL := https://raw.githubusercontent.com/etclabscore/ethereum-json-rpc-specification/refs/heads/master/openrpc.json
OPENRPC_SPEC_FILE := openrpc_eth.json

.DEFAULT_GOAL := help

################
### Commands ###
################

.PHONY: help quickstart env-file venv install download-spec build-index ask query-eth clean

help: ## Display grouped command list
	@echo ""
	@echo "$(BOLD_GREEN)=== Quickstart ===$(RESET)"
	@grep -h -E '^(quickstart):.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-20s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD_GREEN)=== Setup ===$(RESET)"
	@grep -h -E '^(env-file|venv|install|download-spec):.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-20s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD_GREEN)=== Workflow ===$(RESET)"
	@grep -h -E '^(build-index|ask|query-eth):.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-20s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BOLD_GREEN)=== Cleanup ===$(RESET)"
	@grep -h -E '^(clean):.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-20s$(RESET) %s\n", $$1, $$2}'

quickstart: env-file build-index ## Create env script, install deps, and build index
	@echo ""
	@echo "Environment script ready: $(ENV_SCRIPT)"
	@echo "Run: source $(ENV_SCRIPT)"
	@echo "Then: make ask Q=\"What is the height of the ethereum blockchain?\""

################
### Setup     ###
################

env-file: ## Create $(ENV_SCRIPT) exporting GROVE_APP_ID if absent
	@if [ ! -f $(ENV_SCRIPT) ]; then \
		echo 'export GROVE_APP_ID="YOUR_GROVE_APP_ID"' > $(ENV_SCRIPT); \
		chmod +x $(ENV_SCRIPT); \
		echo "Created $(ENV_SCRIPT). Please edit it to set your GROVE_APP_ID."; \
	else \
		echo "$(ENV_SCRIPT) already exists. Make sure GROVE_APP_ID is set."; \
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

download-spec: ## Download Ethereum OpenRPC specification
	@if [ -f $(OPENRPC_SPEC_FILE) ]; then \
		echo "$(OPENRPC_SPEC_FILE) already exists. Remove it first to re-download."; \
	else \
		echo "Downloading Ethereum OpenRPC spec from $(OPENRPC_SPEC_URL)"; \
		curl -L -o $(OPENRPC_SPEC_FILE) $(OPENRPC_SPEC_URL); \
		echo "Downloaded $(OPENRPC_SPEC_FILE)"; \
	fi

################
### Workflow ###
################

build-index: install ## Generate ChromaDB index from schema
	$(PYTHON) build_index.py

ask: install ## Retrieve best method for Q="question" and output curl command
	@if [ -z "$(Q)" ]; then \
		echo "Usage: make ask Q=\"your question\""; \
		exit 1; \
	fi
	@if [ -z "$$GROVE_APP_ID" ]; then \
		echo "Error: GROVE_APP_ID is not set. Please source $(ENV_SCRIPT) first."; \
		echo "Run: source $(ENV_SCRIPT)"; \
		exit 1; \
	fi
	$(PYTHON) query_to_curl.py "$(Q)"

query-eth: install ## Query Ethereum using REQUEST="your natural language request"
	@if [ -z "$(REQUEST)" ]; then \
		echo "Usage: make query-eth REQUEST=\"your natural language request\""; \
		echo "Example: make query-eth REQUEST=\"What is the current block height?\""; \
		exit 1; \
	fi
	@if [ -z "$$GROVE_APP_ID" ]; then \
		echo "Error: GROVE_APP_ID is not set. Please source $(ENV_SCRIPT) first."; \
		echo "Run: source $(ENV_SCRIPT)"; \
		exit 1; \
	fi
	$(PYTHON) query_to_curl.py "$(REQUEST)"

################
### Cleanup  ###
################

clean: ## Remove virtual environment, artifacts, and env script
	rm -rf $(VENV) $(CHROMA_DATA_DIR) $(ENV_SCRIPT) $(OPENRPC_SPEC_FILE)
