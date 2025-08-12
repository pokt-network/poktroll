#!/bin/bash

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Fetching latest block height...${NC}"
latest=$(pocketd status --network=main -o json | jq -r '.sync_info.latest_block_height')
echo -e "${YELLOW}Latest block: $latest${NC}"

echo ""
echo -e "${GREEN}Building validator map...${NC}"

# Build map: proposer_address → validator info
declare -A MAP

# Fetch all validators
validators_json=$(pocketd query staking validators --network=main --grpc-insecure=false -o json)

# Count validators
total_validators=$(echo "$validators_json" | jq '.validators | length')
echo "Processing $total_validators validators..."

# Process each validator
count=0
while read -r pubkey_type pubkey_b64 operator moniker; do
  ((count++)) || true
  
  # Skip if pubkey is empty
  if [ -z "$pubkey_b64" ] || [ "$pubkey_b64" = "null" ]; then
    continue
  fi
  
  # Progress indicator
  printf "\r  Processing validator %d/%d..." "$count" "$total_validators"
  
  # For ed25519 keys in Cosmos SDK:
  # 1. The consensus_pubkey.value is base64 encoded public key
  # 2. The proposer_address in blocks is first 20 bytes of SHA256(pubkey_bytes) in hex
  
  # Decode the base64 pubkey
  pubkey_hex=$(echo "$pubkey_b64" | base64 --decode | xxd -p -c 256)
  
  # Calculate the address: SHA256 hash of the raw pubkey bytes, take first 20 bytes (40 hex chars)
  proposer_addr=$(echo -n "$pubkey_hex" | xxd -r -p | openssl dgst -sha256 -binary | xxd -p -c 20 | tr '[:lower:]' '[:upper:]')
  
  # Store the mapping
  if [ -n "$proposer_addr" ]; then
    MAP["$proposer_addr"]="$operator|$moniker"
  fi
done < <(echo "$validators_json" | jq -r '.validators[] | [.consensus_pubkey["@type"], .consensus_pubkey.value, .operator_address, .description.moniker] | @tsv')

echo ""
echo -e "${GREEN}Validator map built with ${#MAP[@]} entries${NC}"

echo ""
echo -e "${GREEN}Fetching last 10 blocks...${NC}"
echo "================================"

# Loop through last 10 blocks
for ((h=latest; h>latest-10 && h>0; h--)); do
  # Get the block data
  block_json=$(pocketd query block --type=height "$h" --network=main -o json 2>/dev/null || echo "{}")
  
  # Extract proposer address (it's in uppercase hex format in the block header)
  proposer_addr=$(echo "$block_json" | jq -r '.header.proposer_address // ""' | tr '[:lower:]' '[:upper:]')
  
  if [ -n "$proposer_addr" ] && [ "$proposer_addr" != "null" ] && [ "$proposer_addr" != "" ]; then
    if [ -n "${MAP[$proposer_addr]:-}" ]; then
      validator_info="${MAP[$proposer_addr]}"
      operator=$(echo "$validator_info" | cut -d'|' -f1)
      moniker=$(echo "$validator_info" | cut -d'|' -f2)
      echo -e "Block ${YELLOW}$h${NC}: ${GREEN}$moniker${NC} (${operator:0:20}...)"
    else
      echo -e "Block ${YELLOW}$h${NC}: ${RED}Unknown validator${NC} (${proposer_addr:0:8}...)"
    fi
  else
    echo -e "Block ${YELLOW}$h${NC}: ${RED}Could not retrieve proposer${NC}"
  fi
done

echo ""
echo -e "${GREEN}Done!${NC}"