#!/bin/bash

# Beta TestNet

SHANNON_BETA_URL=https://shannon-testnet-grove-rpc.beta.poktroll.com
pocketd tx authz exec ./slashing_params_beta_20250531_153448.json --from=pnf_beta --keyring-backend=test --chain-id=pocket-beta --node=${SHANNON_BETA_URL} --yes --home=~/.pocket_prod --fees=200upokt
pocketd tx authz exec ./application_params_beta_20250531_153122.json --from=pnf_beta --keyring-backend=test --chain-id=pocket-beta --node=${SHANNON_BETA_URL} --yes --home=~/.pocket_prod --fees=200upokt
pocketd tx authz exec ./gateway_params_beta_20250531_153128.json --from=pnf_beta --keyring-backend=test --chain-id=pocket-beta --node=${SHANNON_BETA_URL} --yes --home=~/.pocket_prod --fees=200upokt
pocketd tx authz exec ./mint_params_beta_20250531_153002.json --from=pnf_beta --keyring-backend=test --chain-id=pocket-beta --node=${SHANNON_BETA_URL} --yes --home=~/.pocket_prod --fees=200upokt
pocketd tx authz exec ./mint_params_beta_20250531_153315.json --from=pnf_beta --keyring-backend=test --chain-id=pocket-beta --node=${SHANNON_BETA_URL} --yes --home=~/.pocket_prod --fees=200upokt
pocketd tx authz exec ./proof_params_beta_20250531_153144.json --from=pnf_beta --keyring-backend=test --chain-id=pocket-beta --node=${SHANNON_BETA_URL} --yes --home=~/.pocket_prod --fees=200upokt
pocketd tx authz exec ./service_params_beta_20250531_153131.json --from=pnf_beta --keyring-backend=test --chain-id=pocket-beta --node=${SHANNON_BETA_URL} --yes --home=~/.pocket_prod --fees=200upokt
pocketd tx authz exec ./session_params_beta_20250531_153140.json --from=pnf_beta --keyring-backend=test --chain-id=pocket-beta --node=${SHANNON_BETA_URL} --yes --home=~/.pocket_prod --fees=200upokt
pocketd tx authz exec ./shared_params_beta_20250531_153203.json --from=pnf_beta --keyring-backend=test --chain-id=pocket-beta --node=${SHANNON_BETA_URL} --yes --home=~/.pocket_prod --fees=200upokt
pocketd tx authz exec ./staking_params_beta_20250531_153324.json --from=pnf_beta --keyring-backend=test --chain-id=pocket-beta --node=${SHANNON_BETA_URL} --yes --home=~/.pocket_prod --fees=200upokt
pocketd tx authz exec ./supplier_params_beta_20250531_153136.json --from=pnf_beta --keyring-backend=test --chain-id=pocket-beta --node=${SHANNON_BETA_URL} --yes --home=~/.pocket_prod --fees=200upokt
pocketd tx authz exec ./tokenomics_params_beta_20250531_152843.json --from=pnf_beta --keyring-backend=test --chain-id=pocket-beta --node=${SHANNON_BETA_URL} --yes --home=~/.pocket_prod --fees=200upokt

# MainNet

pocketd tx authz exec ./slashing_params_beta_20250531_153448.json --from=grove_mainnet_genesis --keyring-backend=test --chain-id=pocket --node=https://shannon-testnet-grove-rpc.poktroll.com --yes --home=~/.pocket_prod --fees=200upokt
