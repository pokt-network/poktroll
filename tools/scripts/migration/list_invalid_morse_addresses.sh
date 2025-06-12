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
SHOW_STATS=false # If true, ONLY print account statistics
SHOW_ADDRS=false # If true, ONLY print the collected invalid addresses; reacts to the --testnet flag
SHOW_TESTNET_ADDRS=false # If true, ONLY print TestNet addresses (MainNet + TestNet export state) which are included in the invalid addresses allowlist
MAINNET_HEIGHT=$DEFAULT_MAINNET_HEIGHT
TESTNET=false # If true, consider the TestNet artifacts and ONLY output missing TestNet accounts
TESTNET_HEIGHT=$DEFAULT_TESTNET_HEIGHT

# Function to display help information
show_usage() {
  cat <<🚀
Usage: $(basename "$0") [OPTIONS]

Collect invalid Morse addresses for migration allowlist inclusion.

This script analyzes Morse state exports to identify invalid addresses of various actor types,
then generates Go code snippets that can be included in the recovery allowlist for on-chain migration.

OPTIONS:
    --defaults                       Use default MainNet and TestNet heights.
    --height HEIGHT                  Use MainNet state export with the specified height.
    --testnet                        Also include TestNet data in collection and output.
    --testnet-height TESTNET_HEIGHT  Use TestNet state export with the specified height.
    --show-stats                     Print only account statistics by actor type (no allowlist output).
    --show-addresses-only            Output only the raw invalid address list (newline-delimited).
    -h, --help                       Show this help message.

EXAMPLES:
    $(basename "$0") --defaults                               # Use default MainNet + TestNet state exports
    $(basename "$0") --height 169825                           # Only process MainNet export at given height
    $(basename "$0") --height 169825 --testnet-height 179148   # Process both MainNet and TestNet exports

FULL EXAMPLES:
    tools/scripts/migration/collect_invalid_morse_addresses.sh --defaults
    tools/scripts/migration/collect_invalid_morse_addresses.sh --height 169825 --show-stats
    tools/scripts/migration/collect_invalid_morse_addresses.sh --height 169825 --testnet --show-addresses-only

FILES USED:
    - Default MainNet state export: morse_state_export_${DEFAULT_MAINNET_HEIGHT}_2025-06-03.json
    - Default TestNet state export: morse_state_export_${DEFAULT_TESTNET_HEIGHT}_2025-06-01.json
    - State export filenames must match: morse_state_export_<height>_*.json

OUTPUT:
    If --show-addresses-only is used:
        - Newline-delimited list of all invalid addresses from the relevant export(s)

    If --show-stats is used:
        - Total invalid address counts grouped by actor type (account, application, supplier, etc.)

    Otherwise:
        - Formatted Go code diff to be included in x/migration/recovery/recovery_allowlist.go

🚀
}

while [[ $# -gt 0 ]]; do
  arg="$1"
  case $arg in
  --show-addresses-only)
    # Output only the collected invalid addresses; reacts to the --testnet flag.
    SHOW_ADDRS=true
    shift
    ;;
  --show-stats)
    # Output only account statistics instead of generating JSON
    SHOW_STATS=true
    shift
    ;;
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
  -h|--help)
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

function get_raw_invalid_morse_account_addresses() {
  jq -r '[.app_state.auth.accounts[]|select(.type == "posmint/Account")|select(.value.address|length != 40)]|map(.value.address)[]' "$1"
}

function get_raw_invalid_morse_application_addresses() {
  jq -r '[.app_state.application.applications[]|select(.address|length != 40)]|map(.address)[]' "$1"
}

function get_raw_invalid_morse_supplier_addresses() {
  jq '[.app_state.pos.validators[]|select(.address|length != 40)]|map(.address)[]' "$1"
}

function get_raw_invalid_non_custodial_morse_owner_addresses() {
  jq -r '[.app_state.pos.validators[]|select(.output_address != .address and .output_address != "" and (.output_address|length != 40))]|map(.output_address)[]' "$1"
}


mainnet_state_export_path="$SCRIPT_DIR/morse_state_export_170616_2025-06-03.json"

recovery_allowlist_diff_template=$(cat <<EOF
  ...
  // IsMorseAddressRecoverable checks if a given address exists in any of the
  // allowlists for Morse address recovery.
  func IsMorseAddressRecoverable(address string) bool {
    ...
+   // Check if the address is in the invalid addresses allowlist
+   if listContainsTarget(invalidAddressesAllowlist, address) {
+     return true
+   }
    ...
  }

  // Ensure that all address slices are sorted in ascending order for use in binary search.
  func init() {
    ...
+   sort.Strings(invalidAddressesAllowlist)
    ...
  }

+ var invalidAddressesAllowlist = []string{
{{.InvalidAddresses}}
+ }
EOF
)

# Collect new-line delimited invalid addresses for each type of actor/account.
# MUST NOT be called in a subshell because it sets the following global variables:
# - INVALID_MORSE_ACCOUNT_ADDRESSES
# - INVALID_MORSE_APPLICATION_ADDRESSES
# - INVALID_MORSE_SUPPLIER_ADDRESSES
# - INVALID_NON_CUSTODIAL_MORSE_OWNER_ADDRESSES
collect_invalid_morse_account_addresses() {
  mainnet_state_export_path="$1"
  testnet_state_export_path="$2"

#  echo "mainnet_state_export_path: $mainnet_state_export_path" >&2
#  echo "testnet_state_export_path: $testnet_state_export_path" >&2

  if ! INVALID_MORSE_ACCOUNT_ADDRESSES=$(get_raw_invalid_morse_account_addresses "$mainnet_state_export_path"); then
    exit $?
  fi

  if ! INVALID_MORSE_APPLICATION_ADDRESSES=$(get_raw_invalid_morse_application_addresses "$mainnet_state_export_path"); then
    exit $?
  fi

  if ! INVALID_MORSE_SUPPLIER_ADDRESSES=$(get_raw_invalid_morse_supplier_addresses "$mainnet_state_export_path"); then
    exit $?
  fi

  if ! INVALID_NON_CUSTODIAL_MORSE_OWNER_ADDRESSES=$(get_raw_invalid_non_custodial_morse_owner_addresses "$mainnet_state_export_path"); then
    exit $?
  fi

  # If the testnet flag is set, merge the state-shift day Morse MainNet & TestNet state exports.
  if [ "$testnet_state_export_path" != "" ]; then
    collect_testnet_invalid_morse_account_addresses "$testnet_state_export_path"
  fi
}

# Collect and merge TestNet invalid Morse addresses into the global variables.
# This function must NOT be run in a subshell, as it mutates global variables:
# - INVALID_MORSE_ACCOUNT_ADDRESSES
# - INVALID_MORSE_APPLICATION_ADDRESSES
# - INVALID_MORSE_SUPPLIER_ADDRESSES
# - INVALID_NON_CUSTODIAL_MORSE_OWNER_ADDRESSES
#
# Should only be invoked if the --testnet flag is active.
collect_testnet_invalid_morse_account_addresses() {
  # Collect testnet addresses in new local variables.
  local testnet_invalid_account_addresses
  if ! testnet_invalid_account_addresses=$(get_raw_invalid_morse_account_addresses "$testnet_state_export_path"); then
    exit $?
  fi

  local testnet_invalid_morse_application_addresses
  if ! testnet_invalid_morse_application_addresses=$(get_raw_invalid_morse_application_addresses "$testnet_state_export_path"); then
    exit $?
  fi

  local testnet_invalid_morse_supplier_addresses
  if ! testnet_invalid_morse_supplier_addresses=$(get_raw_invalid_morse_supplier_addresses "$testnet_state_export_path"); then
    exit $?
  fi

  local testnet_invalid_non_custodial_morse_owner_addresses
  if ! testnet_invalid_non_custodial_morse_owner_addresses=$(get_raw_invalid_non_custodial_morse_owner_addresses "$testnet_state_export_path"); then
    exit $?
  fi

  # Append to respective global variables.
  if ! INVALID_MORSE_ACCOUNT_ADDRESSES=$(join_lists "$INVALID_MORSE_ACCOUNT_ADDRESSES" "$testnet_invalid_account_addresses"); then
    exit $?
  fi

  if ! INVALID_MORSE_APPLICATION_ADDRESSES=$(join_lists "$INVALID_MORSE_APPLICATION_ADDRESSES" "$testnet_invalid_morse_application_addresses"); then
    exit $?
  fi

  if ! INVALID_MORSE_SUPPLIER_ADDRESSES=$(join_lists "$INVALID_MORSE_SUPPLIER_ADDRESSES" "$testnet_invalid_morse_supplier_addresses"); then
    exit $?
  fi

  if ! INVALID_NON_CUSTODIAL_MORSE_OWNER_ADDRESSES=$(join_lists "$INVALID_NON_CUSTODIAL_MORSE_OWNER_ADDRESSES" "$testnet_invalid_non_custodial_morse_owner_addresses"); then
    exit $?
  fi
}

# Print a formatted Go code diff that includes the invalid Morse addresses.
# This output is intended to be pasted into x/migration/recovery/recovery_allowlist.go.
#   $1 - Newline-delimited list of uppercase invalid Morse addresses.
print_invalid_morse_account_addresses() {
  all_invalid_morse_addresses="$1"

  echo "Compare and apply the following diff to x/migration/recovery/recovery_allowlist.go:"
  invalid_addresses_diff_lines=$(echo "$all_invalid_morse_addresses" | to_uppercase | sed 's/.*/+   "&",/')
  recovery_allowlist_diff="${recovery_allowlist_diff_template//'{{.InvalidAddresses}}'/$invalid_addresses_diff_lines}"
  echo "$recovery_allowlist_diff"

}

# Generate and print total counts of invalid addresses by actor type.
# Used when the --show-stats flag is set to true.
print_account_stats() {
  echo "Total invalid Morse accounts: $(echo "$INVALID_MORSE_ACCOUNT_ADDRESSES" | count_non_empty_lines)"
  echo "Total invalid Morse applications: $(echo "$INVALID_MORSE_APPLICATION_ADDRESSES" | count_non_empty_lines)"
  echo "Total invalid Morse suppliers: $(echo "$INVALID_MORSE_SUPPLIER_ADDRESSES" | count_non_empty_lines)"
  echo "Total invalid non-custodial Morse owners: $(echo "$INVALID_NON_CUSTODIAL_MORSE_OWNER_ADDRESSES" | count_non_empty_lines)"
}

# Execute the main script functionality.
run() {
  local mainnet_morse_state_export_path
  if ! mainnet_morse_state_export_path=$(get_state_export_path_by_height "$MAINNET_HEIGHT"); then
    exit $?
  fi

  local testnet_morse_state_export_path
  if [ $TESTNET == true ]; then
    if ! testnet_morse_state_export_path=$(get_state_export_path_by_height "$TESTNET_HEIGHT"); then
      exit $?
    fi
  fi

  # Collect invalid addresses, by type, into global variables.
  collect_invalid_morse_account_addresses "$mainnet_morse_state_export_path" "$testnet_morse_state_export_path"

  if [ $SHOW_STATS = true ]; then
    print_account_stats
    exit 0
  fi

  # Join all invalid addresses into a single new-line delimited string.
  local all_invalid_morse_addresses="$INVALID_MORSE_ACCOUNT_ADDRESSES"
  if ! all_invalid_morse_addresses="$(join_lists "$all_invalid_morse_addresses" "$INVALID_MORSE_APPLICATION_ADDRESSES")"; then
    exit $?
  fi

  if ! all_invalid_morse_addresses="$(join_lists "$all_invalid_morse_addresses" "$INVALID_MORSE_SUPPLIER_ADDRESSES")"; then
    exit $?
  fi

  if ! all_invalid_morse_addresses="$(join_lists "$all_invalid_morse_addresses" "$INVALID_NON_CUSTODIAL_MORSE_OWNER_ADDRESSES")"; then
    exit $?
  fi

  if [ $SHOW_ADDRS = true ]; then
    echo "$all_invalid_morse_addresses"
    exit 0
  fi

  print_invalid_morse_account_addresses "$all_invalid_morse_addresses"
}

if [ "$SHOW_USAGE" = true ]; then
  show_usage
  exit 0
fi
run
