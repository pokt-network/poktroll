#!/bin/bash

# Script to help update module parameters for different environments.
#
# Usage: ./update_params.sh <module_name> [options]
#   <module_name>: Required. The module to update (e.g., tokenomics, auth, bank).
#   [options]: Optional flags:
#     --env <environment>: Target environment (local, alpha, beta, main). Default: beta
#     --output-dir <dir>: Directory to save transaction files. Default: . (current directory)
#     --network <network>: Network flag for query. Default: uses --env value
#     --home <path>: Home directory for pocketd. Default: ~/.pocket
#     --no-prompt: Skip the edit prompt and just generate the template
#
# This script will:
# 1. Query current parameters for the specified module
# 2. Display them formatted for review
# 3. Generate a transaction template file
# 4. Prompt user to edit the parameters
# 5. Provide instructions for submitting the transaction

set -e

# Check if module name is provided
if [ -z "$1" ] || [[ "$1" == --* ]] || [[ "$1" == "help" ]] || [[ "$1" == "--help" ]]; then
    echo "Error: Module name is required as the first argument" >&2
    echo ""
    echo "Usage: ./update_params.sh <module_name> [options]"
    echo ""
    echo "Available modules:"
    echo "  auth          - Authentication parameters [Cosmos]"
    echo "  bank          - Bank module parameters [Cosmos]"
    echo "  tokenomics    - Tokenomics parameters (mint allocation, rewards) [Pocket]"
    echo "  gov           - Governance parameters [Cosmos]"
    echo "  staking       - Staking parameters [Cosmos]"
    echo "  slashing      - Slashing parameters [Cosmos]"
    echo "  distribution  - Distribution parameters [Cosmos]"
    echo "  mint          - Mint parameters [Cosmos]"
    echo "  params        - Global chain parameters [Cosmos]"
    echo "  application   - Application module parameters [Pocket]"
    echo "  gateway       - Gateway module parameters [Pocket]"
    echo "  service       - Service module parameters [Pocket]"
    echo "  supplier      - Supplier module parameters [Pocket]"
    echo "  session       - Session module parameters [Pocket]"
    echo "  proof         - Proof module parameters [Pocket]"
    echo "  shared        - Shared module parameters [Pocket]"
    echo ""
    echo "Available options:"
    echo "  --env <environment>: Target environment (local, alpha, beta, main). Default: beta"
    echo "  --output-dir <dir>: Directory to save transaction files. Default: . (current directory)"
    echo "  --network <network>: Network flag for query. Default: uses --env value"
    echo "  --home <path>: Home directory for pocketd. Default: ~/.pocket"
    echo "  --no-prompt: Skip the edit prompt and just generate the template"
    echo ""
    echo "Examples:"
    echo "  ./update_params.sh tokenomics"
    echo "  ./update_params.sh auth --env local"
    echo "  ./update_params.sh bank --env alpha --output-dir ./params"
    exit 1
fi

MODULE_NAME="$1"
shift # Remove module name from arguments

# Default values
ENVIRONMENT="beta"
OUTPUT_DIR="."
HOME_DIR="~/.pocket"
NETWORK=""
NO_PROMPT=false

# Parse optional arguments
while [[ "$#" -gt 0 ]]; do
    case $1 in
    --env)
        ENVIRONMENT="$2"
        shift 2
        ;;
    --output-dir)
        OUTPUT_DIR="$2"
        shift 2
        ;;
    --network)
        NETWORK="$2"
        shift 2
        ;;
    --home)
        HOME_DIR="$2"
        shift 2
        ;;
    --no-prompt)
        NO_PROMPT=true
        shift
        ;;
    *)
        echo "Unknown parameter passed: $1"
        exit 1
        ;;
    esac
done

# Use environment as network if network not explicitly set
if [ -z "$NETWORK" ]; then
    NETWORK="$ENVIRONMENT"
fi

# Define authorities and network configs for each environment
case $ENVIRONMENT in
local)
    AUTHORITY="pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t"
    FROM_KEY="pnf_local"
    CHAIN_ID="poktroll"
    NODE=""
    ;;
alpha)
    AUTHORITY="pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t"
    FROM_KEY="pnf_alpha"
    CHAIN_ID="pocket-alpha"
    NODE="--node https://alpha-testnet-rpc.poktroll.com:443"
    ;;
beta)
    AUTHORITY="pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t"
    FROM_KEY="pnf_beta"
    CHAIN_ID="pocket-beta"
    NODE="--node https://beta-testnet-rpc.poktroll.com:443"
    ;;
main)
    AUTHORITY="pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t"
    FROM_KEY="pnf_mainnet"
    CHAIN_ID="poktroll"
    NODE="--node https://mainnet-rpc.poktroll.com:443"
    ;;
*)
    echo "Error: Unknown environment '$ENVIRONMENT'. Use: local, alpha, beta, or main" >&2
    exit 1
    ;;
esac

# Create output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

# Build the query command
QUERY_CMD="./pocketd query $MODULE_NAME params --home=$HOME_DIR"
if [ "$NETWORK" != "local" ]; then
    QUERY_CMD="$QUERY_CMD --network=$NETWORK"
fi
QUERY_CMD="$QUERY_CMD -o json"

echo "========================================="
echo "Querying current $MODULE_NAME parameters"
echo "Environment: $ENVIRONMENT"
echo "Network: $NETWORK"
echo "Command: $QUERY_CMD"
echo "========================================="
echo ""

# Query current parameters
echo "Current parameters:"
CURRENT_PARAMS=$(eval $QUERY_CMD)
if [ $? -ne 0 ]; then
    echo "Error: Failed to query parameters for module '$MODULE_NAME'" >&2
    exit 1
fi

# Display current parameters with nice formatting
echo "$CURRENT_PARAMS" | jq '.'
echo ""

# Extract just the params object
PARAMS_ONLY=$(echo "$CURRENT_PARAMS" | jq '.params')

# Generate timestamp for unique filename
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
OUTPUT_FILE="$OUTPUT_DIR/${MODULE_NAME}_params_${ENVIRONMENT}_${TIMESTAMP}.json"

# Generate the transaction template
cat >"$OUTPUT_FILE" <<EOF
{
  "body": {
    "messages": [
      {
        "@type": "/pocket.${MODULE_NAME}.MsgUpdateParams",
        "authority": "$AUTHORITY",
        "params": $(echo "$PARAMS_ONLY" | jq '.')
      }
    ]
  }
}
EOF

echo "========================================="
echo "Transaction template created: $OUTPUT_FILE"
echo "========================================="
echo ""

if [ "$NO_PROMPT" = false ]; then
    echo "The transaction file has been created with current parameters."
    echo "You should now:"
    echo "1. Edit the file to update the desired parameter values"
    echo "2. Review your changes carefully"
    echo ""
    read -p "Press Enter to open the file for editing (or Ctrl+C to skip)..."

    # Try to open with common editors
    if command -v windsurf >/dev/null 2>&1; then
        windsurf "$OUTPUT_FILE"
    elif command -v code >/dev/null 2>&1; then
        code "$OUTPUT_FILE"
    elif command -v nano >/dev/null 2>&1; then
        nano "$OUTPUT_FILE"
    elif command -v vim >/dev/null 2>&1; then
        vim "$OUTPUT_FILE"
    elif command -v vi >/dev/null 2>&1; then
        vi "$OUTPUT_FILE"
    else
        echo "No suitable editor found. Please edit the file manually: $OUTPUT_FILE"
    fi

    echo ""
    echo "After editing, press Enter to continue..."
    read -p ""
fi

echo "========================================="
echo "Submit the transaction"
echo "========================================="
echo ""
echo "To submit your parameter update transaction, run:"
echo ""

# Generate the submission command based on environment
if [ "$ENVIRONMENT" = "local" ]; then
    echo "  make params_update_${MODULE_NAME} PARAM_FILE=$OUTPUT_FILE"
    echo ""
    echo "Or manually:"
    echo "  pocketd tx authz exec $OUTPUT_FILE --from=$FROM_KEY --keyring-backend=test --home=$HOME_DIR --chain-id=$CHAIN_ID --yes"
else
    echo "  pocketd tx authz exec $OUTPUT_FILE --from=$FROM_KEY --keyring-backend=test --chain-id=$CHAIN_ID $NODE --yes"
    echo ""
    echo "Or with network flag:"
    echo "  pocketd tx authz exec $OUTPUT_FILE --from=$FROM_KEY --keyring-backend=test --network=$ENVIRONMENT --yes"
fi

echo ""
echo "Template file location: $OUTPUT_FILE"
echo ""
echo "⚠️  IMPORTANT: Review your changes carefully before submitting!"
echo "⚠️  Parameter updates affect the entire network and cannot be easily reverted."
echo ""

# Show a preview of what will be submitted
echo "========================================="
echo "Transaction preview:"
echo "========================================="
cat "$OUTPUT_FILE" | jq '.'
