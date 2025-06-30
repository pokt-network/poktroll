#!/bin/bash

# -----------------------------------------------
# shannon_query_unique_tx_msgs_and_events
# -----------------------------------------------
# Description:
#   Queries all transactions between two block heights and extracts:
#   1. All unique message types that start with "pocket"
#   2. All unique event types that start with "pocket"
#
# Arguments:
#   $1 - Minimum block height (exclusive)
#   $2 - Maximum block height (exclusive)
#   $3 - Network environment - must be one of: alpha, beta, main
#
# Example:
#   shannon_query_unique_tx_msgs_and_events 13000 37364 beta
#   shannon_query_unique_tx_msgs_and_events 115010 116550 main
#
function shannon_query_unique_tx_msgs_and_events() {
  if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    echo "shannon_query_unique_tx_msgs_and_events - Query unique message and event types"
    echo ""
    echo "DESCRIPTION:"
    echo "  Queries all transactions between two block heights and extracts:"
    echo "  1. All unique message types that start with 'pocket'"
    echo "  2. All unique event types that start with 'pocket'"
    echo ""
    echo "USAGE:"
    echo "  shannon_query_unique_tx_msgs_and_events <start_height> <end_height> <env>"
    echo ""
    echo "ARGUMENTS:"
    echo "  start_height    Minimum block height (exclusive)"
    echo "  end_height      Maximum block height (exclusive)"
    echo "  env             Network environment - must be one of: alpha, beta, main"
    echo ""
    echo "EXAMPLES:"
    echo "  shannon_query_unique_tx_msgs_and_events 13000 37364 beta"
    echo "  shannon_query_unique_tx_msgs_and_events 115010 116550 main"
    return 0
  fi

  if [[ $# -ne 3 ]]; then
    echo "Error: Invalid number of arguments."
    echo ""
    echo "Usage:"
    echo "  shannon_query_unique_tx_msgs_and_events <start_height> <end_height> <env (alpha|beta|main)>"
    echo ""
    echo "Example:"
    echo "  shannon_query_unique_tx_msgs_and_events 13000 37364 beta"
    echo "  shannon_query_unique_tx_msgs_and_events 115010 116550 main"
    echo ""
    echo "Use --help for more information."
    return 1
  fi

  local START_HEIGHT="$1"
  local END_HEIGHT="$2"
  local ENV="$3"

  if [[ "$ENV" != "alpha" && "$ENV" != "beta" && "$ENV" != "main" ]]; then
    echo "Error: Invalid environment. Must be one of: alpha, beta, main"
    return 1
  fi

  local TMP_FILE="/tmp/shannon_txs.json"

  echo "Querying transactions from height $START_HEIGHT to $END_HEIGHT on '$ENV' network..."

  pocketd query txs \
    --network="$ENV" --grpc-insecure=false \
    --query="tx.height > $START_HEIGHT AND tx.height < $END_HEIGHT" \
    --limit "10000000000" --page 1 -o json >"$TMP_FILE"

  echo ""
  echo "## Unique message types starting with 'pocket':"
  jq -r '.txs[].tx.body.messages[]."@type"' "$TMP_FILE" |
    grep '^/pocket' |
    sort -u |
    sed 's/^/- /'

  echo ""
  echo "## Unique event types starting with 'pocket':"
  jq -r '.txs[].events[].type' "$TMP_FILE" |
    grep '^pocket' |
    sort -u |
    sed 's/^/- /'
}

# -----------------------------------------------
# shannon_query_tx_messages
# -----------------------------------------------
# Description:
#   Queries transactions between two block heights on a given environment,
#   filtering by message type (required) and optionally by sender address.
#   Outputs height, txhash, message type, signer(s), and list of event types.
#
# Arguments:
#   $1 - Minimum block height (exclusive)
#   $2 - Maximum block height (exclusive)
#   $3 - Message type (e.g., /pocket.supplier.MsgUnstakeSupplier)
#   $4 - (Optional) Sender address (use "" if not filtering by sender)
#   $5 - Network environment - must be one of: alpha, beta, main
#
# Example:
#   shannon_query_tx_messages 13000 37364 /pocket.supplier.MsgUnstakeSupplier pokt1abc123... beta
#   shannon_query_tx_messages 13000 37364 /pocket.supplier.MsgUnstakeSupplier "" beta
#   shannon_query_tx_messages 115010 116550 /pocket.supplier.MsgUnstakeSupplier "" main
#
function shannon_query_tx_messages() {
  if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    echo "shannon_query_tx_messages - Query transactions by message type"
    echo ""
    echo "DESCRIPTION:"
    echo "  Queries transactions between two block heights on a given environment,"
    echo "  filtering by message type (required) and optionally by sender address."
    echo "  Outputs height, txhash, message type, signer(s), and list of event types."
    echo ""
    echo "USAGE:"
    echo "  shannon_query_tx_messages <start_height> <end_height> <message_type> <sender_or_empty> <env>"
    echo ""
    echo "ARGUMENTS:"
    echo "  start_height    Minimum block height (exclusive)"
    echo "  end_height      Maximum block height (exclusive)"
    echo "  message_type    Message type (e.g., /pocket.supplier.MsgUnstakeSupplier)"
    echo "  sender_or_empty (Optional) Sender address (use \"\" if not filtering by sender)"
    echo "  env             Network environment - must be one of: alpha, beta, main"
    echo ""
    echo "EXAMPLES:"
    echo "  shannon_query_tx_messages 13000 37364 /pocket.supplier.MsgUnstakeSupplier pokt1abc123... beta"
    echo "  shannon_query_tx_messages 13000 37364 /pocket.supplier.MsgUnstakeSupplier \"\" beta"
    echo "  shannon_query_tx_messages 115010 116550 /pocket.supplier.MsgUnstakeSupplier \"\" main"
    return 0
  fi

  if [[ $# -ne 5 ]]; then
    echo "Error: Invalid number of arguments."
    echo ""
    echo "Usage:"
    echo "  shannon_query_tx_messages <start_height> <end_height> <message_type> <sender_or_empty> <env (alpha|beta|main)>"
    echo ""
    echo "Example:"
    echo "  shannon_query_tx_messages 13000 37364 /pocket.supplier.MsgUnstakeSupplier pokt1abc123... beta"
    echo "  shannon_query_tx_messages 13000 37364 /pocket.supplier.MsgUnstakeSupplier \"\" beta"
    echo "  shannon_query_tx_messages 115010 116550 /pocket.supplier.MsgUnstakeSupplier \"\" main"
    echo ""
    echo "Use --help for more information."
    return 1
  fi

  local START="$1"
  local END="$2"
  local MSG_TYPE="$3"
  local SENDER="$4"
  local ENV="$5"

  if [[ "$ENV" != "alpha" && "$ENV" != "beta" && "$ENV" != "main" ]]; then
    echo "Error: Invalid environment. Must be one of: alpha, beta, main"
    return 1
  fi

  local TMP_FILE="/tmp/shannon_filtered_msgs.json"

  local QUERY="message.action='${MSG_TYPE}' AND tx.height > ${START} AND tx.height < ${END}"
  if [[ -n "$SENDER" ]]; then
    QUERY="${QUERY} AND message.sender='${SENDER}'"
  fi

  echo "Querying '$ENV' for message type '$MSG_TYPE' from height $START to $END..."
  [[ -n "$SENDER" ]] && echo "Sender filter: $SENDER"

  pocketd query txs \
    --network="$ENV" --grpc-insecure=false \
    --query="$QUERY" \
    --limit "10000000000" --page 1 -o json >"$TMP_FILE"

  echo ""
  echo "Results:"
  jq --arg MSG_TYPE "$MSG_TYPE" '
      .txs[] |
      {
        height,
        txhash,
        type: ($MSG_TYPE),
        signers: (
          .tx.body.messages[]
          | select(."@type" == $MSG_TYPE)
          | [.signer] // [.operator_address] // []
        ),
        event_types_emitted: (.events | map(.type) | unique)
      }
    ' "$TMP_FILE"
}

# -----------------------------------------------
# shannon_query_tx_events
# -----------------------------------------------
# Description:
#   Queries all transactions between a given block height range from the Pocket Network Shannon testnet,
#   filtering for a specific event type and returning matching events with decoded attributes,
#   along with the transaction height and hash.
#
# Arguments:
#   $1 - Start block height (exclusive)
#   $2 - End block height (exclusive)
#   $3 - Event type (e.g., pocket.supplier.EventSupplierUnbondingBegin)
#   $4 - Network environment - must be one of: alpha, beta, main
#
# Example:
#   shannon_query_tx_events 13000 37364 pocket.supplier.EventSupplierUnbondingBegin beta
#   shannon_query_tx_events 115010 116550 pocket.supplier.EventSupplierUnbondingBegin main
#
function shannon_query_tx_events() {
  if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    echo "shannon_query_tx_events - Query transactions by event type"
    echo ""
    echo "DESCRIPTION:"
    echo "  Queries all transactions between a given block height range from the Pocket Network Shannon testnet,"
    echo "  filtering for a specific event type and returning matching events with decoded attributes,"
    echo "  along with the transaction height and hash."
    echo ""
    echo "USAGE:"
    echo "  shannon_query_tx_events <start_height> <end_height> <event_type> <env>"
    echo ""
    echo "ARGUMENTS:"
    echo "  start_height    Start block height (exclusive)"
    echo "  end_height      End block height (exclusive)"
    echo "  event_type      Event type (e.g., pocket.supplier.EventSupplierUnbondingBegin)"
    echo "  env             Network environment - must be one of: alpha, beta, main"
    echo ""
    echo "EXAMPLES:"
    echo "  shannon_query_tx_events 13000 37364 pocket.supplier.EventSupplierUnbondingBegin beta"
    echo "  shannon_query_tx_events 115010 116550 pocket.supplier.EventSupplierUnbondingBegin main"
    return 0
  fi

  if [[ $# -ne 4 ]]; then
    echo "Error: Invalid number of arguments."
    echo ""
    echo "Usage:"
    echo "  shannon_query_tx_events <start_height> <end_height> <event_type> <env (alpha|beta|main)>"
    echo ""
    echo "Example:"
    echo "  shannon_query_tx_events 13000 37364 pocket.supplier.EventSupplierUnbondingBegin beta"
    echo "  shannon_query_tx_events 115010 116550 pocket.supplier.EventSupplierUnbondingBegin main"
    echo ""
    echo "Use --help for more information."
    return 1
  fi

  local START="$1"
  local END="$2"
  local EVENT_TYPE="$3"
  local ENV="$4"

  if [[ "$ENV" != "alpha" && "$ENV" != "beta" && "$ENV" != "main" ]]; then
    echo "Error: Invalid environment. Must be one of: alpha, beta, main"
    return 1
  fi

  local TMP_FILE="/tmp/event_filtered.json"

  echo "Querying all transactions from $START to $END on '$ENV'..."
  echo "Filtering for event type: $EVENT_TYPE"
  echo "--------------------------------------"

  pocketd query txs \
    --network="$ENV" --grpc-insecure=false \
    --query="tx.height > ${START} AND tx.height < ${END}" \
    --limit "10000000000" --page 1 -o json >"$TMP_FILE"

  jq --arg EVENT_TYPE "$EVENT_TYPE" '
      .txs[]
      | {
          height,
          txhash,
          events
        }
      | . as $tx
      | {
          filtered_events: (
            $tx.events
            | map(select(.type == $EVENT_TYPE))
            | map({
                height: $tx.height,
                txhash: $tx.txhash,
                event_type: .type,
                attributes: (
                  .attributes
                  | map(select(.key != "msg_index"))
                  | map({ (.key): .value })
                  | add
                  | with_entries(.value |= try fromjson catch .)
                )
              })
          )
        }
      | .filtered_events[]
    ' "$TMP_FILE"
}

# -----------------------------------------------
# shannon_query_block_events
# -----------------------------------------------
# Description:
#   Queries block results for a range of block heights from the Pocket Network Shannon testnet,
#   extracting both transaction events and finalize_block_events, with optional filtering
#   to ignore common/noisy event types.
#
# Arguments:
#   $1 - Start block height (inclusive)
#   $2 - End block height (inclusive)
#   $3 - Network environment - must be one of: alpha, beta, main
#   $4 - (Optional) Event type filter - if provided, only returns events of this type
#   $5 - (Optional) --include-ignored - if provided, includes default ignored event types
#
# Example:
#   shannon_query_block_events 115575 115580 main
#   shannon_query_block_events 115575 115580 main pocket.supplier.EventSupplierUnbondingBegin
#   shannon_query_block_events 115575 115580 main "" --include-ignored
#
function shannon_query_block_events() {
  if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    echo "shannon_query_block_events - Query block events from block results"
    echo ""
    echo "DESCRIPTION:"
    echo "  Queries block results for a range of block heights from the Pocket Network Shannon testnet,"
    echo "  extracting both transaction events and finalize_block_events, with optional filtering"
    echo "  to ignore common/noisy event types."
    echo ""
    echo "USAGE:"
    echo "  shannon_query_block_events <start_height> <end_height> <env> [event_type] [--include-ignored]"
    echo ""
    echo "ARGUMENTS:"
    echo "  start_height    Start block height (inclusive)"
    echo "  end_height      End block height (inclusive)"
    echo "  env             Network environment - must be one of: alpha, beta, main"
    echo "  event_type      (Optional) Event type filter - only returns events of this type"
    echo "  --include-ignored (Optional) Include default ignored event types"
    echo ""
    echo "IGNORED EVENT TYPES (by default):"
    echo "  - coin_spent, coin_received, transfer, message, tx"
    echo "  - commission, rewards, mint"
    echo ""
    echo "EXAMPLES:"
    echo "  shannon_query_block_events 115575 115580 main"
    echo "  shannon_query_block_events 115575 115580 main pocket.supplier.EventSupplierUnbondingBegin"
    echo "  shannon_query_block_events 115575 115580 main \"\" --include-ignored"
    return 0
  fi

  if [[ $# -lt 3 ]]; then
    echo "Error: Invalid number of arguments."
    echo ""
    echo "Usage:"
    echo "  shannon_query_block_events <start_height> <end_height> <env (alpha|beta|main)> [event_type] [--include-ignored]"
    echo ""
    echo "Example:"
    echo "  shannon_query_block_events 115575 115580 main"
    echo "  shannon_query_block_events 115575 115580 main pocket.supplier.EventSupplierUnbondingBegin"
    echo ""
    echo "Use --help for more information."
    return 1
  fi

  local START="$1"
  local END="$2"
  local ENV="$3"
  local EVENT_TYPE="${4:-}"
  local INCLUDE_IGNORED=""

  # Check for --include-ignored flag in any position
  if [[ "$4" == "--include-ignored" || "$5" == "--include-ignored" ]]; then
    INCLUDE_IGNORED="true"
  fi

  if [[ "$ENV" != "alpha" && "$ENV" != "beta" && "$ENV" != "main" ]]; then
    echo "Error: Invalid environment. Must be one of: alpha, beta, main"
    return 1
  fi

  # Validate block range
  if [[ $START -gt $END ]]; then
    echo "Error: Start height ($START) cannot be greater than end height ($END)"
    return 1
  fi

  local TMP_FILE="/tmp/block_events_filtered.json"

  # Default ignored event types
  local IGNORED_EVENTS='["coin_spent", "coin_received", "transfer", "message", "tx", "commission", "rewards", "mint"]'

  echo "Querying block events from $START to $END on '$ENV'..."
  if [[ -n "$EVENT_TYPE" ]]; then
    echo "Filtering for event type: $EVENT_TYPE"
  fi
  if [[ -n "$INCLUDE_IGNORED" ]]; then
    echo "Including ignored event types"
  else
    echo "Ignoring common event types: coin_spent, coin_received, transfer, message, tx, commission, rewards, mint"
  fi
  echo "--------------------------------------"

  # Initialize empty JSON array
  echo "[]" >"$TMP_FILE"

  # Loop through each block height
  for ((height = $START; height <= $END; height++)); do
    echo "Processing block $height..."

    local BLOCK_TMP="/tmp/block_$height.json"

    # Query block results for specific height
    if ! pocketd query block-results "$height" \
      --network="$ENV" --grpc-insecure=false \
      -o json >"$BLOCK_TMP" 2>/dev/null; then
      echo "Warning: Failed to query block $height, skipping..."
      continue
    fi

    # Process the block data and output events directly
    jq --argjson height "$height" \
      --arg event_type "$EVENT_TYPE" \
      --argjson ignored_events "$IGNORED_EVENTS" \
      --arg include_ignored "$INCLUDE_IGNORED" '
          # Function to redact large values
          def redact_large_values(obj):
            if obj | type == "object" then
              obj | with_entries(.value |= redact_large_values(.))
            elif obj | type == "array" then
              obj | map(redact_large_values(.))
            elif obj | type == "string" then
              if (obj | length) > 80 then
                "[REDACTED - " + (obj | length | tostring) + " bytes]"
              else
                obj
              end
            else
              obj
            end;

          # Function to filter and format events
          def process_events(events; txhash):
            events
            | map(select(
                # If event_type is specified, only include that type
                (if $event_type != "" then .type == $event_type else true end)
                and
                # If not including ignored events, filter them out
                (if $include_ignored == "true" then true else (.type as $t | $ignored_events | index($t) | not) end)
              ))
            | map({
                height: ($height | tostring),
                txhash: txhash,
                event_type: .type,
                attributes: (
                  .attributes
                  | map(select(.key != "msg_index"))
                  | map({ (.key): .value })
                  | add
                  | with_entries(.value |= try fromjson catch .)
                  | redact_large_values(.)
                )
              });

          # Process transaction events (use tx index as identifier for now)
          (.txs_results // [] | to_entries | map(
            .value.events as $events |
            .key as $tx_index |
            ("tx_" + ($tx_index | tostring)) as $txhash |
            process_events($events; $txhash)
          ) | flatten),

          # Process finalize block events (no txhash)
          process_events(.finalize_block_events // []; null)

          # Output each event individually
          | .[]
        ' "$BLOCK_TMP"

    # Clean up temporary block file
    rm -f "$BLOCK_TMP"
  done

  echo "Query completed."

  rm -f "$TMP_FILE"
}
