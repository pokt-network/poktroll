
.PHONY: proof_payload_size_analysis
proof_payload_size_analysis: ## Analyze block size savings from hashing RelayResponse payloads in MsgSubmitProofs.
	go run ./tools/scripts/proof_payload_hash_analysis/main.go

.PHONY: decode_relay
decode_relay: ## Debug base64-encoded RelayRequest data that fails to unmarshal.
	@echo "Usage: make decode_relay BASE64_DATA=<base64_encoded_relay_request>"
	@if [ -z "$(BASE64_DATA)" ]; then \
		echo "Error: BASE64_DATA parameter is required"; \
		echo "Example: make decode_relay BASE64_DATA='ChYKFGFwcGxpY2F0aW9uX2FkZHJlc3M='"; \
		exit 1; \
	fi
	go run ./tools/scripts/decode_relay/main.go $(BASE64_DATA)

.PHONY: query_helpers
query_helpers: ## Load query helpers into your shell
	@(source ./tools/rc_helpers/queries.sh)
