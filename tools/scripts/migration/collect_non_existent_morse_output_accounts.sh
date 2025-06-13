#!/usr/bin/env bash

set -eo pipefail
shopt -s nullglob

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common_bash_utils.sh"

# Default Morse state export heights; can be overridden with the --height and --testnet-height flags, respectively.
DEFAULT_MAINNET_HEIGHT="170616"
DEFAULT_TESTNET_HEIGHT="179148"

# Set default values for global flag variables
SHOW_USAGE=true # If left set, the help/usage text will be printed.
MAINNET_HEIGHT=$DEFAULT_MAINNET_HEIGHT
TESTNET=false # If true, consider the TestNet artifacts and ONLY output missing TestNet accounts
TESTNET_HEIGHT=$DEFAULT_TESTNET_HEIGHT

# Function to display help information
show_usage() {
  cat <<🚀
Usage: $(basename "$0") [OPTIONS]

Collect non-existent morse output accounts for migration.

This script analyzes Morse state exports to identify validator output addresses
that don't have corresponding morse claimable accounts, then generates zero-balance
morse claimable accounts for those missing addresses.

OPTIONS:
    --defaults                       ONLY generate and output the missing Morse accounts JSON for MainNet and TestNet heights (if none of --defaults, --height, or --testnet-height are specified, the help/usage text will be printed)
    --height HEIGHT                  Use the MainNet artifacts with the given height (required even if --testnet is specified)
    --testnet                        ONLY generate missing Morse accounts from TestNet artifacts
    --testnet-height TESTNET_HEIGHT  Use TestNet artifacts with the given height (only required if --testnet is specified)
    --help, -h                       Show this help message

EXAMPLES:
    $(basename "$0") --defaults                               # Run with default MainNet and TestNet artifacts
    $(basename "$0") --height 167639                          # Run with MainNet artifacts with height 167639
    $(basename "$0") --height 167639 --testnet-height 176966  # Run with MainNet artifacts with height 167639 and TestNet artifacts with height 176966

FULL EXAMPLES:
    tools/scripts/migration/collect_non_existent_morse_output_accounts.sh --defaults
    tools/scripts/migration/collect_non_existent_morse_output_accounts.sh --defaults --show-stats
    tools/scripts/migration/collect_non_existent_morse_output_accounts.sh --defaults --testnet
    tools/scripts/migration/collect_non_existent_morse_output_accounts.sh --defaults --height 169825 --testnet-height 179148

FILES USED:
    - Default MainNet state export: morse_state_export_170616_2025-06-03.json (MainNet snapshot)
    - Default TestNet state export: morse_state_export_179148_2025-06-01.json (TestNet snapshot)
    - Import message files: msg_import_morse_accounts_*.json (auto-generated based on MainNet and TestNet height(s))

OUTPUT:
    JSON array of zero-balance morse claimable accounts for missing addresses
🚀
}

while [[ $# -gt 0 ]]; do
  arg="$1"
  case $arg in
  --height)
    # Use the MainNet artifacts with the given height and don't show usage.
    MAINNET_HEIGHT="$2"
    SHOW_USAGE=false
    shift 2
    ;;
  --testnet)
    # Use TestNet artifacts with the default TestNet height and don't show usage.
    TESTNET=true
    shift
    ;;
  --testnet-height)
    # Use TestNet artifacts with the given height and don't show usage.
    TESTNET_HEIGHT="$2"
    TESTNET=true
    SHOW_USAGE=false
    shift 2
    ;;
  --defaults)
    # Set MainNet and TestNet heights to their respective defaults and don't show usage if --defaults is specified
    # NOTE: This DOES NOT set nor imply the --testnet flag.
    MAINNET_HEIGHT=$DEFAULT_MAINNET_HEIGHT
    TESTNET_HEIGHT=$DEFAULT_TESTNET_HEIGHT
    SHOW_USAGE=false
    shift
    ;;
  -h | --help)
    # Show usage if -h or --help is specified.
    SHOW_USAGE=true
    shift
    ;;
  *)
    # Ignore unrecognized arguments
    shift
    ;;
  esac
done

# Get the unquoted and newline-delimited string of all non-custodial Morse output addresses from the given state export file.
# All addresses are normalized to uppercase, sorted, and deduplicated.
#   $1 - Path to the Morse state export file
get_raw_non_custodial_morse_output_addresses() {
  jq -r '[.app_state.pos.validators[]|select(.output_address != "" and .output_address != .address)]|map(.output_address)[]' "$1" | to_uppercase | sort | uniq
}

# Get the unquoted and newline-delimited string of all Morse claimable accounts from the given import message file.
# All addresses are normalized to uppercase, sorted, and deduplicated.
#   $1 - Path to the Morse accounts import message file
get_all_raw_morse_claimable_account_src_addresses() {
  jq -r '.morse_account_state.accounts|map(.morse_src_address)[]' "$1" | to_uppercase | sort | uniq
}

# Generate a JSON array of zero-balance Morse claimable accounts for the given JSON array of Morse claimable accounts..
#   $1 - JSON array of Morse claimable accounts
zero_balance_morse_claimable_accounts_for_addresses_json() {
  echo "$1" | jq -r '.|map({morse_src_address: ., unstaked_balance: "0upokt", supplier_stake: "0upokt", application_stake: "0upokt", claimed_at_height: 0, shannon_dest_address: "", morse_output_address: ""})'
}

# Generate a JSON array of missing Morse claimable accounts for the MainNet state export with the given MainNet height.
collect_mainnet_missing_morse_accounts_json() {
  local mainnet_height="$1"

  local mainnet_morse_state_export_path
  if ! mainnet_morse_state_export_path=$(get_state_export_path_by_height "$mainnet_height"); then
    exit $?
  fi

  local msg_import_morse_accounts_path
  if ! msg_import_morse_accounts_path=$(get_import_message_path_by_height "$mainnet_height"); then
    exit $?
  fi

  # Convert missing addresses to JSON array format
  local missing_morse_account_addresses_json
  if ! missing_morse_account_addresses_json=$(collect_missing_morse_output_addresses_json "$mainnet_morse_state_export_path" "$msg_import_morse_accounts_path"); then
    exit $?
  fi

  # Generate and output zero-balance morse claimable accounts for missing addresses
  zero_balance_morse_claimable_accounts_for_addresses_json "$missing_morse_account_addresses_json"
}

# Generate a JSON array of missing Morse claimable accounts for the TestNet state export with the given MainNet height and TestNet heights.
collect_testnet_missing_morse_accounts_json() {
  local mainnet_height="$1"
  local testnet_height="$2"

  local mainnet_morse_state_export_path
  if ! mainnet_morse_state_export_path=$(get_state_export_path_by_height "$mainnet_height"); then
    exit $?
  fi

  local msg_import_morse_accounts_path
  if ! msg_import_morse_accounts_path=$(get_import_message_path_by_height "$mainnet_height" "$testnet_height"); then
    exit $?
  fi

  local testnet_morse_state_export_path
  if ! testnet_morse_state_export_path=$(get_state_export_path_by_height "$testnet_height"); then
    exit $?
  fi

  # Convert missing addresses to JSON array format
  local missing_morse_account_addresses_json
  if ! missing_morse_account_addresses_json=$(collect_missing_morse_output_addresses_json "$testnet_morse_state_export_path" "$msg_import_morse_accounts_path"); then
    exit $?
  fi

  # Generate zero-balance morse claimable accounts for missing addresses
  zero_balance_morse_claimable_accounts_for_addresses_json "$missing_morse_account_addresses_json"
}

# Generate a JSON array of missing Morse claimable accounts for the given Morse state export and import message files.
#   $1 - Path to the Morse state export file
#   $2 - Path to the Morse accounts import message file
collect_missing_morse_output_addresses_json() {
  morse_state_export_path="$1"
  msg_import_morse_accounts_path="$2"

  local expected_morse_output_addresses
  if ! expected_morse_output_addresses=$(get_raw_non_custodial_morse_output_addresses "$morse_state_export_path"); then
    exit $?
  fi

  local all_morse_claimable_account_src_addresses
  if ! all_morse_claimable_account_src_addresses=$(get_all_raw_morse_claimable_account_src_addresses "$msg_import_morse_accounts_path"); then
    exit $?
  fi

  local missing_morse_account_addresses
  if ! missing_morse_account_addresses=$(diff_A_sub_B "$expected_morse_output_addresses" "$all_morse_claimable_account_src_addresses"); then
    exit $?
  fi

  # Convert missing addresses to JSON array format
  lines_to_json_array "$missing_morse_account_addresses"
}

# Execute the main script functionality.
run() {
  if [ "$TESTNET" = true ]; then
    # ONLY generate missing Morse accounts from TestNet state exports.
    collect_testnet_missing_morse_accounts_json "$MAINNET_HEIGHT" "$TESTNET_HEIGHT"
    if [[ $? -ne 0 ]]; then
      exit $?
    fi
  else
    # ONLY generate missing Morse accounts from MainNet state exports.
    collect_mainnet_missing_morse_accounts_json "$MAINNET_HEIGHT"
    if [[ $? -ne 0 ]]; then
      exit $?
    fi
  fi
}

if [ "$SHOW_USAGE" = true ]; then
  show_usage
  exit 0
fi
run
