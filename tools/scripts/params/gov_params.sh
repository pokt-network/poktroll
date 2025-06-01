#!/bin/bash

# Script to help query and update module parameters for different environments.
#
# Usage: ./update_params.sh <command> [module_name] [options]
#   <command>: Required. One of:
#     query <module_name>     - Query parameters for a specific module
#     query-all              - Query parameters for all available modules
#     update <module_name>    - Generate update transaction for a module
#     export-params <module_name> - Export parameters to a specified file
#   [options]: Optional flags:
#     --env <environment>: Target environment (local, alpha, beta, main). Default: beta
#     --output-dir <dir>: Directory to save transaction files. Default: . (current directory)
#     --output-file <file>: Specific output file path (export-params only)
#     --network <network>: Network flag for query. Default: uses --env value
#     --home <path>: Home directory for pocketd. Default: ~/.pocket
#     --no-prompt: Skip the edit prompt and just generate the template (update only)
#
# This script can:
# 1. Query current parameters for a specific module or all modules
# 2. Display them in a pretty formatted output
# 3. Generate transaction template files for parameter updates
# 4. Export parameters to a specific file path for external tools
# 5. Provide instructions for submitting transactions

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
    "application"
    "gateway"
    "service"
    "supplier"
    "session"
    "proof"
    "shared"
)

# Cosmos modules (use cosmos.module.v1beta1.MsgUpdateParams)
COSMOS_MODULES=(
    "auth"
    "bank"
    "gov"
    "staking"
    "slashing"
    "distribution"
    "mint"
)

# Function to get module description
get_module_description() {
    case $1 in
    "auth") echo "Authentication parameters [Cosmos]" ;;
    "bank") echo "Bank module parameters [Cosmos]" ;;
    "tokenomics") echo "Tokenomics parameters (mint allocation, rewards) [Pocket]" ;;
    "gov") echo "Governance parameters [Cosmos]" ;;
    "staking") echo "Staking parameters [Cosmos]" ;;
    "slashing") echo "Slashing parameters [Cosmos]" ;;
    "distribution") echo "Distribution parameters [Cosmos]" ;;
    "mint") echo "Mint parameters [Cosmos]" ;;
    "application") echo "Application module parameters [Pocket]" ;;
    "gateway") echo "Gateway module parameters [Pocket]" ;;
    "service") echo "Service module parameters [Pocket]" ;;
    "supplier") echo "Supplier module parameters [Pocket]" ;;
    "session") echo "Session module parameters [Pocket]" ;;
    "proof") echo "Proof module parameters [Pocket]" ;;
    "shared") echo "Shared module parameters [Pocket]" ;;
    *) echo "Unknown module" ;;
    esac
}

# Function to get the correct message type for a module
get_message_type() {
    local module=$1

    # Check if it's a Cosmos module
    for cosmos_module in "${COSMOS_MODULES[@]}"; do
        if [ "$cosmos_module" = "$module" ]; then
            echo "/cosmos.${module}.v1beta1.MsgUpdateParams"
            return
        fi
    done

    # Default to Pocket module format
    echo "/pocket.${module}.MsgUpdateParams"
}

# Check if command is provided
if [ -z "$1" ] || [[ "$1" == "help" ]] || [[ "$1" == "--help" ]]; then
    echo "Usage: ./update_params.sh <command> [module_name] [options]"
    echo ""
    echo "Commands:"
    echo "  query <module_name>     - Query parameters for a specific module"
    echo "  query-all              - Query parameters for all available modules"
    echo "  update <module_name>    - Generate update transaction for a module"
    echo "  export-params <module_name> - Export parameters to a specified file"
    echo ""
    echo "Available modules:"
    for module in "${AVAILABLE_MODULES[@]}"; do
        printf "  %-12s - %s\n" "$module" "$(get_module_description "$module")"
    done
    echo ""
    echo "Available options:"
    echo "  --env <environment>: Target environment (local, alpha, beta, main). Default: beta"
    echo "  --output-dir <dir>: Directory to save transaction files. Default: . (current directory)"
    echo "  --output-file <file>: Specific output file path (export-params only)"
    echo "  --network <network>: Network flag for query. Default: uses --env value"
    echo "  --home <path>: Home directory for pocketd. Default: ~/.pocket"
    echo "  --no-prompt: Skip the edit prompt and just generate the template (update only)"
    echo ""
    echo "Examples:"
    echo "  ./update_params.sh query tokenomics"
    echo "  ./update_params.sh query-all --env alpha"
    echo "  ./update_params.sh update tokenomics --env local"
    echo "  ./update_params.sh update auth --env beta --output-dir ./params"
    echo "  ./update_params.sh export-params application --output-file tools/scripts/params/bulk_params/application_params.json"
    exit 1
fi

COMMAND="$1"
shift # Remove command from arguments

# For update and export-params commands, module name is required
if [ "$COMMAND" = "update" ] || [ "$COMMAND" = "export-params" ]; then
    if [ -z "$1" ] || [[ "$1" == --* ]]; then
        echo "Error: Module name is required for $COMMAND command" >&2
        echo "Usage: ./update_params.sh $COMMAND <module_name> [options]"
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
    echo "Error: Unknown command '$COMMAND'. Use: query, query-all, update, or export-params" >&2
    exit 1
fi

# Default values
ENVIRONMENT="beta"
OUTPUT_DIR="."
OUTPUT_FILE=""
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
    --output-file)
        OUTPUT_FILE="$2"
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
    AUTHORITY="pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw"
    FROM_KEY="pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw"
    CHAIN_ID="pocket"
    NODE="--node=localhost:26657"
    ;;
alpha)
    AUTHORITY="pokt1r6ja6rz6rpae58njfrsgs5n5sp3r36r2q9j04h"
    FROM_KEY="pnf_alpha"
    CHAIN_ID="pocket-alpha"
    NODE="--node=https://shannon-testnet-grove-rpc.alpha.poktroll.com"
    ;;
beta)
    AUTHORITY="pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t"
    FROM_KEY="pokt1f0c9y7mahf2ya8tymy8g4rr75ezh3pkklu4c3e"
    CHAIN_ID="pocket-beta"
    NODE="--node=https://shannon-testnet-grove-rpc.beta.poktroll.com"
    ;;
main)
    AUTHORITY="pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t"
    FROM_KEY="pokt18808wvw0h4t450t06uvauny8lvscsxjfyua7vh"
    CHAIN_ID="pocket"
    NODE="--node=https://shannon-grove-rpc.mainnet.poktroll.com"
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
        echo "Module: $module ($(get_module_description "$module"))"
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

# Function to export parameters to a specific file
export_module_params() {
    local module=$1
    local output_file=$2

    # Build the query command
    local query_cmd="./pocketd query $module params --home=$HOME_DIR"
    if [ "$NETWORK" != "local" ]; then
        query_cmd="$query_cmd --network=$NETWORK"
    fi
    query_cmd="$query_cmd -o json"

    echo "========================================="
    echo "Exporting $module parameters"
    echo "Environment: $ENVIRONMENT"
    echo "Network: $NETWORK"
    echo "Output file: $output_file"
    echo "========================================="
    echo ""

    # Query current parameters
    local current_params
    current_params=$(eval $query_cmd)
    if [ $? -ne 0 ]; then
        echo "Error: Failed to query parameters for module '$module'" >&2
        exit 1
    fi

    # Extract just the params object
    local params_only
    params_only=$(echo "$current_params" | jq '.params')

    if [ "$params_only" = "null" ] || [ -z "$params_only" ]; then
        echo "❌ No parameters found for module '$module'"
        exit 1
    fi

    # Create the directory if it doesn't exist
    local output_dir
    output_dir=$(dirname "$output_file")
    mkdir -p "$output_dir"

    # Get the correct message type for this module
    local message_type
    message_type=$(get_message_type "$module")

    # Generate the full transaction structure
    local transaction_content
    transaction_content=$(
        cat <<EOF
{
  "body": {
    "messages": [
      {
        "@type": "$message_type",
        "authority": "$AUTHORITY",
        "params": $(echo "$params_only" | jq '.')
      }
    ]
  }
}
EOF
    )

    # Write to file
    echo "$transaction_content" | jq '.' >"$output_file"

    if [ $? -eq 0 ]; then
        echo "✅ Successfully exported $module parameters to: $output_file"
        echo ""
        echo "Transaction structure:"
        echo "$transaction_content" | jq '.'
        echo ""
        echo "The file contains the complete transaction structure with MsgUpdateParams."
        echo "You can modify the 'params' section as needed for your parameter updates."
        echo ""
        echo "Message type used: $message_type"
    else
        echo "❌ Failed to write parameters to: $output_file"
        exit 1
    fi
}

# Execute the requested command
case $COMMAND in
"query")
    query_module_params "$MODULE_NAME"
    ;;
"query-all")
    query_all_modules
    ;;
"export-params")
    # Validate that output file is specified
    if [ -z "$OUTPUT_FILE" ]; then
        echo "Error: --output-file is required for export-params command" >&2
        echo "Usage: ./update_params.sh export-params <module_name> --output-file <path>"
        echo "Example: ./update_params.sh export-params application --output-file tools/scripts/params/bulk_params/application_params.json"
        exit 1
    fi
    export_module_params "$MODULE_NAME" "$OUTPUT_FILE"
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
    OUTPUT_FILE_UPDATE="$OUTPUT_DIR/${MODULE_NAME}_params_${ENVIRONMENT}_${TIMESTAMP}.json"

    # Get the correct message type for this module
    MESSAGE_TYPE=$(get_message_type "$MODULE_NAME")

    # Generate the transaction template
    cat >"$OUTPUT_FILE_UPDATE" <<EOF
{
  "body": {
    "messages": [
      {
        "@type": "$MESSAGE_TYPE",
        "authority": "$AUTHORITY",
        "params": $(echo "$PARAMS_ONLY" | jq '.')
      }
    ]
  }
}
EOF

    echo "========================================="
    echo "Transaction template created: $OUTPUT_FILE_UPDATE"
    echo "Message type used: $MESSAGE_TYPE"
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
            windsurf "$OUTPUT_FILE_UPDATE"
        elif command -v code >/dev/null 2>&1; then
            code "$OUTPUT_FILE_UPDATE"
        elif command -v nano >/dev/null 2>&1; then
            nano "$OUTPUT_FILE_UPDATE"
        elif command -v vim >/dev/null 2>&1; then
            vim "$OUTPUT_FILE_UPDATE"
        elif command -v vi >/dev/null 2>&1; then
            vi "$OUTPUT_FILE_UPDATE"
        else
            echo "No suitable editor found. Please edit the file manually: $OUTPUT_FILE_UPDATE"
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
    echo "  pocketd tx authz exec $OUTPUT_FILE_UPDATE --from=$FROM_KEY --keyring-backend=test --chain-id=$CHAIN_ID $NODE --yes --home=$HOME_DIR --fees=200upokt"
    echo ""
    echo "Template file location: $OUTPUT_FILE_UPDATE"
    echo "Message type used: $MESSAGE_TYPE"
    echo ""
    echo "⚠️  IMPORTANT: Review your changes carefully before submitting!"
    echo "⚠️  Parameter updates affect the entire network and cannot be easily reverted."
    echo ""

    # Show a preview of what will be submitted
    echo "========================================="
    echo "Transaction preview:"
    echo "========================================="
    cat "$OUTPUT_FILE_UPDATE" | jq '.'
    ;;
esac
