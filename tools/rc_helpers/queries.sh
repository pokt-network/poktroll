#!/bin/bash

# Shannon Query Utilities - Consolidated
# TODO_IMPROVE: Reference these helpers in the proper documentation in dev.poktroll.com

# ===============================================
# HELP AND OVERVIEW
# ===============================================

function help() {
  LATEST_BLOCK=$(pocketd query block --network=main --grpc-insecure=false -o json | tail -n +2 | jq '.header.height')
  LATEST_BLOCK=$(($LATEST_BLOCK))
  LATEST_BLOCK_MINUS_100=$(($LATEST_BLOCK - 100))

  echo "=========================================="
  echo "Shannon Blockchain Query Utilities"
  echo "=========================================="
  echo ""
  echo "Available commands:"
  echo "  shannon_query_unique_tx_msgs_and_events  - Get unique message and event types"
  echo "  shannon_query_unique_block_events        - Get unique block events"
  echo "  shannon_query_tx_messages                - Query transactions by message type"
  echo "  shannon_query_tx_events                  - Query transactions by event type"
  echo "  shannon_query_block_events               - Query block events"
  echo "  shannon_query_unique_claim_suppliers     - Get unique claim supplier addresses"
  echo "  shannon_query_supplier_tx_events         - Get supplier-specific transaction events"
  echo "  shannon_query_supplier_block_events      - Get supplier-specific block events"
  echo "  shannon_query_application_block_events   - Get application-specific block events"
  echo ""
  echo "Current latest block on mainnet: $LATEST_BLOCK"
  echo ""
  echo "Quick start examples (using last 100 blocks - focus on Claim messages & events):"
  echo "  shannon_query_unique_tx_msgs_and_events $LATEST_BLOCK_MINUS_100 $LATEST_BLOCK main"
  echo "  shannon_query_unique_block_events $LATEST_BLOCK_MINUS_100 $LATEST_BLOCK main"
  echo "  shannon_query_tx_messages $LATEST_BLOCK_MINUS_100 $LATEST_BLOCK /pocket.proof.MsgCreateClaim \"\" main"
  echo "  shannon_query_tx_events $LATEST_BLOCK_MINUS_100 $LATEST_BLOCK pocket.proof.EventClaimCreated main"
  echo "  shannon_query_block_events $LATEST_BLOCK_MINUS_100 $LATEST_BLOCK main"
  echo "  shannon_query_unique_claim_suppliers $LATEST_BLOCK_MINUS_100 $LATEST_BLOCK main"
  echo "  shannon_query_supplier_tx_events $LATEST_BLOCK_MINUS_100 $LATEST_BLOCK pokt1hcfx7lx92p03r5gwjt7t7jk0j667h7rcvart9f main"
  echo "  shannon_query_supplier_block_events $LATEST_BLOCK_MINUS_100 $LATEST_BLOCK pokt1hcfx7lx92p03r5gwjt7t7jk0j667h7rcvart9f main"
  echo "  shannon_query_application_block_events $LATEST_BLOCK_MINUS_100 $LATEST_BLOCK pokt14tg8v3hns5tjefnmqs9u98jqjp6mw6wmwwmuh2 main"
  echo ""
  echo "Use --help with any command for detailed information"
  echo "=========================================="

  echo ""
  echo "TIPS:"
  echo "  Available event types can be found with:"
  echo "    find . -name \"*.proto\" -exec grep -h \"^message Event\" {} \\; | sed 's/^message \\(Event[^{]*\\).*/\\1/'"
  echo ""
  echo "  Available message types can be found with:"
  echo "    find . -name \"*.proto\" -exec grep -h \"^message Msg\" {} \\; | sed 's/^message \\(Msg[^{]*\\).*/\\1/' | head -10"
  echo "    (Note: Add module prefix like /pocket.proof.MsgCreateClaim or /pocket.supplier.MsgStakeSupplier)"
}

# ===============================================
# COMMON UTILITIES
# ===============================================

# Validate environment parameter
validate_env() {
  local env="$1"
  if [[ "$env" != "alpha" && "$env" != "beta" && "$env" != "main" ]]; then
    echo "Error: Invalid environment. Must be one of: alpha, beta, main"
    return 1
  fi
  return 0
}

# Validate block range
validate_block_range() {
  local start="$1"
  local end="$2"
  if [[ $start -gt $end ]]; then
    echo "Error: Start height ($start) cannot be greater than end height ($end)"
    return 1
  fi
  return 0
}

# Query and cache transactions for a given range
query_txs_range() {
  local start="$1"
  local end="$2"
  local env="$3"
  local output_file="$4"
  local additional_query="${5:-}"

  # Ensure we use absolute path if a relative path was provided
  if [[ "${output_file}" != /* ]]; then
    output_file="/tmp/${output_file}"
  fi

  if [[ -f "$output_file" ]]; then
    echo "Using existing cached data from $output_file"
    return 0
  fi

  local query="tx.height > $start AND tx.height < $end"
  if [[ -n "$additional_query" ]]; then
    query="$query AND $additional_query"
  fi

  echo "Querying transactions from height $start to $end on '$env' network..."
  pocketd query txs \
    --network="$env" --grpc-insecure=false \
    --query="$query" \
    --limit "10000000000" --page 1 -o json >"$output_file"
}

# Query single block with error handling
query_single_block() {
  local height="$1"
  local env="$2"
  local output_file="$3"

  if ! pocketd query block-results "$height" \
    --network="$env" --grpc-insecure=false \
    -o json >"$output_file" 2>/dev/null; then
    echo "Warning: Failed to query block $height, skipping..."
    return 1
  fi
  return 0
}

# Common JQ filter for supplier-related events
get_supplier_event_types_json() {
  cat <<'EOF'
[
    "pocket.proof.EventClaimCreated",
    "pocket.proof.EventClaimUpdated",
    "pocket.proof.EventProofSubmitted",
    "pocket.proof.EventProofUpdated",
    "pocket.proof.EventProofValidityChecked",
    "pocket.supplier.EventSupplierStaked",
    "pocket.supplier.EventSupplierUnbondingBegin",
    "pocket.supplier.EventSupplierUnbondingEnd",
    "pocket.supplier.EventSupplierUnbondingCanceled",
    "pocket.supplier.EventSupplierServiceConfigActivated",
    "pocket.tokenomics.EventClaimExpired",
    "pocket.tokenomics.EventClaimSettled",
    "pocket.tokenomics.EventApplicationOverserviced",
    "pocket.tokenomics.EventSupplierSlashed",
    "pocket.tokenomics.EventApplicationReimbursementRequest"
]
EOF
}

# Common JQ filter for application-related events
get_application_event_types_json() {
  cat <<'EOF'
[
    "pocket.application.EventApplicationStaked",
    "pocket.application.EventApplicationUnbondingBegin",
    "pocket.application.EventApplicationUnbondingEnd",
    "pocket.application.EventApplicationUnbondingCanceled",
    "pocket.proof.EventClaimCreated",
    "pocket.proof.EventClaimUpdated",
    "pocket.proof.EventProofSubmitted",
    "pocket.proof.EventProofUpdated",
    "pocket.proof.EventProofValidityChecked",
    "pocket.tokenomics.EventClaimExpired",
    "pocket.tokenomics.EventClaimSettled",
    "pocket.tokenomics.EventApplicationOverserviced",
    "pocket.tokenomics.EventApplicationReimbursementRequest",
    "pocket.morse.EventMorseApplicationClaimed"
]
EOF
}

# Common JQ filter for ignored event types
get_ignored_event_types_json() {
  cat <<'EOF'
["coin_spent", "coin_received", "transfer", "message", "tx", "commission", "rewards", "mint"]
EOF
}

# ===============================================
# GENERAL QUERY FUNCTIONS
# ===============================================

function shannon_query_unique_tx_msgs_and_events() {
  if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    cat <<'EOF'
shannon_query_unique_tx_msgs_and_events - Query unique message and event types

DESCRIPTION:
  Queries all transactions between two block heights and extracts:
  1. All unique message types that start with 'pocket'
  2. All unique event types that start with 'pocket'

USAGE:
  shannon_query_unique_tx_msgs_and_events <start_height> <end_height> <env>

ARGUMENTS:
  start_height    Minimum block height (exclusive)
  end_height      Maximum block height (exclusive)
  env             Network environment - must be one of: alpha, beta, main

EXAMPLES:
  shannon_query_unique_tx_msgs_and_events 13000 37364 beta
  shannon_query_unique_tx_msgs_and_events 115010 116550 main
EOF
    return 0
  fi

  if [[ $# -ne 3 ]]; then
    echo "Error: Invalid number of arguments. Use --help for more information."
    return 1
  fi

  local start="$1"
  local end="$2"
  local env="$3"

  validate_env "$env" || return 1

  local tmp_file="/tmp/shannon_txs_${start}_${end}_${env}.json"
  query_txs_range "$start" "$end" "$env" "$tmp_file"

  echo ""
  echo "## Unique message types starting with 'pocket':"
  jq -r '.txs[].tx.body.messages[]."@type"' "$tmp_file" |
    grep '^/pocket' | sort -u | sed 's/^/- /'

  echo ""
  echo "## Unique event types starting with 'pocket':"
  jq -r '.txs[].events[].type' "$tmp_file" |
    grep '^pocket' | sort -u | sed 's/^/- /'
}

function shannon_query_tx_messages() {
  if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    cat <<'EOF'
shannon_query_tx_messages - Query transactions by message type

DESCRIPTION:
  Queries transactions between two block heights on a given environment,
  filtering by message type (required) and optionally by sender address.
  Outputs height, txhash, message type, signer(s), and list of event types.

USAGE:
  shannon_query_tx_messages <start_height> <end_height> <message_type> <sender_or_empty> <env>

ARGUMENTS:
  start_height    Minimum block height (exclusive)
  end_height      Maximum block height (exclusive)
  message_type    Message type (e.g., /pocket.supplier.MsgUnstakeSupplier)
  sender_or_empty (Optional) Sender address (use "" if not filtering by sender)
  env             Network environment - must be one of: alpha, beta, main

EXAMPLES:
  shannon_query_tx_messages 13000 37364 /pocket.supplier.MsgUnstakeSupplier pokt1abc123... beta
  shannon_query_tx_messages 13000 37364 /pocket.supplier.MsgUnstakeSupplier "" beta
EOF
    return 0
  fi

  if [[ $# -ne 5 ]]; then
    echo "Error: Invalid number of arguments. Use --help for more information."
    return 1
  fi

  local start="$1"
  local end="$2"
  local msg_type="$3"
  local sender="$4"
  local env="$5"

  validate_env "$env" || return 1

  local msg_type_safe=$(echo "$msg_type" | sed 's/[^a-zA-Z0-9._-]/_/g')
  local sender_safe=$(echo "$sender" | sed 's/[^a-zA-Z0-9._-]/_/g' | cut -c1-20)
  local tmp_file="/tmp/shannon_tx_msgs_${start}_${end}_${msg_type_safe}_${sender_safe}_${env}.json"
  local additional_query="message.action='${msg_type}'"

  if [[ -n "$sender" ]]; then
    additional_query="${additional_query} AND message.sender='${sender}'"
  fi

  echo "Querying '$env' for message type '$msg_type' from height $start to $end..."
  [[ -n "$sender" ]] && echo "Sender filter: $sender"

  query_txs_range "$start" "$end" "$env" "$tmp_file" "$additional_query"

  echo ""
  echo "Results:"
  jq --arg MSG_TYPE "$msg_type" '
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
      ' "$tmp_file"
}

function shannon_query_tx_events() {
  if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    cat <<'EOF'
shannon_query_tx_events - Query transactions by event type

DESCRIPTION:
  Queries all transactions between a given block height range,
  filtering for a specific event type and returning matching events with decoded attributes,
  along with the transaction height and hash.

USAGE:
  shannon_query_tx_events <start_height> <end_height> <event_type> <env>

ARGUMENTS:
  start_height    Start block height (exclusive)
  end_height      End block height (exclusive)
  event_type      Event type (e.g., pocket.supplier.EventSupplierUnbondingBegin)
  env             Network environment - must be one of: alpha, beta, main

EXAMPLES:
  shannon_query_tx_events 13000 37364 pocket.supplier.EventSupplierUnbondingBegin beta
  shannon_query_tx_events 115010 116550 pocket.supplier.EventSupplierUnbondingBegin main
EOF
    return 0
  fi

  if [[ $# -ne 4 ]]; then
    echo "Error: Invalid number of arguments. Use --help for more information."
    return 1
  fi

  local start="$1"
  local end="$2"
  local event_type="$3"
  local env="$4"

  validate_env "$env" || return 1

  local event_type_safe=$(echo "$event_type" | sed 's/[^a-zA-Z0-9._-]/_/g')
  local tmp_file="/tmp/shannon_tx_events_${start}_${end}_${event_type_safe}_${env}.json"

  echo "Querying all transactions from $start to $end on '$env'..."
  echo "Filtering for event type: $event_type"
  echo "--------------------------------------"

  query_txs_range "$start" "$end" "$env" "$tmp_file"

  jq --arg EVENT_TYPE "$event_type" '
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
      ' "$tmp_file"
}

# ===============================================
# BLOCK QUERY FUNCTIONS
# ===============================================

function shannon_query_unique_block_events() {
  if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    cat <<'EOF'
shannon_query_unique_block_events - Query unique block events from block results

DESCRIPTION:
  Queries block results for a range of block heights and returns only unique event types
  and their attributes, removing duplicates and providing a summary of all distinct
  events found in the block range.

USAGE:
  shannon_query_unique_block_events <start_height> <end_height> <env> [event_prefix] [--include-ignored] [--show-count]

ARGUMENTS:
  start_height    Start block height (inclusive)
  end_height      End block height (inclusive)
  env             Network environment - must be one of: alpha, beta, main
  event_prefix    (Optional) Event type prefix filter (e.g., 'pocket' for pocket events)
  --include-ignored (Optional) Include default ignored event types
  --show-count    (Optional) Show count of each unique event type

IGNORED EVENT TYPES (by default):
  - coin_spent, coin_received, transfer, message, tx
  - commission, rewards, mint

EXAMPLES:
  shannon_query_unique_block_events 115575 115580 main
  shannon_query_unique_block_events 115575 115580 main pocket
  shannon_query_unique_block_events 115575 115580 main "" --include-ignored --show-count
EOF
    return 0
  fi

  if [[ $# -lt 3 ]]; then
    echo "Error: Invalid number of arguments. Use --help for more information."
    return 1
  fi

  local start="$1"
  local end="$2"
  local env="$3"
  local event_prefix="${4:-}"
  local include_ignored=""
  local show_count=""

  # Check for flags in any position
  for arg in "$@"; do
    if [[ "$arg" == "--include-ignored" ]]; then
      include_ignored="true"
    elif [[ "$arg" == "--show-count" ]]; then
      show_count="true"
    fi
  done

  validate_env "$env" || return 1
  validate_block_range "$start" "$end" || return 1

  local event_prefix_safe=$(echo "$event_prefix" | sed 's/[^a-zA-Z0-9._-]/_/g')
  local tmp_file="/tmp/unique_block_events_${start}_${end}_${event_prefix_safe}_${env}.json"
  local raw_events_file="/tmp/raw_events_${start}_${end}_${event_prefix_safe}_${env}.json"
  local ignored_events=$(get_ignored_event_types_json)

  echo "Querying unique block events from $start to $end on '$env'..."
  [[ -n "$event_prefix" ]] && echo "Filtering for events starting with: $event_prefix"
  [[ -n "$include_ignored" ]] && echo "Including ignored event types" || echo "Ignoring common event types"
  echo "--------------------------------------"

  # Initialize empty JSON array for raw events
  local all_events_tmp="/tmp/shannon_block_events_tmp_${start}_${end}_${event_prefix_safe}_${env}.json"
  echo "[]" >"$all_events_tmp"

  # Loop through each block height and collect all events
  for ((height = $start; height <= $end; height++)); do
    mkdir -p "/tmp/blocks_${start}_${end}_${env}"
    local block_tmp="/tmp/blocks_${start}_${end}_${env}/block_$height.json"
    echo "Processing block $height..."

    if query_single_block "$height" "$env" "$block_tmp"; then
      jq --argjson height "$height" \
        --arg event_prefix "$event_prefix" \
        --argjson ignored_events "$ignored_events" \
        --arg include_ignored "$include_ignored" '
                def process_events(events; source):
                  events
                  | map(select(
                      (if $event_prefix != "" then (.type | startswith($event_prefix)) else true end)
                      and
                      (if $include_ignored == "true" then true else (.type as $t | $ignored_events | index($t) | not) end)
                    ))
                  | map({
                      height: ($height | tostring),
                      source: source,
                      event_type: .type,
                      attributes: (
                        .attributes
                        | map(select(.key != "msg_index"))
                        | map({ (.key): .value })
                        | add // {}
                        | with_entries(.value |= try fromjson catch .)
                      )
                    });

                [
                  (.txs_results // [] | to_entries | map(
                    .value.events as $events |
                    process_events($events; "transaction")
                  ) | flatten),
                  process_events(.finalize_block_events // []; "finalize_block")
                ] | flatten
              ' "$block_tmp" >"$block_tmp.events"

      # Merge events into the main array
      jq -s '.[0] + .[1]' "$all_events_tmp" "$block_tmp.events" >"$all_events_tmp.new"
      mv "$all_events_tmp.new" "$all_events_tmp"
      rm -f "$block_tmp.events"
    fi
    rm -f "$block_tmp"
  done

  # Move the final events file
  mv "$all_events_tmp" "$raw_events_file"

  echo ""
  echo "Processing unique events..."

  jq --arg show_count "$show_count" '
      group_by(.event_type)
      | map({
          event_type: .[0].event_type,
          count: length,
          sources: (map(.source) | unique),
          sample_attributes: (.[0].attributes // {}),
          unique_attribute_keys: (map(.attributes // {} | keys) | flatten | unique | sort)
        })
      | sort_by(.event_type)
      | if $show_count == "true" then
          ("## Summary of Unique Event Types:\n" +
           (map("- " + .event_type + " (count: " + (.count | tostring) + ", sources: " + (.sources | join(", ")) + ")") | join("\n")) +
           "\n\n## Detailed Event Information:"),
          .[]
        else
          .[]
        end
    ' "$raw_events_file" >"$tmp_file"

  # Display results based on show_count flag
  if [[ "$show_count" == "true" ]]; then
    echo "## Summary of Unique Event Types:"
    jq -r '"- " + .event_type + " (count: " + (.count | tostring) + ", sources: " + (.sources | join(", ")) + ")"' "$tmp_file"
    echo ""
    echo "## Detailed Event Information:"
    jq -r '
          "### Event Type: " + .event_type,
          "**Count:** " + (.count | tostring),
          "**Sources:** " + (.sources | join(", ")),
          "**Unique Attribute Keys:** " + (.unique_attribute_keys | join(", ")),
          "**Sample Attributes:**",
          (.sample_attributes | to_entries | map("  - " + .key + ": " + (.value | tostring)) | join("\n")),
          ""
        ' "$tmp_file"
  else
    echo "## Unique Event Types Found:"
    jq -r '.event_type' "$tmp_file" | sort -u | sed 's/^/- /'
    echo ""
    echo "## Detailed Event Information:"
    jq -r '
          "### Event Type: " + .event_type,
          "**Sources:** " + (.sources | join(", ")),
          "**Unique Attribute Keys:** " + (.unique_attribute_keys | join(", ")),
          "**Sample Attributes:**",
          (.sample_attributes | to_entries | map("  - " + .key + ": " + (.value | tostring)) | join("\n")),
          ""
        ' "$tmp_file"
  fi

  echo "Query completed. Raw events saved to: $raw_events_file"
  rm -f "$tmp_file"
}

function shannon_query_block_events() {
  if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    cat <<'EOF'
shannon_query_block_events - Query block events from block results

DESCRIPTION:
  Queries block results for a range of block heights,
  extracting both transaction events and finalize_block_events, with optional filtering
  to ignore common/noisy event types.

USAGE:
  shannon_query_block_events <start_height> <end_height> <env> [event_type] [--include-ignored]

ARGUMENTS:
  start_height    Start block height (inclusive)
  end_height      End block height (inclusive)
  env             Network environment - must be one of: alpha, beta, main
  event_type      (Optional) Event type filter - only returns events of this type
  --include-ignored (Optional) Include default ignored event types

IGNORED EVENT TYPES (by default):
  - coin_spent, coin_received, transfer, message, tx
  - commission, rewards, mint

EXAMPLES:
  shannon_query_block_events 115575 115580 main
  shannon_query_block_events 115575 115580 main pocket.supplier.EventSupplierUnbondingBegin
  shannon_query_block_events 115575 115580 main "" --include-ignored
EOF
    return 0
  fi

  if [[ $# -lt 3 ]]; then
    echo "Error: Invalid number of arguments. Use --help for more information."
    return 1
  fi

  local start="$1"
  local end="$2"
  local env="$3"
  local event_type="${4:-}"
  local include_ignored=""

  # Check for --include-ignored flag
  if [[ "$4" == "--include-ignored" || "$5" == "--include-ignored" ]]; then
    include_ignored="true"
  fi

  validate_env "$env" || return 1
  validate_block_range "$start" "$end" || return 1

  local ignored_events=$(get_ignored_event_types_json)

  echo "Querying block events from $start to $end on '$env'..."
  [[ -n "$event_type" ]] && echo "Filtering for event type: $event_type"
  [[ -n "$include_ignored" ]] && echo "Including ignored event types" || echo "Ignoring common event types"
  echo "--------------------------------------"

  # Loop through each block height
  for ((height = $start; height <= $end; height++)); do
    echo "Processing block $height..."

    local event_type_safe=$(echo "$event_type" | sed 's/[^a-zA-Z0-9._-]/_/g')
    local block_file="/tmp/shannon_block_events_${height}_${event_type_safe}_${env}.json"
    if query_single_block "$height" "$env" "$block_file"; then
      jq --argjson height "$height" \
        --arg event_type "$event_type" \
        --argjson ignored_events "$ignored_events" \
        --arg include_ignored "$include_ignored" '
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

                def process_events(events; txhash):
                  events
                  | map(select(
                      (if $event_type != "" then .type == $event_type else true end)
                      and
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

                (.txs_results // [] | to_entries | map(
                  .value.events as $events |
                  .key as $tx_index |
                  ("tx_" + ($tx_index | tostring)) as $txhash |
                  process_events($events; $txhash)
                ) | flatten),
                process_events(.finalize_block_events // []; null)
                | .[]
              ' "$block_file"
    fi
  done

  echo "Query completed."
}

# ===============================================
# SUPPLIER-SPECIFIC FUNCTIONS
# ===============================================

function shannon_query_unique_claim_suppliers() {
  if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    cat <<'EOF'
shannon_query_unique_claim_suppliers - Query unique claim supplier addresses

DESCRIPTION:
  Queries all MsgCreateClaim transactions within a given block height range,
  returning a unique list of supplier_operator_address values.

USAGE:
  shannon_query_unique_claim_suppliers <min_height> <max_height> <env>

ARGUMENTS:
  min_height      Minimum block height (exclusive)
  max_height      Maximum block height (inclusive)
  env             Network environment - must be one of: alpha, beta, main

EXAMPLES:
  shannon_query_unique_claim_suppliers 13000 37364 beta
  shannon_query_unique_claim_suppliers 115010 116550 main
EOF
    return 0
  fi

  if [[ $# -ne 3 ]]; then
    echo "Error: Invalid number of arguments. Use --help for more information."
    return 1
  fi

  local min_height="$1"
  local max_height="$2"
  local env="$3"

  validate_env "$env" || return 1

  local additional_query="message.action='/pocket.proof.MsgCreateClaim'"

  echo "Querying MsgCreateClaim transactions from $env between heights ($min_height, $max_height]..."

  local additional_query_safe=$(echo "$additional_query" | sed 's/[^a-zA-Z0-9._-]/_/g' | cut -c1-50)
  local tmp_file="/tmp/shannon_txs_claim_suppliers_${min_height}_${max_height}_${env}_${additional_query_safe}.json"
  
  query_txs_range "$min_height" "$max_height" "$env" "$tmp_file" "$additional_query"

  jq '[.txs[].tx.body.messages[]
        | select(."@type" == "/pocket.proof.MsgCreateClaim" and .supplier_operator_address != null)
        | .supplier_operator_address]
        | unique' "$tmp_file"
}

function shannon_query_supplier_tx_events() {
  if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    cat <<'EOF'
shannon_query_supplier_tx_events - Query supplier-related transaction events

DESCRIPTION:
  Queries transactions between two block heights, filtering for all supplier-related
  events and matching by supplier operator address. Returns events with decoded
  attributes where the supplier operator address matches the provided address.

USAGE:
  shannon_query_supplier_tx_events <start_height> <end_height> <supplier_address> <env>

ARGUMENTS:
  start_height      Minimum block height (exclusive)
  end_height        Maximum block height (exclusive)
  supplier_address  Supplier operator address to filter by
  env               Network environment - must be one of: alpha, beta, main

EXAMPLES:
  shannon_query_supplier_tx_events 13000 37364 pokt1hcfx7lx92p03r5gwjt7t7jk0j667h7rcvart9f beta
  shannon_query_supplier_tx_events 115010 116550 pokt1hcfx7lx92p03r5gwjt7t7jk0j667h7rcvart9f main

FILTERED EVENT TYPES:
  - All pocket.proof.* events
  - All pocket.supplier.* events
  - All pocket.tokenomics.* supplier-related events
EOF
    return 0
  fi

  if [[ $# -ne 4 ]]; then
    echo "Error: Invalid number of arguments. Use --help for more information."
    return 1
  fi

  local start="$1"
  local end="$2"
  local supplier_addr="$3"
  local env="$4"

  validate_env "$env" || return 1

  local event_types_json=$(get_supplier_event_types_json)
  local additional_query="message.sender='${supplier_addr}'"

  echo "Querying supplier events from $start to $end on '$env'..."
  echo "Supplier operator address: $supplier_addr"
  echo "--------------------------------------"

  local additional_query_safe=$(echo "$additional_query" | sed 's/[^a-zA-Z0-9._-]/_/g' | cut -c1-50)
  local tmp_file="/tmp/shannon_txs_supplier_tx_events_${start}_${end}_${env}_${additional_query_safe}.json"
  
  query_txs_range "$start" "$end" "$env" "$tmp_file" "$additional_query"

  jq --argjson EVENT_TYPES "$event_types_json" --arg SUPPLIER_ADDR "$supplier_addr" '
      .txs[]
      | {
          height,
          txhash,
          events
        }
      | . as $tx
      | {
          supplier_events: (
            $tx.events
            | map(select(.type as $type | $EVENT_TYPES | index($type)))
            | map(
                {
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
                }
                | . as $event
                | select(
                    (.attributes.supplier_operator_address == $SUPPLIER_ADDR) or
                    (.attributes.supplier_operator_addr == $SUPPLIER_ADDR) or
                    (.attributes.claim.supplier_operator_address == $SUPPLIER_ADDR) or
                    (.attributes.proof.supplier_operator_address == $SUPPLIER_ADDR) or
                    (.attributes.supplier.operator_address == $SUPPLIER_ADDR)
                  )
                | {
                    height,
                    txhash,
                    event_type,
                    supplier_operator_address: (
                      .attributes.supplier_operator_address //
                      .attributes.supplier_operator_addr //
                      .attributes.claim.supplier_operator_address //
                      .attributes.proof.supplier_operator_address //
                      .attributes.supplier.operator_address
                    ),
                    application_address: (
                      .attributes.claim.session_header.application_address //
                      .attributes.proof.session_header.application_address //
                      .attributes.application_addr //
                      .attributes.session_header.application_address
                    ),
                    service_id: (
                      .attributes.claim.session_header.service_id //
                      .attributes.proof.session_header.service_id //
                      .attributes.service_id //
                      .attributes.session_header.service_id
                    ),
                    num_relays: .attributes.num_relays,
                    num_claimed_compute_units: .attributes.num_claimed_compute_units,
                    num_estimated_compute_units: .attributes.num_estimated_compute_units,
                    claimed_upokt_amount: .attributes.claimed_upokt.amount
                  }
              )
          )
        }
      | .supplier_events[]
    ' "$tmp_file"
}

function shannon_query_supplier_block_events() {
  if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    cat <<'EOF'
shannon_query_supplier_block_events - Query supplier-related block events

DESCRIPTION:
  Queries block results for a range of block heights, extracting both transaction
  events and finalize_block_events, filtering for supplier-related events and
  matching by supplier operator address.

USAGE:
  shannon_query_supplier_block_events <start_height> <end_height> <supplier_address> <env>

ARGUMENTS:
  start_height      Start block height (inclusive)
  end_height        End block height (inclusive)
  supplier_address  Supplier operator address to filter by
  env               Network environment - must be one of: alpha, beta, main

EXAMPLES:
  shannon_query_supplier_block_events 115575 115580 pokt1hcfx7lx92p03r5gwjt7t7jk0j667h7rcvart9f main

FILTERED EVENT TYPES:
  - All pocket.proof.* events
  - All pocket.supplier.* events
  - All pocket.tokenomics.* supplier-related events
EOF
    return 0
  fi

  if [[ $# -ne 4 ]]; then
    echo "Error: Invalid number of arguments. Use --help for more information."
    return 1
  fi

  local start="$1"
  local end="$2"
  local supplier_addr="$3"
  local env="$4"

  validate_env "$env" || return 1
  validate_block_range "$start" "$end" || return 1

  local event_types_json=$(get_supplier_event_types_json)

  echo "Querying supplier block events from $start to $end on '$env'..."
  echo "Supplier operator address: $supplier_addr"
  echo "--------------------------------------"

  # Loop through each block height
  for ((height = $start; height <= $end; height++)); do
    echo "Processing block $height..."

    local supplier_safe=$(echo "$supplier_addr" | sed 's/[^a-zA-Z0-9._-]/_/g' | cut -c1-20)
    local block_file="/tmp/shannon_supplier_block_events_${height}_${supplier_safe}_${env}.json"
    if query_single_block "$height" "$env" "$block_file"; then
      jq --argjson height "$height" \
        --argjson event_types "$event_types_json" \
        --arg supplier_addr "$supplier_addr" '
              def process_supplier_events(events; txhash):
                events
                | map(select(.type as $type | $event_types | index($type)))
                | map(
                    {
                      height: ($height | tostring),
                      txhash: txhash,
                      event_type: .type,
                      attributes: (
                        .attributes
                        | map(select(.key != "msg_index"))
                        | map({ (.key): .value })
                        | add
                        | with_entries(.value |= try fromjson catch .)
                      )
                    }
                    | . as $event
                    | select(
                        (.attributes.supplier_operator_address == $supplier_addr) or
                        (.attributes.supplier_operator_addr == $supplier_addr) or
                        (.attributes.claim.supplier_operator_address == $supplier_addr) or
                        (.attributes.proof.supplier_operator_address == $supplier_addr) or
                        (.attributes.supplier.operator_address == $supplier_addr)
                      )
                    | {
                        height,
                        txhash,
                        event_type,
                        supplier_operator_address: (
                          .attributes.supplier_operator_address //
                          .attributes.supplier_operator_addr //
                          .attributes.claim.supplier_operator_address //
                          .attributes.proof.supplier_operator_address //
                          .attributes.supplier.operator_address
                        ),
                        application_address: (
                          .attributes.claim.session_header.application_address //
                          .attributes.proof.session_header.application_address //
                          .attributes.application_addr //
                          .attributes.session_header.application_address
                        ),
                        service_id: (
                          .attributes.claim.session_header.service_id //
                          .attributes.proof.session_header.service_id //
                          .attributes.service_id //
                          .attributes.session_header.service_id
                        ),
                        num_relays: .attributes.num_relays,
                        num_claimed_compute_units: .attributes.num_claimed_compute_units,
                        num_estimated_compute_units: .attributes.num_estimated_compute_units,
                        claimed_upokt_amount: .attributes.claimed_upokt.amount
                      }
                  );

              ((.txs_results // [] | to_entries | map(
                .value.events as $events |
                .key as $tx_index |
                ("tx_" + ($tx_index | tostring)) as $txhash |
                process_supplier_events($events; $txhash)
              ) | flatten),
              process_supplier_events(.finalize_block_events // []; null))
              | .[]
            ' "$block_file"
    fi
  done

  echo "Query completed."
}

function shannon_query_application_block_events() {
  if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    cat <<'EOF'
shannon_query_application_block_events - Query application-related block events

DESCRIPTION:
  Queries block results for a range of block heights, extracting both transaction
  events and finalize_block_events, filtering for application-related events and
  matching by application address.

USAGE:
  shannon_query_application_block_events <start_height> <end_height> <application_address> <env>

ARGUMENTS:
  start_height         Start block height (inclusive)
  end_height           End block height (inclusive)
  application_address  Application address to filter by
  env                  Network environment - must be one of: alpha, beta, main

EXAMPLES:
  shannon_query_application_block_events 115575 115580 pokt14tg8v3hns5tjefnmqs9u98jqjp6mw6wmwwmuh2 main

FILTERED EVENT TYPES:
  - All pocket.application.* events
  - All pocket.proof.* events (where application is involved)
  - All pocket.tokenomics.* application-related events
  - All pocket.morse.* application events
EOF
    return 0
  fi

  if [[ $# -ne 4 ]]; then
    echo "Error: Invalid number of arguments. Use --help for more information."
    return 1
  fi

  local start="$1"
  local end="$2"
  local application_addr="$3"
  local env="$4"

  validate_env "$env" || return 1
  validate_block_range "$start" "$end" || return 1

  local event_types_json=$(get_application_event_types_json)

  echo "Querying application block events from $start to $end on '$env'..."
  echo "Application address: $application_addr"
  echo "--------------------------------------"

  # Loop through each block height
  for ((height = $start; height <= $end; height++)); do
    echo "Processing block $height..."

    local application_safe=$(echo "$application_addr" | sed 's/[^a-zA-Z0-9._-]/_/g' | cut -c1-20)
    local block_file="/tmp/shannon_application_block_events_${height}_${application_safe}_${env}.json"
    if query_single_block "$height" "$env" "$block_file"; then
      jq --argjson height "$height" \
        --argjson event_types "$event_types_json" \
        --arg application_addr "$application_addr" '
              def process_application_events(events; txhash):
                events
                | map(select(.type as $type | $event_types | index($type)))
                | map(
                    {
                      height: ($height | tostring),
                      txhash: txhash,
                      event_type: .type,
                      attributes: (
                        .attributes
                        | map(select(.key != "msg_index"))
                        | map({ (.key): .value })
                        | add
                        | with_entries(.value |= try fromjson catch .)
                      )
                    }
                    | . as $event
                    | select(
                        (.attributes.application_address == $application_addr) or
                        (.attributes.application_addr == $application_addr) or
                        (.attributes.claim.session_header.application_address == $application_addr) or
                        (.attributes.proof.session_header.application_address == $application_addr) or
                        (.attributes.session_header.application_address == $application_addr) or
                        (.attributes.application.address == $application_addr)
                      )
                    | {
                        height,
                        txhash,
                        event_type,
                        application_address: (
                          .attributes.application_address //
                          .attributes.application_addr //
                          .attributes.claim.session_header.application_address //
                          .attributes.proof.session_header.application_address //
                          .attributes.session_header.application_address //
                          .attributes.application.address
                        ),
                        supplier_operator_address: (
                          .attributes.claim.supplier_operator_address //
                          .attributes.proof.supplier_operator_address //
                          .attributes.supplier_operator_address //
                          .attributes.supplier_operator_addr
                        ),
                        service_id: (
                          .attributes.claim.session_header.service_id //
                          .attributes.proof.session_header.service_id //
                          .attributes.service_id //
                          .attributes.session_header.service_id
                        ),
                        num_relays: .attributes.num_relays,
                        num_claimed_compute_units: .attributes.num_claimed_compute_units,
                        num_estimated_compute_units: .attributes.num_estimated_compute_units,
                        claimed_upokt_amount: .attributes.claimed_upokt.amount,
                        stake_amount: .attributes.stake.amount,
                        unbonding_height: .attributes.unbonding_height
                      }
                  );

              ((.txs_results // [] | to_entries | map(
                .value.events as $events |
                .key as $tx_index |
                ("tx_" + ($tx_index | tostring)) as $txhash |
                process_application_events($events; $txhash)
              ) | flatten),
              process_application_events(.finalize_block_events // []; null))
              | .[]
            ' "$block_file"
    fi
  done

  echo "Query completed."
}

# ===============================================
# INITIALIZATION
# ===============================================

# Show help by default when script is sourced
help
