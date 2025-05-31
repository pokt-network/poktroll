#!/bin/bash

PARAM_DIR="tools/scripts/params/state_shift_params"
PARAM_FILES=(
  application_params.json
  gateway_params.json
  mint_params.json
  proof_params.json
  service_params.json
  session_params.json
  shared_params.json
  slashing_params.json
  staking_params.json
  supplier_params.json
  tokenomics_params.json
)

for file in "${PARAM_FILES[@]}"; do
  pocketd tx authz exec "$PARAM_DIR/$file" \
    --from=pnf_beta \
    --keyring-backend=test \
    --chain-id=pocket-beta \
    --node="https://shannon-testnet-grove-rpc.beta.poktroll.com \
    --yes \
    --home=~/.pocket_prod \
    --fees=200upokt
done