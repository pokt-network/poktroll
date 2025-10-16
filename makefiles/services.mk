################
### Services ###
################
#
# Usage examples:
#   make service_list
#   SERVICE_ID=anvil make service_show
#   SERVICE_ID=my-svc SERVICE_NAME="My Service" COMPUTE_UNITS=10 SERVICE_OWNER=app1 make service_add
#   SERVICE_ID=my-svc SERVICE_NAME="My Service" COMPUTE_UNITS=10 SERVICE_OWNER=app1 METADATA_FILE=./api.json make service_add_with_metadata_file

.PHONY: service_list
service_list: ## List all services registered on the network
	pocketd --home=$(POCKETD_HOME) q service list-service --node $(POCKET_NODE)

.PHONY: service_show
service_show: ## Show details of a service (must specify SERVICE_ID env var)
	pocketd --home=$(POCKETD_HOME) q service show-service $(SERVICE_ID) --node $(POCKET_NODE)

.PHONY: service_add
service_add: ## Add a new service (must specify SERVICE_ID, SERVICE_NAME, COMPUTE_UNITS, SERVICE_OWNER env vars)
	pocketd --home=$(POCKETD_HOME) tx service add-service \
		$(SERVICE_ID) "$(SERVICE_NAME)" $(COMPUTE_UNITS) \
		--from $(SERVICE_OWNER) --keyring-backend test --node $(POCKET_NODE) -y

.PHONY: service_add_with_metadata_file
service_add_with_metadata_file: ## Add a service with metadata from file (must specify SERVICE_ID, SERVICE_NAME, COMPUTE_UNITS, SERVICE_OWNER, METADATA_FILE env vars)
	pocketd --home=$(POCKETD_HOME) tx service add-service \
		$(SERVICE_ID) "$(SERVICE_NAME)" $(COMPUTE_UNITS) \
		--experimental--metadata-file $(METADATA_FILE) \
		--from $(SERVICE_OWNER) --keyring-backend test --node $(POCKET_NODE) -y

.PHONY: service_add_with_metadata_base64
service_add_with_metadata_base64: ## Add a service with base64-encoded metadata (must specify SERVICE_ID, SERVICE_NAME, COMPUTE_UNITS, SERVICE_OWNER, METADATA_BASE64 env vars)
	pocketd --home=$(POCKETD_HOME) tx service add-service \
		$(SERVICE_ID) "$(SERVICE_NAME)" $(COMPUTE_UNITS) \
		--experimental--metadata-base64 $(METADATA_BASE64) \
		--from $(SERVICE_OWNER) --keyring-backend test --node $(POCKET_NODE) -y

.PHONY: service_add_pocket
service_add_pocket: ## Add the pocket service with OpenAPI specification metadata
	SERVICE_ID=pocket \
	SERVICE_NAME="Pocket Network RPC" \
	COMPUTE_UNITS=1 \
	SERVICE_OWNER=app1 \
	METADATA_FILE=docs/static/openapi_small.json \
	make service_add_with_metadata_file

.PHONY: service_update_pocket_metadata
service_update_pocket_metadata: ## Update the pocket service metadata with OpenAPI specification
	SERVICE_ID=pocket \
	SERVICE_NAME="Pocket Network RPC" \
	COMPUTE_UNITS=1 \
	SERVICE_OWNER=app1 \
	METADATA_FILE=docs/static/openapi_small.json \
	make service_add_with_metadata_file
