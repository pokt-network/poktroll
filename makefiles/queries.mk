.PHONY: shannon_query_services_by_owner
shannon_query_services_by_owner: ## Query services by owner address
	./e2e/scripts/shannon_preliminary_services_helpers.sh shannon_query_services_by_owner

.PHONY: shannon_query_service_tlds_by_id
shannon_query_service_tlds_by_id: ## Query service TLDs by service ID
	./e2e/scripts/shannon_preliminary_services_helpers.sh shannon_query_service_tlds_by_id


scripts
- Gov params
- queries
- What else?
./tools/scripts/params/gov_params.sh update supplier --env main --home ~/.pocket_prod
