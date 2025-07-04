
.PHONY: proof_payload_size_analysis
proof_payload_size_analysis: ## Analyze block size savings from hashing RelayResponse payloads in MsgSubmitProofs.
	go run ./tools/scripts/proof_payload_hash_analysis.go
