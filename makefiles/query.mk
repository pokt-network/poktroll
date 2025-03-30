#####################
### Query Helpers ###
#####################

.PHONY: query_tx
query_tx: ## Query for a transaction by hash and output as YAML (default).
	pocketd --home=$(POCKETD_HOME) query tx $(HASH) --node $(POCKET_NODE)

.PHONY: query_tx_json
query_tx_json: ## Query for a transaction by hash and output as JSON.
	pocketd --home=$(POCKETD_HOME) query tx $(HASH) --output json --node $(POCKET_NODE)

.PHONY: query_tx_log
query_tx_log: ## Query for a transaction and print its raw log.
	$(MAKE) -s query_tx_json | jq .raw_log
