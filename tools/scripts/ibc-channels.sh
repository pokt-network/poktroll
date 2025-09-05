#!/bin/bash
set -eo pipefail

# Check if jq is installed, if not, provide installation instructions
if ! command -v jq &> /dev/null
then
    echo "jq is required but not installed. Please install it using:"
    echo "  Linux: sudo apt-get install jq"
    echo "  macOS: brew install jq"
    exit 1
fi

#POCKETD_HOME?=../../localnet/pocketd
POCKET_NODE=${POCKET_NODE:-tcp://127.0.0.1:26657} # The pocket node (validator in the localnet context)
NETWORK=${NETWORK:-local}

# Execute a command in a pod matching the given pod name regex.
kubectl_exec_grep_pod() {
  pod_name_regex="$1"
  shift
  kubectl exec $(kubectl get pods | grep $pod_name_regex | cut -f1 -d' ') -- "$@"
}

kubectl_exec_grep_pod_interactive() {
  pod_name_regex="$1"
  shift
  kubectl exec -it $(kubectl get pods | grep $pod_name_regex | cut -f1 -d' ') -- "$@"
}

# Get the binary name for a given chain ID
get_chain_binary() {
    local chain_id="$1"
    case "$chain_id" in
        "pocket") echo "pocketd" ;;
        "agoriclocal") echo "agd" ;;
        "axelar") echo "axelard" ;;
        "osmosis") echo "osmosisd" ;;
        *) echo "" ;;
    esac
}

# Get the pod name pattern for a given chain ID
get_chain_pod_pattern() {
    local chain_id="$1"
    case "$chain_id" in
        "pocket") echo "" ;;  # pocket runs locally, no pod
        "agoriclocal") echo "agoric-validator" ;;
        "axelar") echo "axelar-validator" ;;
        "osmosis") echo "osmosis-validator" ;;
        *) echo "" ;;
    esac
}

# Execute a command on a specific chain (either locally for pocket or in a pod for others)
ibc_exec_chain_command() {
    local chain_id="$1"
    shift
    local binary=$(get_chain_binary "$chain_id")
    local pod_pattern=$(get_chain_pod_pattern "$chain_id")

    if [ -z "$binary" ]; then
        echo "Error: Unsupported chain ID: $chain_id" >&2
        return 1
    fi

    if [ "$chain_id" = "pocket" ]; then
        # Execute locally for pocket
        "$binary" --node="${POCKET_NODE}" --network="${NETWORK}" "$@"
    else
        # Execute in pod for other chains
        if [ -z "$pod_pattern" ]; then
            echo "Error: No pod pattern for chain: $chain_id" >&2
            return 1
        fi
        kubectl_exec_grep_pod "$pod_pattern" "$binary" "$@"
    fi
}


# Query IBC channels for a given chain
query_network_channels() {
    local chain_id="$1"
    ibc_exec_chain_command "$chain_id" q ibc channel channels -o json
}

# Analogous to NAT traversal: look up the source channel for the counterparty chain with the given chain ID and destination channel ID.
# Arguments:
# - counterparty_chain_id: The chain ID of the counterparty chain
# - target_channel_id: The destination channel ID; counterparty channel ID to match
# - target_port_id: The port ID of the destination port; counterparty port ID to match
ibc_get_counterparty_channel_id() {
    local counterparty_chain_id="$1"
    local target_channel_id="$2"
    local target_port_id="$3"
    #echo "Looking up source channel ID for network: $counterparty_chain_id dest channel: $target_channel_id..."

    # Query channels for the network (replace this with the actual query logic)
    channels_data=$(query_network_channels "$counterparty_chain_id")

    if [[ $? -ne 0 ]]; then
        echo "Error querying network $counterparty_chain_id, skipping." >&2
        return 1
    fi

    # Parse the channels and populate the path-to-channel mapping
    # We're interested only in channels with state "STATE_OPEN"
    for channel in $(echo "$channels_data" | jq -c '.channels[] | select(.state == "STATE_OPEN")'); do
        counterparty_channel_id=$(echo "$channel" | jq -r '.counterparty.channel_id')
        counterparty_port_id=$(echo "$channel" | jq -r '.counterparty.port_id')
        source_channel_id=$(echo "$channel" | jq -r '.channel_id')

        # Check if both the destination channel ID and port match.
        if [ "$counterparty_channel_id" = "$target_channel_id" ] && [ "$counterparty_port_id" = "$target_port_id" ]; then
            echo "$source_channel_id"
            return 0
        fi
    done
    # If no channel found, return empty string
    return 1
}

# Enhanced function to find a channel connecting to a specific target chain
# Arguments:
# - source_chain_id: The chain to query channels from
# - target_chain_id: The chain we want to connect to
# - port_id: The port ID to use (default: "transfer")
ibc_find_channel_to_target() {
    local source_chain_id="$1"
    local target_chain_id="$2"
    local port_id="${3:-transfer}"

    # Query channels from the source chain
    channels_data=$(query_network_channels "$source_chain_id")

    if [[ $? -ne 0 ]]; then
        echo "Error querying network $source_chain_id, skipping." >&2
        return 1
    fi

    # Find channels that connect to the target chain
    for channel in $(echo "$channels_data" | jq -c '.channels[] | select(.state == "STATE_OPEN")'); do
        # Get the connection ID from this channel
        connection_id=$(echo "$channel" | jq -r '.connection_hops[0]')
        channel_port=$(echo "$channel" | jq -r '.port_id')
        # Skip if not the right port
        if [ "$channel_port" != "$port_id" ]; then
            continue
        fi

        # Query the connection to find what chain it connects to
        connection_info=$(ibc_exec_chain_command "$source_chain_id" q ibc connection end "$connection_id" -o json 2>/dev/null)

        if [[ $? -eq 0 ]] && [[ -n "$connection_info" ]]; then
            # Extract client ID and query client to determine target chain
            client_id=$(echo "$connection_info" | jq -r '.connection.client_id')

            client_info=$(ibc_exec_chain_command "$source_chain_id" q ibc client state "$client_id" -o json 2>/dev/null)

            if [[ $? -eq 0 ]] && [[ -n "$client_info" ]]; then
                # Extract chain ID from client state
                client_chain_id=$(echo "$client_info" | jq -r '.client_state.chain_id // .client_state.value.chain_id // empty')

                # If this channel connects to our target chain, return the channel ID
                if [ "$client_chain_id" = "$target_chain_id" ]; then
                    echo $(echo "$channel" | jq -r '.channel_id')
                    return 0
                fi
            fi
        fi
    done
    return 1
}

# Assign source channel ID variables for each path.
# Note: We query the source chain to find which channel connects to the target chain's channel-0
export AGORIC_POCKET_SRC_CHANNEL_ID="channel-0"
export AXELAR_POCKET_SRC_CHANNEL_ID="channel-0"
export OSMOSIS_POCKET_SRC_CHANNEL_ID="channel-0"

# Dynamically discover pocket channels by reverse lookup
# Find which pocket channel connects to each target chain
ibc_find_pocket_channel_to_chain() {
    local target_chain_id="$1"

    # Query pocket channels
    local channels_data=$(query_network_channels "pocket")
    if [[ $? -ne 0 ]]; then
        echo "Error querying pocket channels" >&2
        return 1
    fi

    # For each pocket channel, check what chain it connects to
    for channel in $(echo "$channels_data" | jq -c '.channels[] | select(.state == "STATE_OPEN")'); do
        local pocket_channel_id=$(echo "$channel" | jq -r '.channel_id')
        local connection_id=$(echo "$channel" | jq -r '.connection_hops[0]')
        local channel_port=$(echo "$channel" | jq -r '.port_id')

        # Skip if not transfer port
        if [ "$channel_port" != "transfer" ]; then
            continue
        fi

        # Query the connection to get client ID
        local connection_info=$(ibc_exec_chain_command "pocket" q ibc connection end "$connection_id" -o json 2>/dev/null)
        if [[ $? -eq 0 ]] && [[ -n "$connection_info" ]]; then
            local client_id=$(echo "$connection_info" | jq -r '.connection.client_id')

            # Query client to get target chain ID
            local client_info=$(ibc_exec_chain_command "pocket" q ibc client state "$client_id" -o json 2>/dev/null)
            if [[ $? -eq 0 ]] && [[ -n "$client_info" ]]; then
                local client_chain_id=$(echo "$client_info" | jq -r '.client_state.chain_id')

                # If this channel connects to our target chain, return the pocket channel ID
                if [ "$client_chain_id" = "$target_chain_id" ]; then
                    echo "$pocket_channel_id"
                    return 0
                fi
            fi
        fi
    done

    return 1
}

# Discover pocket channels dynamically
export POCKET_AGORIC_SRC_CHANNEL_ID=$(ibc_find_pocket_channel_to_chain "agoriclocal" 2>/dev/null || echo "")
export POCKET_AXELAR_SRC_CHANNEL_ID=$(ibc_find_pocket_channel_to_chain "axelar" 2>/dev/null || echo "")
export POCKET_OSMOSIS_SRC_CHANNEL_ID=$(ibc_find_pocket_channel_to_chain "osmosis" 2>/dev/null || echo "")
# Direct connections between Agoric and Osmosis (for PFM)
export AGORIC_OSMOSIS_SRC_CHANNEL_ID=$(ibc_find_channel_to_target "agoriclocal" "osmosis" "transfer" 2>/dev/null || echo "channel-1")
export OSMOSIS_AGORIC_SRC_CHANNEL_ID=$(ibc_find_channel_to_target "osmosis" "agoriclocal" "transfer" 2>/dev/null || echo "channel-1")

# Debug output for troubleshooting
if [[ "${IBC_DEBUG:-}" == "true" ]]; then
    echo "DEBUG: Pocket channel discovery results:" >&2
    echo "  POCKET_AGORIC_SRC_CHANNEL_ID='$POCKET_AGORIC_SRC_CHANNEL_ID'" >&2
    echo "  POCKET_AXELAR_SRC_CHANNEL_ID='$POCKET_AXELAR_SRC_CHANNEL_ID'" >&2
    echo "  POCKET_OSMOSIS_SRC_CHANNEL_ID='$POCKET_OSMOSIS_SRC_CHANNEL_ID'" >&2
    echo "  AGORIC_OSMOSIS_SRC_CHANNEL_ID='$AGORIC_OSMOSIS_SRC_CHANNEL_ID'" >&2
    echo "  OSMOSIS_AGORIC_SRC_CHANNEL_ID='$OSMOSIS_AGORIC_SRC_CHANNEL_ID'" >&2
fi


# Debug output (uncomment for troubleshooting)
if [[ "${IBC_DEBUG:-}" == "true" ]]; then
    echo "DEBUG: Direct channel discovery results:" >&2
fi
