#!/usr/bin/env bash

#set -eo pipefail

PRINT_COUNTS=false
TESTNET=false

# Parse args
for arg in "$@"; do
  case $arg in
    --count)
      PRINT_COUNTS=true
      shift
      ;;
    --testnet)
      TESTNET=true
      shift
      ;;
    *)
      ;;
  esac
done

function get_all_raw_morse_output_addresses() {
  jq -r '[.app_state.pos.validators[]|select(.output_address != "")]|map(.output_address)[]' $1
}

function get_all_raw_morse_claimable_account_src_addresses() {
  jq -r '.morse_account_state.accounts|map(.morse_src_address)[]' $1
}

function lines_to_json_array() {
  # NOTE: -1 index drops the empty value after the trailing newline.
  jq -R -s 'split("\n")[:-1] | map(.)' <<< "$1"
}

function zero_balance_morse_claimable_accounts_for_addresses() {
  jq -r '.|map({morse_src_address: ., unstaked_balance: "0upokt", supplier_stake: "0upokt", application_stake: "0upokt", claimed_at_height: 0, shannon_dest_address: "", morse_output_address: ""})' <<< "$1"
}

SCRIPT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Use the state-shift day Morse MainNet snapshot.
MORSE_STATE_EXPORT_PATH="$SCRIPT_PATH/morse_state_export_170616_2025-06-03.json"
MSG_MORSE_IMPORT_ACCOUNTS_PATH="$SCRIPT_PATH/msg_import_morse_accounts_170616_2025-06-03.json"
ALL_MORSE_OUTPUT_ADDRESSES=$(get_all_raw_morse_output_addresses "$MORSE_STATE_EXPORT_PATH" | tr '[:lower:]' '[:upper:]' | sort | uniq)

# If the testnet flag is set, use the state-shift day Morse MainNet & TestNet merged snapshot.
if [ "$TESTNET" = true ]; then
  TESTNET_MORSE_STATE_EXPORT_PATH="$SCRIPT_PATH/morse_state_export_179148_2025-06-01.json"
  MSG_MORSE_IMPORT_ACCOUNTS_PATH="$SCRIPT_PATH/msg_import_morse_accounts_m170616_t179148.json"
  TESTNET_MORSE_OUTPUT_ADDRESSES=$(get_all_raw_morse_output_addresses "$TESTNET_MORSE_STATE_EXPORT_PATH" | tr '[:lower:]' '[:upper:]' | sort | uniq)
  ALL_MORSE_OUTPUT_ADDRESSES=$(echo "$ALL_MORSE_OUTPUT_ADDRESSES"\n"$TESTNET_MORSE_OUTPUT_ADDRESSES" | sort | uniq)
fi

ALL_MORSE_CLAIMABLE_ACCOUNT_SRC_ADDRESSES=$(get_all_raw_morse_claimable_account_src_addresses "$MSG_MORSE_IMPORT_ACCOUNTS_PATH" | tr '[:lower:]' '[:upper:]' | sort | uniq)
MISSING_MORSE_ACCOUNT_ADDRESSES=$(comm -23 <(echo "$ALL_MORSE_OUTPUT_ADDRESSES") <(echo "$ALL_MORSE_CLAIMABLE_ACCOUNT_SRC_ADDRESSES"))
MISSING_MORSE_ACCOUNT_ADDRESSES_JSON=$(lines_to_json_array "$MISSING_MORSE_ACCOUNT_ADDRESSES")
ZERO_BALANCE_MORSE_CLAIMABLE_ACCOUNTS_JSON=$(zero_balance_morse_claimable_accounts_for_addresses "$MISSING_MORSE_ACCOUNT_ADDRESSES_JSON")

if [ "$PRINT_COUNTS" = true ]; then
  echo "Total Morse output addresses: $(wc -l <<< "$ALL_MORSE_OUTPUT_ADDRESSES")"
  echo "Total Morse claimable accounts: $(wc -l <<< "$ALL_MORSE_CLAIMABLE_ACCOUNT_SRC_ADDRESSES")"
  echo "Total missing accounts: $(wc -l <<< "$MISSING_MORSE_ACCOUNT_ADDRESSES")"
else
  echo "$ZERO_BALANCE_MORSE_CLAIMABLE_ACCOUNTS_JSON"
fi
