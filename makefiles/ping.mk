########################
### Localnet Helpers ###
########################

.PHONY: localnet_relayminer1_ping
localnet_relayminer1_ping:
	@echo "Pinging relayminer 1..."
	@curl -X GET localhost:7001 || (echo "Failed to ping relayminer1. Make sure your localnet environment or the relayminer 1 pod is up and running"; exit 1)
	@echo "OK"

.PHONY: localnet_relayminer2_ping
localnet_relayminer2_ping:
	@echo "Pinging relayminer 2..."
	@curl -X GET localhost:7002 || (echo "Failed to ping relayminer2. Make sure your localnet environment or the relayminer 2 pod is up and running"; exit 1)
	@echo "OK"

.PHONY: localnet_relayminer3_ping
localnet_relayminer3_ping:
	@echo "Pinging relayminer 3..."
	@curl -X GET localhost:7003 || (echo "Failed to ping relayminer3. Make sure your localnet environment or the relayminer 3 pod is up and running"; exit 1)
	@echo "OK"
