#!/usr/bin/env bash

set -eo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

parse_args "$@"

function get_raw_invalid_morse_account_addresses() {
  jq -r '[.app_state.auth.accounts[]|select(.type == "posmint/Account")|select(.value.address|length != 40)]|map(.value.address)[]' $1
}

function get_raw_invalid_morse_application_addresses() {
  jq -r '[.app_state.application.applications[]|select(.address|length != 40)]|map(.address)[]' $1
}

function get_raw_invalid_morse_supplier_addresses() {
  jq '[.app_state.pos.validators[]|select(.address|length != 40)]|map(.address)[]' $1
}

function get_raw_invalid_non_custodial_morse_owner_addresses() {
  jq -r '[.app_state.pos.validators[]|select(.output_address != .address and .output_address != "" and (.output_address|length != 40))]|map(.output_address)[]' $1
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

MORSE_STATE_EXPORT_PATH="$SCRIPT_DIR/morse_state_export_170616_2025-06-03.json"

# Collect new-line delimited invalid addresses for each type of actor/account.
INVALID_MORSE_ACCOUNT_ADDRESSES="$(get_raw_invalid_morse_account_addresses "$MORSE_STATE_EXPORT_PATH")"
INVALID_MORSE_APPLICATION_ADDRESSES="$(get_raw_invalid_morse_application_addresses "$MORSE_STATE_EXPORT_PATH")"
INVALID_MORSE_SUPPLIER_ADDRESSES="$(get_raw_invalid_morse_supplier_addresses "$MORSE_STATE_EXPORT_PATH")"
INVALID_NON_CUSTODIAL_MORSE_OWNER_ADDRESSES="$(get_raw_invalid_non_custodial_morse_owner_addresses "$MORSE_STATE_EXPORT_PATH")"

# If the testnet flag is set, merge the state-shift day Morse MainNet & TestNet state exports.
if [ "$TESTNET" = true ]; then
  TESTNET_STATE_EXPORT_PATH="$SCRIPT_DIR/morse_state_export_179148_2025-06-01.json"
  INVALID_MORSE_ACCOUNT_ADDRESSES=$(join_lists "$INVALID_MORSE_ACCOUNT_ADDRESSES" "$(get_raw_invalid_morse_account_addresses "$TESTNET_STATE_EXPORT_PATH")")
  INVALID_MORSE_APPLICATION_ADDRESSES=$(join_lists "$INVALID_MORSE_APPLICATION_ADDRESSES" "$(get_raw_invalid_morse_application_addresses "$TESTNET_STATE_EXPORT_PATH")")
  INVALID_MORSE_SUPPLIER_ADDRESSES=$(join_lists "$INVALID_MORSE_SUPPLIER_ADDRESSES" "$(get_raw_invalid_morse_supplier_addresses "$TESTNET_STATE_EXPORT_PATH")")
  INVALID_NON_CUSTODIAL_MORSE_OWNER_ADDRESSES=$(join_lists "$INVALID_NON_CUSTODIAL_MORSE_OWNER_ADDRESSES" "$(get_raw_invalid_non_custodial_morse_owner_addresses "$TESTNET_STATE_EXPORT_PATH")")
fi

# Join all invalid addresses into a single new-line delimited string.
ALL_INVALID_MORSE_ADDRESSES_JSON="$INVALID_MORSE_ACCOUNT_ADDRESSES"
ALL_INVALID_MORSE_ADDRESSES_JSON="$(join_lists "$ALL_INVALID_MORSE_ADDRESSES_JSON" "$INVALID_MORSE_APPLICATION_ADDRESSES")"
ALL_INVALID_MORSE_ADDRESSES_JSON="$(join_lists "$ALL_INVALID_MORSE_ADDRESSES_JSON" "$INVALID_MORSE_SUPPLIER_ADDRESSES")"
ALL_INVALID_MORSE_ADDRESSES_JSON="$(join_lists "$ALL_INVALID_MORSE_ADDRESSES_JSON" "$INVALID_NON_CUSTODIAL_MORSE_OWNER_ADDRESSES")"

if [ "$PRINT_COUNTS" == true ]; then
  echo "Total invalid Morse accounts: $(echo "$INVALID_MORSE_ACCOUNT_ADDRESSES" | count_non_empty_lines)"
  echo "Total invalid Morse applications: $(echo "$INVALID_MORSE_APPLICATION_ADDRESSES" | count_non_empty_lines)"
  echo "Total invalid Morse suppliers: $(echo "$INVALID_MORSE_SUPPLIER_ADDRESSES" | count_non_empty_lines)"
  echo "Total invalid non-custodial Morse owners: $(echo "$INVALID_NON_CUSTODIAL_MORSE_OWNER_ADDRESSES" | count_non_empty_lines)"
  exit 0
fi

echo "Copy/paste the quoted and comma-delimited elements into an \`invalidAddressesAllowlist\` variable in x/migration/recovery/recovery_allowlist.go:"
# Print all invalid addresses as a JSON array to simplify copy/pasting (quotes and commas).
echo "$ALL_INVALID_MORSE_ADDRESSES_JSON" | jq -R -s 'split("\n")[:-1]'