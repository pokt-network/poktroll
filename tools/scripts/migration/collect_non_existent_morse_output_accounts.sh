#!/usr/bin/env bash

# Script to collect non-existent morse output accounts for migration
# This script identifies validator output addresses that don't have corresponding morse claimable accounts

set -eo pipefail

# Declare constants
DEFAULT_INPUT_FILE="morse_state_export_170616_2025-06-03.json"

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source common utilities
source "$SCRIPT_DIR/common_bash_utils.sh"

# Function to display help information
show_help() {
  cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Collect non-existent morse output accounts for migration.

This script analyzes Morse state exports to identify validator output addresses
that don't have corresponding morse claimable accounts, then generates zero-balance
morse claimable accounts for those missing addresses.

OPTIONS:
    --input FILE           Input morse state export file (default: morse_state_export_170616_2025-06-03.json)
    --testnet              Include TestNet data in addition to MainNet
    --print-counts         Only print account counts instead of generating JSON
    --help, -h             Show this help message

EXAMPLES:
    $(basename "$0")                                    # Run with default input file
    $(basename "$0") --input custom_export.json        # Run with custom input file
    $(basename "$0") --testnet                          # Run with MainNet and TestNet data
    $(basename "$0") --print-counts                     # Show only count statistics

FULL EXAMPLE:
    tools/scripts/migration/collect_non_existent_morse_output_accounts.sh --run --input morse_state_export_170616_2025-06-03.json --output results.json

FILES USED:
    - Default input: morse_state_export_170616_2025-06-03.json (MainNet export)
    - TestNet file: morse_state_export_179148_2025-06-01.json (if --testnet used; in addition to MainNet export)
    - Import message files: msg_import_morse_accounts_*.json (auto-generated based on input)

OUTPUT:
    JSON array of zero-balance morse claimable accounts for missing addresses

EOF
}


# Parse all command line arguments
parse_args "$@"

# Set default values options
PRINT_COUNTS=false # Whether to print line counts
TESTNET=false      # Whether to consider MainNet OR TestNet data ONLY
INPUT_FILE=""     # Explicit list of input files
OUTPUT_FILE=""

# Parse additional arguments for input and output files
while [[ $# -gt 0 ]]; do
  case $1 in
  --count)
    PRINT_COUNTS=true # Enable count printing when --count flag is passed
    shift
    ;;
  --testnet)
    TESTNET=true # Enable testnet mode when --testnet flag is passed
    shift
    ;;
  --input)
    INPUT_FILE="$2"
    shift 2
    ;;
  *)
    # Skip other arguments (handled by parse_args)
    shift
    ;;
  esac
done

# Function to extract non-custodial morse output addresses from state export
# Selects validators where output_address is set and different from validator address
get_raw_non_custodial_morse_output_addresses() {
  jq -r '[.app_state.pos.validators[]|select(.output_address != "" and .output_address != .address)]|map(.output_address)[]' "$1"
}

# Function to get all morse source addresses from claimable accounts
get_all_raw_morse_claimable_account_src_addresses() {
  jq -r '.morse_account_state.accounts|map(.morse_src_address)[]' "$1"
}

# Function to create zero-balance morse claimable accounts for given addresses
# Takes a JSON array of addresses and creates account objects with zero balances
zero_balance_morse_claimable_accounts_for_addresses() {
  jq -r '.|map({morse_src_address: ., unstaked_balance: "0upokt", supplier_stake: "0upokt", application_stake: "0upokt", claimed_at_height: 0, shannon_dest_address: "", morse_output_address: ""})' <<<"$1"
}

# Function to extract base filename without extension for generating related filenames
get_base_filename() {
  local filepath="$1"
  local filename=$(basename "$filepath")
  echo "${filename%.*}" # Remove extension
}

# Function to generate import message filename based on input file
get_import_message_filename() {
  local input_file="$1"
  local base_name=$(get_base_filename "$input_file")

  # Extract the date portion (assuming format like morse_state_export_170616_2025-06-03.json)
  if [[ "$base_name" =~ morse_state_export_([0-9]+_[0-9]{4}-[0-9]{2}-[0-9]{2}) ]]; then
    local date_part="${BASH_REMATCH[1]}"
    if [ "$TESTNET" = true ]; then
      echo "msg_import_morse_accounts_m${date_part}_t179148.json"
    else
      echo "msg_import_morse_accounts_${date_part}.json"
    fi
  else
    # Fallback if pattern doesn't match
    if [ "$TESTNET" = true ]; then
      echo "msg_import_morse_accounts_m_$(get_base_filename "$input_file")_t179148.json"
    else
      echo "msg_import_morse_accounts_$(get_base_filename "$input_file").json"
    fi
  fi
}

# Function to output results to both stdout and file (if specified)
output_results() {
  local content="$1"
  local output_file="$2"

  # Always output to stdout
  echo "$content"

  # If output file is specified, also write to file
  if [ -n "$output_file" ]; then
    echo "$content" >"$output_file"
    echo "Results written to: $output_file" >&2
  fi
}

# Main execution function
run_script() {
  # Set default input file if not provided
  local input_file="${INPUT_FILE:-"$DEFAULT_INPUT_FILE"}"

  # Build full path for input file (assume it's in script directory if not absolute path)
  if [[ "$input_file" != /* ]]; then
    MORSE_STATE_EXPORT_PATH="$SCRIPT_DIR/$input_file"
  else
    MORSE_STATE_EXPORT_PATH="$input_file"
  fi

  # Check if input file exists
  if [ ! -f "$MORSE_STATE_EXPORT_PATH" ]; then
    echo "Error: Input file '$MORSE_STATE_EXPORT_PATH' not found" >&2
    exit 1
  fi

  # Get all morse output addresses from MainNet, convert to uppercase and deduplicate
  ALL_MORSE_OUTPUT_ADDRESSES=$(get_raw_non_custodial_morse_output_addresses "$MORSE_STATE_EXPORT_PATH" | to_uppercase | sort | uniq)

  # If testnet flag is set, merge MainNet and TestNet data
  if [ "$TESTNET" = true ]; then
    # Use the hardcoded TestNet file (since it's referenced by the default import message naming)
    TESTNET_MORSE_STATE_EXPORT_PATH="$SCRIPT_DIR/morse_state_export_TODO_IN_THIS_PR_UPDATE_DEFAULT_FILE"

    # Check if TestNet file exists
    if [ ! -f "$TESTNET_MORSE_STATE_EXPORT_PATH" ]; then
      echo "Warning: TestNet file '$TESTNET_MORSE_STATE_EXPORT_PATH' not found, skipping TestNet data" >&2
    else
      # Extract TestNet morse output addresses
      TESTNET_MORSE_OUTPUT_ADDRESSES=$(get_raw_non_custodial_morse_output_addresses "$TESTNET_MORSE_STATE_EXPORT_PATH" | to_uppercase | sort | uniq)

      # Combine MainNet and TestNet addresses, removing duplicates
      ALL_MORSE_OUTPUT_ADDRESSES=$(join_lists "$ALL_MORSE_OUTPUT_ADDRESSES" "$TESTNET_MORSE_OUTPUT_ADDRESSES" | sort | uniq)
    fi
  fi

  # Generate import message filename based on input file
  MSG_MORSE_IMPORT_ACCOUNTS_FILENAME=$(get_import_message_filename "$input_file")
  MSG_MORSE_IMPORT_ACCOUNTS_PATH="$SCRIPT_DIR/$MSG_MORSE_IMPORT_ACCOUNTS_FILENAME"

  # Check if import message file exists
  if [ ! -f "$MSG_MORSE_IMPORT_ACCOUNTS_PATH" ]; then
    echo "Error: Import message file '$MSG_MORSE_IMPORT_ACCOUNTS_PATH' not found" >&2
    exit 1
  fi

  # Get all existing morse claimable account source addresses
  ALL_MORSE_CLAIMABLE_ACCOUNT_SRC_ADDRESSES=$(get_all_raw_morse_claimable_account_src_addresses "$MSG_MORSE_IMPORT_ACCOUNTS_PATH" | tr '[:lower:]' '[:upper:]' | sort | uniq)

  # Find addresses that exist in morse output but not in claimable accounts
  MISSING_MORSE_ACCOUNT_ADDRESSES=$(diff_A_sub_B "$ALL_MORSE_OUTPUT_ADDRESSES" "$ALL_MORSE_CLAIMABLE_ACCOUNT_SRC_ADDRESSES")

  # Convert missing addresses to JSON array format
  MISSING_MORSE_ACCOUNT_ADDRESSES_JSON=$(lines_to_json_array "$MISSING_MORSE_ACCOUNT_ADDRESSES")

  # Generate zero-balance morse claimable accounts for missing addresses
  ZERO_BALANCE_MORSE_CLAIMABLE_ACCOUNTS_JSON=$(zero_balance_morse_claimable_accounts_for_addresses "$MISSING_MORSE_ACCOUNT_ADDRESSES_JSON")

  # If print counts flag is set, show statistics and exit
  if [ "$PRINT_COUNTS" = true ]; then
    local count_output="Total Non-Custodial Morse Accounts: $(echo "$ALL_MORSE_OUTPUT_ADDRESSES" | count_non_empty_lines)
Total Morse claimable accounts: $(echo "$ALL_MORSE_CLAIMABLE_ACCOUNT_SRC_ADDRESSES" | count_non_empty_lines)
Total missing MorseClaimableAccounts: $(echo "$MISSING_MORSE_ACCOUNT_ADDRESSES" | count_non_empty_lines)"

    echo "$count_output"
    exit 0
  fi

  # Output the generated zero-balance morse claimable accounts JSON
  echo "$ZERO_BALANCE_MORSE_CLAIMABLE_ACCOUNTS_JSON"
}

# Check if no arguments provided or help requested, show help by default
if [ $# -eq 0 ] || [[ "$1" == "--help" ]] || [[ "$1" == "-h" ]]; then
  show_help
  exit 0
fi

# Execute the main script functionality
run_script
