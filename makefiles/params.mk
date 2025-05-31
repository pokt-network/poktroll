#!/bin/bash

# Script to help query and update module parameters for different environments.
#
# Usage: ./update_params.sh <command> [module_name] [options]
#   <command>: Required. One of:
#     query <module_name>  - Query parameters for a specific module
#     query-all           - Query parameters for all available modules
#     update <module_name> - Generate update transaction for a module
#   [options]: Optional flags:
#     --env <environment>: Target environment (local, alpha, beta, main). Default: beta
#     --output-dir <dir>: Directory to save transaction files. Default: . (current directory)
#     --network <network>: Network flag for query. Default: uses --env value
#     --home <path>: Home directory for pocketd. Default: ~/.pocket
#     --no-prompt: Skip the edit prompt and just generate the template (update only)
#
# This script can:
# 1. Query current parameters for a specific module or all modules
# 2. Display them in a pretty formatted output
# 3. Generate transaction template files for parameter updates
# 4. Provide instructions for submitting transactions

set -e

# Available modules list
AVAILABLE_MODULES=(
  "auth"
  "bank"
  "tokenomics"
  "gov"
  "staking"
  "slashing"
  "distribution"
  "mint"
  "params"
  "application"
  "gateway"
  "service"
  "supplier"
  "session"
  "proof"
  "shared"
)

# Module descriptions
declare -A MODULE_DESCRIPTIONS
MODULE_DESCRIPTIONS["auth"]="Authentication parameters [Cosmos]"
MODULE_DESCRIPTIONS["bank"]="Bank module parameters [Cosmos]"
MODULE_DESCRIPTIONS["tokenomics"]="Tokenomics parameters (mint allocation, rewards) [Pocket]"
MODULE_DESCRIPTIONS["gov"]="Governance parameters [Cosmos]"
MODULE_DESCRIPTIONS["staking"]="Staking parameters [Cosmos]"
MODULE_DESCRIPTIONS["slashing"]="Slashing parameters [Cosmos]"
MODULE_DESCRIPTIONS["distribution"]="Distribution parameters [Cosmos]"
MODULE_DESCRIPTIONS["mint"]="Mint parameters [Cosmos]"
MODULE_DESCRIPTIONS["params"]="Global chain parameters [Cosmos]"
MODULE_DESCRIPTIONS["application"]="Application module parameters [Pocket]"
MODULE_DESCRIPTIONS["gateway"]="Gateway module parameters [Pocket]"
MODULE_DESCRIPTIONS["service"]="Service module parameters [Pocket]"
MODULE_DESCRIPTIONS["supplier"]="Supplier module parameters [Pocket]"
MODULE_DESCRIPTIONS["session"]="Session module parameters [Pocket]"
MODULE_DESCRIPTIONS["proof"]="Proof module parameters [Pocket]"
MODULE_DESCRIPTIONS["shared"]="Shared module parameters [Pocket]"

# Check if command is provided
if [ -z "$1" ] || [[ "$1" == "help" ]] || [[ "$1" == "--help" ]]; then
  echo "Usage: ./update_params.sh <command> [module_name] [options]"
  echo ""
  echo "Commands:"
  echo "  query <module_name>  - Query parameters for a specific module"
  echo "  query-all           - Query parameters for all available modules"
  echo "  update <module_name> - Generate update transaction for a module"
  echo ""
  echo "Available modules:"
  for module in "${AVAILABLE_MODULES[@]}"; do
    printf "  %-12s - %s\n" "$module" "${MODULE_DESCRIPTIONS[$module]}"
  done
  echo ""
  echo "Available options:"
  echo "  --env <environment>: Target environment (local, alpha, beta, main). Default: beta"
  echo "  --output-dir <dir>: Directory to save transaction files. Default: . (current directory)"
  echo "  --network <network>: Network flag for query. Default: uses --env value"
  echo "  --home <path>: Home directory for pocketd. Default: ~/.pocket"
  echo "  --no-prompt: Skip the edit prompt and just generate the template (update only)"
  echo ""
  echo "Examples:"
  echo "  ./update_params.sh query tokenomics"
  echo "  ./update_params.sh query-all --env alpha"
  echo "  ./update_params.sh update tokenomics --env local"
  echo "  ./update_params.sh update auth --env beta --output-dir ./params"
  exit 1
fi

COMMAND="$1"
shift # Remove command from arguments

# For update command, module name is required
if [ "$COMMAND" = "update" ]; then
  if [ -z "$1" ] || [[ "$1" == --* ]]; then
    echo "Error: Module name is required for update command" >&2
    echo "Usage: ./update_params.sh update <module_name> [options]"
    exit 1
  fi
  MODULE_NAME="$1"
  shift # Remove module name from arguments
elif [ "$COMMAND" = "query" ]; then
  if [ -z "$1" ] || [[ "$1" == --* ]]; then
    echo "Error: Module name is required for query command" >&2
    echo "Usage: ./update_params.sh query <module_name> [options]"
    exit 1
  fi
  MODULE_NAME="$1"
  shift # Remove module name from arguments
elif [ "$COMMAND" = "query-all" ]; then
  # No module name needed for query-all
  MODULE_NAME=""
else
  echo "Error: Unknown command '$COMMAND'. Use: query, query-all, or update" >&2
  exit 1
fi

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

# Create output directory if it doesn't exist (only needed for update command)
if [ "$COMMAND" = "update" ]; then
  mkdir -p "$OUTPUT_DIR"
fi

# Function to query and display parameters for a single module
query_module_params() {
  local module=$1
  local show_header=${2:-true}

  # Build the query command
  local query_cmd="./pocketd query $module params --home=$HOME_DIR"
  if [ "$NETWORK" != "local" ]; then
    query_cmd="$query_cmd --network=$NETWORK"
  fi
  query_cmd="$query_cmd -o json"

  if [ "$show_header" = true ]; then
    echo "========================================="
    echo "Module: $module (${MODULE_DESCRIPTIONS[$module]:-Unknown module})"
    echo "Environment: $ENVIRONMENT"
    echo "Network: $NETWORK"
    echo "========================================="
  fi

  # Query parameters
  local params_output
  params_output=$(eval $query_cmd 2>/dev/null)
  local query_exit_code=$?

  if [ $query_exit_code -ne 0 ] || [ -z "$params_output" ]; then
    echo "❌ Failed to query parameters for module '$module'"
    if [ "$show_header" = true ]; then
      echo "   This module may not exist or may not have queryable parameters"
    fi
    return 1
  fi

  # Check if the response contains parameters
  local params_only
  params_only=$(echo "$params_output" | jq '.params' 2>/dev/null)

  if [ "$params_only" = "null" ] || [ -z "$params_only" ]; then
    echo "❌ No parameters found for module '$module'"
    return 1
  fi

  echo "✅ Parameters for $module:"
  echo "$params_only" | jq '.'
  echo ""

  return 0
}

# Function to query all modules
query_all_modules() {
  echo "========================================="
  echo "Querying parameters for all modules"
  echo "Environment: $ENVIRONMENT"
  echo "Network: $NETWORK"
  echo "========================================="
  echo ""

  local successful_modules=()
  local failed_modules=()

  for module in "${AVAILABLE_MODULES[@]}"; do
    echo "🔍 Checking module: $module..."
    if query_module_params "$module" false; then
      successful_modules+=("$module")
    else
      failed_modules+=("$module")
    fi
  done

  echo "========================================="
  echo "Query Summary"
  echo "========================================="
  echo "✅ Successfully queried ${#successful_modules[@]} modules: ${successful_modules[*]}"
  if [ ${#failed_modules[@]} -gt 0 ]; then
    echo "❌ Failed to query ${#failed_modules[@]} modules: ${failed_modules[*]}"
  fi
  echo ""
}

# Execute the requested command
case $COMMAND in
  "query")
    query_module_params "$MODULE_NAME"
    ;;
  "query-all")
    query_all_modules
    ;;
  "update")
    # Existing update logic starts here

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
    cat > "$OUTPUT_FILE" <<EOF
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
    ;;
esac