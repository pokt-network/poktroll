#!/bin/bash

# Script to automate network upgrades for different environments
#
# Usage: ./tools/scripts/upgrades/submit_upgrade.sh <environment> <version> [options]
#   <environment>: Required. One of: local, alpha, beta, main
#   <version>: Required. Version string (e.g., v0.1.2)
#   [options]: Optional flags:
#     --height-offset <blocks>: Number of blocks to add to current height. Default: 5
#     --keyring-backend <backend>: Keyring backend to use. Default: test
#     --home <path>: Home directory for pocketd. Default: ~/.pocket
#     --fees <amount>: Transaction fees. Default: 300upokt
#
# This script will:
# 1. Set up environment variables based on the network
# 2. Calculate and update the upgrade height in the transaction JSON
# 3. Provide copy-paste commands for submitting the upgrade
# 4. Provide copy-paste commands for monitoring the upgrade

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
BOLD='\033[1m'
REGULAR='\033[0m'
NC='\033[0m' # No Color

# Function to print colored output
print_step() {
    echo -e "${BLUE}==>${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_header() {
    echo -e "${PURPLE}$1${NC}"
}

print_command() {
    echo -e "${CYAN}$1${NC}"
}

# Check if command is provided
if [ -z "$1" ] || [ -z "$2" ] || [[ "$1" == "help" ]] || [[ "$1" == "--help" ]]; then
    echo "Usage: ./tools/scripts/upgrades/submit_upgrade.sh <environment> <version> [options]"
    echo ""
    echo "Arguments:"
    echo "  <environment>: Target environment (local, alpha, beta, main)"
    echo "  <version>: Version string (e.g., v0.1.2)"
    echo ""
    echo "Options:"
    echo "  --height-offset <blocks>: Number of blocks to add to current height. Default: 5"
    echo "  --keyring-backend <backend>: Keyring backend to use. Default: test"
    echo "  --home <path>: Home directory for pocketd. Default: ~/.pocket"
    echo "  --fees <amount>: Transaction fees. Default: 300upokt"
    echo "  --instruction-only: Show instructions without modifying the JSON file"
    echo ""
    echo "Examples:"
    echo "  ./tools/scripts/upgrades/submit_upgrade.sh alpha v0.1.2"
    echo "  ./tools/scripts/upgrades/submit_upgrade.sh beta v0.1.3 --height-offset 10"
    echo "  ./tools/scripts/upgrades/submit_upgrade.sh main v0.1.2 --instruction-only"
    exit 1
fi

ENVIRONMENT="$1"
VERSION="$2"
shift 2

# Default values
HEIGHT_OFFSET=5
KEYRING_BACKEND="test"
HOME_DIR="~/.pocket"
FEES="300upokt"
INSTRUCTION_ONLY=false

# Parse optional arguments
while [[ "$#" -gt 0 ]]; do
    case $1 in
    --height-offset)
        HEIGHT_OFFSET="$2"
        shift 2
        ;;
    --keyring-backend)
        KEYRING_BACKEND="$2"
        shift 2
        ;;
    --home)
        HOME_DIR="$2"
        shift 2
        ;;
    --fees)
        FEES="$2"
        shift 2
        ;;
    --instruction-only)
        INSTRUCTION_ONLY=true
        shift
        ;;
    *)
        echo "Unknown parameter passed: $1"
        exit 1
        ;;
    esac
done

# Validate environment and set Grafana dashboard link
case $ENVIRONMENT in
local)
    RPC_ENDPOINT="localhost:26657"
    FROM_ACCOUNT="pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw"
    CHAIN_ID="pocket"
    NODE_FLAG="--node=localhost:26657"
    GRAFANA_DASHBOARD="NA"
    ;;
alpha)
    RPC_ENDPOINT="https://shannon-testnet-grove-rpc.alpha.poktroll.com"
    FROM_ACCOUNT="pnf_alpha"
    CHAIN_ID="pocket-alpha"
    NODE_FLAG="--node=https://shannon-testnet-grove-rpc.alpha.poktroll.com"
    GRAFANA_DASHBOARD="https://grafana.poktroll.com/goto/6u7cD7PHg?orgId=1"
    ;;
beta)
    RPC_ENDPOINT="https://shannon-testnet-grove-rpc.beta.poktroll.com"
    FROM_ACCOUNT="pokt1f0c9y7mahf2ya8tymy8g4rr75ezh3pkklu4c3e"
    CHAIN_ID="pocket-beta"
    NODE_FLAG="--node=https://shannon-testnet-grove-rpc.beta.poktroll.com"
    GRAFANA_DASHBOARD="https://grafana.poktroll.com/goto/haNungjHg?orgId=1"
    ;;
main)
    RPC_ENDPOINT="https://shannon-grove-rpc.mainnet.poktroll.com"
    FROM_ACCOUNT="pokt18808wvw0h4t450t06uvauny8lvscsxjfyua7vh"
    CHAIN_ID="pocket"
    NODE_FLAG="--node=https://shannon-grove-rpc.mainnet.poktroll.com"
    GRAFANA_DASHBOARD="https://grafana.poktroll.com/goto/K3BXngjHR?orgId=1"
    ;;
*)
    print_error "Unknown environment '$ENVIRONMENT'. Use: local, alpha, beta, or main"
    exit 1
    ;;
esac

# Set upgrade transaction JSON path
UPGRADE_TX_JSON="tools/scripts/upgrades/upgrade_tx_${VERSION}_${ENVIRONMENT}.json"

# Check if the upgrade transaction file exists
if [ ! -f "$UPGRADE_TX_JSON" ]; then
    print_error "Upgrade transaction file not found: $UPGRADE_TX_JSON"
    print_warning "Please ensure the upgrade transaction JSON file exists before running this script."
    exit 1
fi

# Print header
print_header "=================================="
print_header " POCKET NETWORK UPGRADE SCRIPT "
print_header "=================================="
echo ""

echo -e "${BLUE}╔════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║${NC}         Configuration Summary              ${BLUE}║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════╝${NC}"
echo -e "  ${BOLD}Environment:${REGULAR} ${CYAN}$ENVIRONMENT${NC}"
echo -e "  ${BOLD}Version:${REGULAR} ${CYAN}$VERSION${NC}"
echo -e "  ${BOLD}RPC Endpoint:${REGULAR} ${CYAN}$RPC_ENDPOINT${NC}"
echo -e "  ${BOLD}From Account:${REGULAR} ${CYAN}$FROM_ACCOUNT${NC}"
echo -e "  ${BOLD}Chain ID:${REGULAR} ${CYAN}$CHAIN_ID${NC}"
echo -e "  ${BOLD}Upgrade TX JSON:${REGULAR} ${CYAN}$UPGRADE_TX_JSON${NC}"
echo -e "  ${BOLD}Height Offset:${REGULAR} ${CYAN}$HEIGHT_OFFSET blocks${NC}"
echo -e "  ${BOLD}Fees:${REGULAR} ${CYAN}$FEES${NC}"
if [ "$INSTRUCTION_ONLY" = true ]; then
    echo -e "  ${BOLD}Mode:${REGULAR} ${YELLOW}INSTRUCTION-ONLY (JSON will not be modified)${NC}"
fi
echo ""

# Export environment variables
echo ""
echo -e "${BLUE}╔════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║${NC}      Setting up environment variables      ${BLUE}║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════╝${NC}"
echo ""
print_command "export RPC_ENDPOINT=$RPC_ENDPOINT"
print_command "export UPGRADE_TX_JSON=\"$UPGRADE_TX_JSON\""
print_command "export NETWORK=$ENVIRONMENT"
print_command "export FROM_ACCOUNT=$FROM_ACCOUNT"
echo ""

# Get current height and calculate upgrade height
if [ "$INSTRUCTION_ONLY" = false ]; then
    echo ""
    echo -e "${BLUE}╔════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║${NC}        Calculating upgrade height          ${BLUE}║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════╝${NC}"

    # Get the current height
    echo ""
    print_command "Getting current height from network..."
    CURRENT_HEIGHT=$(pocketd q block --network=${ENVIRONMENT} -o json | tail -n +2 | jq -r '.header.height')

    if [ -z "$CURRENT_HEIGHT" ] || [ "$CURRENT_HEIGHT" = "null" ]; then
        print_error "Failed to get current height from network"
        exit 1
    fi

    UPGRADE_HEIGHT=$((CURRENT_HEIGHT + HEIGHT_OFFSET))
    print_success "Current height: ${RED}$CURRENT_HEIGHT${NC}"
    print_success "Upgrade height: ${RED}$UPGRADE_HEIGHT${NC} (current + $HEIGHT_OFFSET)"

    # Update the JSON file
    echo -e "Updating upgrade height in ${CYAN}$UPGRADE_TX_JSON${NC}"
    sed -i "" "s/\"height\": \"[^\"]*\"/\"height\": \"$UPGRADE_HEIGHT\"/" ${UPGRADE_TX_JSON}
    print_success "Updated upgrade height in transaction file"

    # Show the updated content
    echo ""
    echo -e "${BLUE}╔════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║${NC} Updated transaction file (for verification) ${BLUE}║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════╝${NC}"
    echo -e "${BOLD}File:${REGULAR} ${CYAN}$UPGRADE_TX_JSON${NC}"
    echo ""
    cat ${UPGRADE_TX_JSON}
    echo ""
else
    echo ""
    print_warning "Skipping height calculation and JSON modification in instruction-only mode"
fi

# Submit the transaction
echo ""
echo -e "${BLUE}╔════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║${NC}      Submit the upgrade transaction        ${BLUE}║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════╝${NC}"
echo ""
print_header "🚀 COPY-PASTE COMMAND TO SUBMIT UPGRADE:"
echo ""
echo -e "${CYAN}pocketd \\"
echo -e "    --keyring-backend=\"$KEYRING_BACKEND\" --home=\"$HOME_DIR\" \\"
echo -e "    --fees=$FEES --network=${ENVIRONMENT} \\"
echo -e "    tx authz exec ${UPGRADE_TX_JSON} --from=${FROM_ACCOUNT}${NC}"
echo ""

# Verification and monitoring commands
echo ""
echo -e "${BLUE}╔════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║${NC}   Verification and monitoring commands     ${BLUE}║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════╝${NC}"
echo ""
print_header "📋 COPY-PASTE COMMANDS FOR MONITORING:"
echo ""
# Grafana dashboard link for monitoring
if [ "$GRAFANA_DASHBOARD" != "NA" ]; then
    echo -e "${NC}📊 Monitor the upgrade via Grafana dashboard: ${CYAN}$GRAFANA_DASHBOARD${NC} 📊"
    echo ""
fi
echo -e "${NC}1. Watch the upgrade plan:${NC}"
echo -e "   ${CYAN}watch -n 5 \"pocketd query upgrade plan --network=${ENVIRONMENT}\"${NC}"
echo -e "   ${CYAN}pocketd query upgrade plan --network=${ENVIRONMENT} -o json | jq${NC}"
echo ""
echo -e "${NC}2. Watch node version:${NC}"
echo -e "   ${CYAN}watch -n 5 \"curl -s ${RPC_ENDPOINT}/abci_info | jq '.result.response.version'\"${NC}"
echo ""
echo -e "${NC}3. Watch the transaction (replace TX_HASH with actual hash after submission):${NC}"
echo -e "   ${CYAN}export TX_HASH=\"<REPLACE_WITH_ACTUAL_TX_HASH>\"${NC}"
echo -e "   ${CYAN}watch -n 5 \"pocketd query tx --type=hash $\{TX_HASH\} --network=${ENVIRONMENT}\"${NC}"
echo ""

# Post-upgrade checklist
echo ""
echo -e "${BLUE}╔════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║${NC}         Post-upgrade checklist             ${BLUE}║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════╝${NC}"
echo ""
print_header "✅ POST-UPGRADE CHECKLIST:"
echo ""
echo -e "1. Generate release notes using:"
echo -e "   ${CYAN}./tools/scripts/upgrades/prepare_upgrade_release_notes.sh $VERSION${NC}"
echo ""
echo -e "2. Update the GitHub release notes and set it as the latest release: ${CYAN}https://github.com/pokt-network/poktroll/releases${NC}"
echo ""
echo -e "3. Update the documentation: ${CYAN}docusaurus/docs/4_develop/upgrades/4_upgrade_list.md${NC}"
echo ""
echo -e "4. Create a snapshot of the network: ${CYAN}https://www.notion.so/buildwithgrove/Shannon-Snapshot-Playbook-1aea36edfff680bbb5a7e71c9846f63c?source=copy_link${NC}"
echo ""
if [ "$INSTRUCTION_ONLY" = false ]; then
    echo -e "5. Commit all updated files to main: ${CYAN}${UPGRADE_TX_JSON}${NC}"
else
    echo -e "5. Update and commit the upgrade JSON file: ${CYAN}${UPGRADE_TX_JSON}${NC}"
fi
echo ""
echo -e "6. Notify all exchanges on Telegram: ${CYAN}make telegram_release_notify${NC}"
echo ""
echo -e "7. Only proceed to the next environment after current upgrade succeeds (Alpha → Beta → MainNet)"
echo ""

# Upgrade cancellation section
echo ""
echo -e "${BLUE}╔════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║${NC}         Emergency: Cancel Upgrade          ${BLUE}║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════╝${NC}"
echo ""
print_header "🚨 COPY-PASTE COMMAND TO CANCEL UPGRADE (EMERGENCY ONLY):"
echo ""

# Set the cancel upgrade JSON based on environment
case $ENVIRONMENT in
local)
    CANCEL_UPGRADE_JSON="tools/scripts/upgrades/cancel_upgrade_alpha.json"
    ;;
alpha)
    CANCEL_UPGRADE_JSON="tools/scripts/upgrades/cancel_upgrade_alpha.json"
    ;;
beta)
    CANCEL_UPGRADE_JSON="tools/scripts/upgrades/cancel_upgrade_beta.json"
    ;;
main)
    CANCEL_UPGRADE_JSON="tools/scripts/upgrades/cancel_upgrade_main.json"
    ;;
esac

echo -e "${CYAN}pocketd \\\\"
echo -e "    --keyring-backend=\"$KEYRING_BACKEND\" --home=\"$HOME_DIR\" \\\\"
echo -e "    --fees=$FEES --network=${ENVIRONMENT} \\\\"
echo -e "    tx authz exec ${CANCEL_UPGRADE_JSON} --from=${FROM_ACCOUNT}${NC}"
echo ""
print_warning "⚠️  Use this command ONLY in emergency situations to cancel a pending upgrade!"
echo ""