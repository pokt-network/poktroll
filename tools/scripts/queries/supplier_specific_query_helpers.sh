# -----------------------------------------------
# shannon_query_unique_claim_suppliers_addresses
# -----------------------------------------------
# Description:
#   Queries all MsgCreateClaim transactions from the Pocket Network Shannon testnet
#   within a given block height range, returning a unique list of supplier_operator_address values.
#
# Arguments:
#   $1 - Minimum block height (exclusive)
#   $2 - Maximum block height (inclusive)
#   $3 - Network environment - must be one of: alpha, beta, main
#
# Example:
#   shannon_query_unique_claim_suppliers_addresses 13000 37364 beta
#   shannon_query_unique_claim_suppliers_addresses 115010 116550 main
#
function shannon_query_unique_claim_suppliers_addresses() {
    if [[ "$1" == "--help" || "$1" == "-h" ]]; then
        echo "shannon_query_unique_claim_suppliers_addresses - Query unique claim supplier addresses"
        echo ""
        echo "DESCRIPTION:"
        echo "  Queries all MsgCreateClaim transactions from the Pocket Network Shannon testnet"
        echo "  within a given block height range, returning a unique list of supplier_operator_address values."
        echo ""
        echo "USAGE:"
        echo "  shannon_query_unique_claim_suppliers_addresses <min_height> <max_height> <env>"
        echo ""
        echo "ARGUMENTS:"
        echo "  min_height      Minimum block height (exclusive)"
        echo "  max_height      Maximum block height (inclusive)"
        echo "  env             Network environment - must be one of: alpha, beta, main"
        echo ""
        echo "EXAMPLES:"
        echo "  shannon_query_unique_claim_suppliers_addresses 13000 37364 beta"
        echo "  shannon_query_unique_claim_suppliers_addresses 115010 116550 main"
        return 0
    fi

    if [[ $# -ne 3 ]]; then
        echo "Error: Invalid number of arguments."
        echo ""
        echo "Usage:"
        echo "  shannon_query_unique_claim_suppliers_addresses <min_height> <max_height> <env (alpha|beta|main)>"
        echo ""
        echo "Example:"
        echo "  shannon_query_unique_claim_suppliers_addresses 13000 37364 beta"
        echo "  shannon_query_unique_claim_suppliers_addresses 115010 116550 main"
        echo ""
        echo "Use --help for more information."
        return 1
    fi

    local MIN_HEIGHT="$1"
    local MAX_HEIGHT="$2"
    local ENV="$3"

    if [[ "$ENV" != "alpha" && "$ENV" != "beta" && "$ENV" != "main" ]]; then
        echo "Error: Invalid environment. Must be one of: alpha, beta, main"
        return 1
    fi

    local TMP_FILE="/tmp/claims.json"

    echo "Querying MsgCreateClaim transactions from $ENV between heights ($MIN_HEIGHT, $MAX_HEIGHT]..."

    pocketd query txs \
        --network="$ENV" --grpc-insecure=false \
        --query="tx.height>${MIN_HEIGHT} AND tx.height<=${MAX_HEIGHT} AND message.action='/pocket.proof.MsgCreateClaim'" \
        --limit 10000000000 --page 1 -o json >"$TMP_FILE"

    jq '[.txs[].tx.body.messages[]
        | select(."@type" == "/pocket.proof.MsgCreateClaim" and .supplier_operator_address != null)
        | .supplier_operator_address]
        | unique' "$TMP_FILE"
}

# -----------------------------------------------
# shannon_query_supplier_tx_events
# -----------------------------------------------
# Description:
#   Queries transactions between two block heights on a given environment,
#   filtering for all supplier-related events and matching by supplier operator address.
#   Returns events with decoded attributes where the supplier operator address matches
#   the provided address in either message.sender or event attribute fields.
#
# Arguments:
#   $1 - Minimum block height (exclusive)
#   $2 - Maximum block height (exclusive)
#   $3 - Supplier operator address
#   $4 - Network environment - must be one of: alpha, beta, main
#
# Example:
#   shannon_query_supplier_tx_events 13000 37364 pokt1abc123... beta
#   shannon_query_supplier_tx_events 115010 116550 pokt1abc123... main
#
function shannon_query_supplier_tx_events() {
    if [[ "$1" == "--help" || "$1" == "-h" ]]; then
        echo "shannon_query_supplier_tx_events - Query supplier-related transaction events"
        echo ""
        echo "DESCRIPTION:"
        echo "  Queries transactions between two block heights on a given environment,"
        echo "  filtering for all supplier-related events and matching by supplier operator address."
        echo "  Returns events with decoded attributes where the supplier operator address matches"
        echo "  the provided address in either message.sender or event attribute fields."
        echo ""
        echo "USAGE:"
        echo "  shannon_query_supplier_tx_events <start_height> <end_height> <supplier_address> <env>"
        echo ""
        echo "ARGUMENTS:"
        echo "  start_height      Minimum block height (exclusive)"
        echo "  end_height        Maximum block height (exclusive)"
        echo "  supplier_address  Supplier operator address to filter by"
        echo "  env               Network environment - must be one of: alpha, beta, main"
        echo ""
        echo "EXAMPLES:"
        echo "  shannon_query_supplier_tx_events 13000 37364 pokt1abc123... beta"
        echo "  shannon_query_supplier_tx_events 115010 116550 pokt1abc123... main"
        echo ""
        echo "FILTERED EVENT TYPES:"
        echo "  - pocket.proof.EventClaimCreated"
        echo "  - pocket.proof.EventClaimUpdated"
        echo "  - pocket.proof.EventProofSubmitted"
        echo "  - pocket.proof.EventProofUpdated"
        echo "  - pocket.proof.EventProofValidityChecked"
        echo "  - pocket.supplier.EventSupplierStaked"
        echo "  - pocket.supplier.EventSupplierUnbondingBegin"
        echo "  - pocket.supplier.EventSupplierUnbondingEnd"
        echo "  - pocket.supplier.EventSupplierUnbondingCanceled"
        echo "  - pocket.supplier.EventSupplierServiceConfigActivated"
        echo "  - pocket.tokenomics.EventClaimExpired"
        echo "  - pocket.tokenomics.EventClaimSettled"
        echo "  - pocket.tokenomics.EventApplicationOverserviced"
        echo "  - pocket.tokenomics.EventSupplierSlashed"
        echo "  - pocket.tokenomics.EventApplicationReimbursementRequest"
        return 0
    fi

    if [[ $# -ne 4 ]]; then
        echo "Error: Invalid number of arguments."
        echo ""
        echo "Usage:"
        echo "  shannon_query_supplier_tx_events <start_height> <end_height> <supplier_address> <env (alpha|beta|main)>"
        echo ""
        echo "Example:"
        echo "  shannon_query_supplier_tx_events 13000 37364 pokt1abc123... beta"
        echo "  shannon_query_supplier_tx_events 115010 116550 pokt1abc123... main"
        echo ""
        echo "Use --help for more information."
        return 1
    fi

    local START="$1"
    local END="$2"
    local SUPPLIER_ADDR="$3"
    local ENV="$4"

    if [[ "$ENV" != "alpha" && "$ENV" != "beta" && "$ENV" != "main" ]]; then
        echo "Error: Invalid environment. Must be one of: alpha, beta, main"
        return 1
    fi

    local TMP_FILE="/tmp/supplier_events_filtered.json"

    echo "Querying supplier events from $START to $END on '$ENV'..."
    echo "Supplier operator address: $SUPPLIER_ADDR"
    echo "--------------------------------------"

    # Query transactions with message sender filter
    pocketd query txs \
        --network="$ENV" --grpc-insecure=false \
        --query="message.sender='${SUPPLIER_ADDR}' AND tx.height > ${START} AND tx.height < ${END}" \
        --limit "10000000000" --page 1 -o json >"$TMP_FILE"

    # Define the supplier-related event types
    local EVENT_TYPES=(
        "pocket.proof.EventClaimCreated"
        "pocket.proof.EventClaimUpdated"
        "pocket.proof.EventProofSubmitted"
        "pocket.proof.EventProofUpdated"
        "pocket.proof.EventProofValidityChecked"
        "pocket.supplier.EventSupplierStaked"
        "pocket.supplier.EventSupplierUnbondingBegin"
        "pocket.supplier.EventSupplierUnbondingEnd"
        "pocket.supplier.EventSupplierUnbondingCanceled"
        "pocket.supplier.EventSupplierServiceConfigActivated"
        "pocket.tokenomics.EventClaimExpired"
        "pocket.tokenomics.EventClaimSettled"
        "pocket.tokenomics.EventApplicationOverserviced"
        "pocket.tokenomics.EventSupplierSlashed"
        "pocket.tokenomics.EventApplicationReimbursementRequest"
    )

    # Convert array to JSON array for jq
    local EVENT_TYPES_JSON=$(printf '%s\n' "${EVENT_TYPES[@]}" | jq -R . | jq -s .)

    jq --argjson EVENT_TYPES "$EVENT_TYPES_JSON" --arg SUPPLIER_ADDR "$SUPPLIER_ADDR" '
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
                # Filter events that contain the supplier address in relevant fields
                | select(
                    (.attributes.supplier_operator_address == $SUPPLIER_ADDR) or
                    (.attributes.supplier_operator_addr == $SUPPLIER_ADDR) or
                    (.attributes.claim.supplier_operator_address == $SUPPLIER_ADDR) or
                    (.attributes.proof.supplier_operator_address == $SUPPLIER_ADDR) or
                    (.attributes.supplier.operator_address == $SUPPLIER_ADDR)
                  )
                # Extract only essential information
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
    ' "$TMP_FILE"
}

# -----------------------------------------------
# shannon_query_supplier_block_events
# -----------------------------------------------
# Description:
#   Queries block results for a range of block heights from the Pocket Network Shannon testnet,
#   extracting both transaction events and finalize_block_events, filtering for supplier-related
#   events and matching by supplier operator address. Returns only essential information.
#
# Arguments:
#   $1 - Start block height (inclusive)
#   $2 - End block height (inclusive)
#   $3 - Supplier operator address
#   $4 - Network environment - must be one of: alpha, beta, main
#
# Example:
#   shannon_query_supplier_block_events 115575 115580 pokt17jjp8pwjk8h7xyhvhqnt4cqxrfdmx3p37eva83 main
#
function shannon_query_supplier_block_events() {
    if [[ "$1" == "--help" || "$1" == "-h" ]]; then
        echo "shannon_query_supplier_block_events - Query supplier-related block events"
        echo ""
        echo "DESCRIPTION:"
        echo "  Queries block results for a range of block heights from the Pocket Network Shannon testnet,"
        echo "  extracting both transaction events and finalize_block_events, filtering for supplier-related"
        echo "  events and matching by supplier operator address. Returns only essential information."
        echo ""
        echo "USAGE:"
        echo "  shannon_query_supplier_block_events <start_height> <end_height> <supplier_address> <env>"
        echo ""
        echo "ARGUMENTS:"
        echo "  start_height      Start block height (inclusive)"
        echo "  end_height        End block height (inclusive)"
        echo "  supplier_address  Supplier operator address to filter by"
        echo "  env               Network environment - must be one of: alpha, beta, main"
        echo ""
        echo "EXAMPLES:"
        echo "  shannon_query_supplier_block_events 115575 115580 pokt17jjp8pwjk8h7xyhvhqnt4cqxrfdmx3p37eva83 main"
        echo ""
        echo "FILTERED EVENT TYPES:"
        echo "  - pocket.proof.EventClaimCreated"
        echo "  - pocket.proof.EventClaimUpdated"
        echo "  - pocket.proof.EventProofSubmitted"
        echo "  - pocket.proof.EventProofUpdated"
        echo "  - pocket.proof.EventProofValidityChecked"
        echo "  - pocket.supplier.EventSupplierStaked"
        echo "  - pocket.supplier.EventSupplierUnbondingBegin"
        echo "  - pocket.supplier.EventSupplierUnbondingEnd"
        echo "  - pocket.supplier.EventSupplierUnbondingCanceled"
        echo "  - pocket.supplier.EventSupplierServiceConfigActivated"
        echo "  - pocket.tokenomics.EventClaimExpired"
        echo "  - pocket.tokenomics.EventClaimSettled"
        echo "  - pocket.tokenomics.EventApplicationOverserviced"
        echo "  - pocket.tokenomics.EventSupplierSlashed"
        echo "  - pocket.tokenomics.EventApplicationReimbursementRequest"
        return 0
    fi

    if [[ $# -ne 4 ]]; then
        echo "Error: Invalid number of arguments."
        echo ""
        echo "Usage:"
        echo "  shannon_query_supplier_block_events <start_height> <end_height> <supplier_address> <env (alpha|beta|main)>"
        echo ""
        echo "Example:"
        echo "  shannon_query_supplier_block_events 115575 115580 pokt17jjp8pwjk8h7xyhvhqnt4cqxrfdmx3p37eva83 main"
        echo ""
        echo "Use --help for more information."
        return 1
    fi

    local START="$1"
    local END="$2"
    local SUPPLIER_ADDR="$3"
    local ENV="$4"

    if [[ "$ENV" != "alpha" && "$ENV" != "beta" && "$ENV" != "main" ]]; then
        echo "Error: Invalid environment. Must be one of: alpha, beta, main"
        return 1
    fi

    # Validate block range
    if [[ $START -gt $END ]]; then
        echo "Error: Start height ($START) cannot be greater than end height ($END)"
        return 1
    fi

    # Define the supplier-related event types
    local EVENT_TYPES=(
        "pocket.proof.EventClaimCreated"
        "pocket.proof.EventClaimUpdated"
        "pocket.proof.EventProofSubmitted"
        "pocket.proof.EventProofUpdated"
        "pocket.proof.EventProofValidityChecked"
        "pocket.supplier.EventSupplierStaked"
        "pocket.supplier.EventSupplierUnbondingBegin"
        "pocket.supplier.EventSupplierUnbondingEnd"
        "pocket.supplier.EventSupplierUnbondingCanceled"
        "pocket.supplier.EventSupplierServiceConfigActivated"
        "pocket.tokenomics.EventClaimExpired"
        "pocket.tokenomics.EventClaimSettled"
        "pocket.tokenomics.EventApplicationOverserviced"
        "pocket.tokenomics.EventSupplierSlashed"
        "pocket.tokenomics.EventApplicationReimbursementRequest"
    )

    # Convert array to JSON array for jq
    local EVENT_TYPES_JSON=$(printf '%s\n' "${EVENT_TYPES[@]}" | jq -R . | jq -s .)

    echo "Querying supplier block events from $START to $END on '$ENV'..."
    echo "Supplier operator address: $SUPPLIER_ADDR"
    echo "--------------------------------------"

    # Loop through each block height
    for ((height = $START; height <= $END; height++)); do
        echo "Processing block $height..."

        local BLOCK_TMP="/tmp/supplier_block_$height.json"

        # Query block results for specific height
        if ! pocketd query block-results "$height" \
            --network="$ENV" --grpc-insecure=false \
            -o json >"$BLOCK_TMP" 2>/dev/null; then
            echo "Warning: Failed to query block $height, skipping..."
            continue
        fi

        # Process the block data and filter for supplier events
        jq --argjson height "$height" \
            --argjson event_types "$EVENT_TYPES_JSON" \
            --arg supplier_addr "$SUPPLIER_ADDR" '
          # Function to filter and format supplier events
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
                # Filter events that contain the supplier address in relevant fields
                | select(
                    (.attributes.supplier_operator_address == $supplier_addr) or
                    (.attributes.supplier_operator_addr == $supplier_addr) or
                    (.attributes.claim.supplier_operator_address == $supplier_addr) or
                    (.attributes.proof.supplier_operator_address == $supplier_addr) or
                    (.attributes.supplier.operator_address == $supplier_addr)
                  )
                # Extract only essential information
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

          # Process transaction events (use tx index as identifier)
          ((.txs_results // [] | to_entries | map(
            .value.events as $events |
            .key as $tx_index |
            ("tx_" + ($tx_index | tostring)) as $txhash |
            process_supplier_events($events; $txhash)
          ) | flatten),

          # Process finalize block events (no txhash)
          process_supplier_events(.finalize_block_events // []; null))

          # Output each event individually
          | .[]
        ' "$BLOCK_TMP"

        # Clean up temporary block file
        rm -f "$BLOCK_TMP"
    done

    echo "Query completed."
}
