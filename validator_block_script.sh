#!/bin/bash

set -euo pipefail

# Get latest block height
echo "Fetching latest block height..."
latest=$(pocketd status --network=main -o json | jq -r '.sync_info.latest_block_height')
echo "Latest block: $latest"

echo ""
echo "Building validator map..."

# Build map: proposer_address (hex) → "valcons | operator | moniker"
declare -A MAP

# Fetch all validators
validators_json=$(pocketd query staking validators --network=main --grpc-insecure=false -o json)

# Process each validator
while read -r pubkey_b64 operator moniker; do
  # Skip if pubkey is empty
  if [ -z "$pubkey_b64" ] || [ "$pubkey_b64" = "null" ]; then
    continue
  fi
  
  # The consensus_pubkey.value is base64 encoded Ed25519 public key
  # We need to decode it and get the address bytes
  
  # Method 1: Direct conversion from consensus pubkey to address
  # Decode base64 -> get raw bytes -> hash -> take first 20 bytes -> encode as hex
  raw_bytes=$(echo "$pubkey_b64" | base64 --decode | xxd -p -c 256)
  
  # The proposer_address in the block header is the SHA256 hash of the pubkey, truncated to 20 bytes
  # This matches how Tendermint/CometBFT generates addresses
  proposer_hex=$(echo -n "$raw_bytes" | xxd -r -p | sha256sum | cut -c1-40 | tr '[:lower:]' '[:upper:]')
  
  # Also try to get the valcons address for display
  valcons=$(echo "$raw_bytes" | xargs -I{} pocketd keys parse {} 2>/dev/null | grep '^poktvalcons1' | head -1 || echo "")
  
  if [ -n "$proposer_hex" ]; then
    MAP["$proposer_hex"]="$valcons | $operator | $moniker"
    echo "  Mapped validator: $moniker (${proposer_hex:0:8}...)"
  fi
done < <(echo "$validators_json" | jq -r '.validators[] | [.consensus_pubkey.value, .operator_address, .description.moniker] | @tsv')

echo ""
echo "Fetching last 10 blocks..."
echo "=========================="

# Loop through last 10 blocks
for ((h=latest; h>latest-10 && h>0; h--)); do
  # Get the block data
  block_json=$(pocketd query block --type=height "$h" --network=main -o json 2>/dev/null || echo "{}")
  
  # Extract proposer address (it's in hex format in the block header)
  proposer_hex=$(echo "$block_json" | jq -r '.header.proposer_address // ""' | tr '[:lower:]' '[:upper:]')
  
  if [ -n "$proposer_hex" ] && [ "$proposer_hex" != "null" ]; then
    if [ -n "${MAP[$proposer_hex]:-}" ]; then
      validator_info="${MAP[$proposer_hex]}"
      # Extract just the moniker for cleaner output
      moniker=$(echo "$validator_info" | cut -d'|' -f3 | xargs)
      operator=$(echo "$validator_info" | cut -d'|' -f2 | xargs)
      echo "Block $h: $moniker ($operator)"
    else
      echo "Block $h: Unknown validator (${proposer_hex:0:8}...)"
    fi
  else
    echo "Block $h: Could not retrieve proposer"
  fi
done