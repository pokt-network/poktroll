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


# Placeholder function for querying network channels (replace with actual query logic)
query_network_channels() {
    chain_id="$1"

    case "$chain_id" in
        "pocket")
            pocketd q ibc channel channels --node=${POCKET_NODE} --network=${NETWORK} -o json 2>/dev/null #--home=${POCKETD_HOME}
            ;;
        "agoriclocal")
            kubectl_exec_grep_pod "agoric-validator" agd q ibc channel channels -o json 2>/dev/null
            ;;
        "axelar")
            kubectl_exec_grep_pod "axelar-validator" axelard q ibc channel channels -o json 2>/dev/null
            ;;
        "osmosis")
            kubectl_exec_grep_pod "osmosis-validator" osmosisd q ibc channel channels -o json 2>/dev/null
            ;;
          *)
            echo "Unsupported chain ID: $chain_id" >&2
            return 1
            ;;
    esac
}

# Analogous to NAT traversal: look up the source channel for the counterparty chain with the given chain ID and destination channel ID.
# Arguments:
# - counterparty_chain_id: The chain ID of the counterparty chain
# - dest_channel_id: The destination channel ID; counterparty channel ID to match
# - dest_port_id: The port ID of the destination port; counterparty port ID to match
ibc_get_counterparty_channel_id() {
    local counterparty_chain_id="$1"
    local dest_channel_id="$2"
    local dest_port_id="$3"
    #echo "Looking up source channel ID for network: $counterparty_chain_id dest channel: $dest_channel_id..."

    # Query channels for the network (replace this with the actual query logic)
    channels_data=$(query_network_channels "$counterparty_chain_id")
#    echo "$channels_data" | jq -c '.channels[] | select(.state == "STATE_OPEN")'
#    return

    if [[ $? -ne 0 ]]; then
        echo "Error querying network $counterparty_chain_id, skipping." >&2
        exit 1
    fi

    # Parse the channels and populate the path-to-channel mapping
    # We're interested only in channels with state "STATE_OPEN"
    for channel in $(echo "$channels_data" | jq -c '.channels[] | select(.state == "STATE_OPEN")'); do
        counterparty_channel_id=$(echo "$channel" | jq -r '.counterparty.channel_id')
        counterparty_port_id=$(echo "$channel" | jq -r '.counterparty.port_id')

        # Check if both the destination channel ID and port match.
        if [ "$counterparty_channel_id" = "$dest_channel_id" ] && [ "$counterparty_port_id" = "$dest_port_id" ]; then
            echo $(echo "$channel" | jq -r '.channel_id')
            return
        fi
    done
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
        case "$source_chain_id" in
            "agoriclocal")
                connection_info=$(kubectl_exec_grep_pod "agoric-validator" agd q ibc connection connection "$connection_id" -o json 2>/dev/null)
                ;;
            "osmosis")
                connection_info=$(kubectl_exec_grep_pod "osmosis-validator" osmosisd q ibc connection connection "$connection_id" -o json 2>/dev/null)
                ;;
            "pocket")
                connection_info=$(pocketd q ibc connection connection "$connection_id" --node=${POCKET_NODE} --network=${NETWORK} -o json 2>/dev/null)
                ;;
            *)
                continue
                ;;
        esac
        
        if [[ $? -eq 0 ]] && [[ -n "$connection_info" ]]; then
            # Extract client ID and query client to determine target chain
            client_id=$(echo "$connection_info" | jq -r '.connection.client_id')
            
            case "$source_chain_id" in
                "agoriclocal")
                    client_info=$(kubectl_exec_grep_pod "agoric-validator" agd q ibc client state "$client_id" -o json 2>/dev/null)
                    ;;
                "osmosis")
                    client_info=$(kubectl_exec_grep_pod "osmosis-validator" osmosisd q ibc client state "$client_id" -o json 2>/dev/null)
                    ;;
                "pocket")
                    client_info=$(pocketd q ibc client state "$client_id" --node=${POCKET_NODE} --network=${NETWORK} -o json 2>/dev/null)
                    ;;
            esac
            
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
# TODO_IN_THIS_COMMIT: comment and/or update - assumes star topology and only transfer port (no ICA)
export AGORIC_POCKET_SRC_CHANNEL_ID="channel-0"
export AXELAR_POCKET_SRC_CHANNEL_ID="channel-0"
export OSMOSIS_POCKET_SRC_CHANNEL_ID="channel-0"
export POCKET_AGORIC_SRC_CHANNEL_ID=$(ibc_get_counterparty_channel_id "agoriclocal" "channel-0" "transfer")
export POCKET_AXELAR_SRC_CHANNEL_ID=$(ibc_get_counterparty_channel_id "axelar" "channel-0" "transfer")
export POCKET_OSMOSIS_SRC_CHANNEL_ID=$(ibc_get_counterparty_channel_id "osmosis" "channel-0" "transfer")

# Direct connections between Agoric and Osmosis (for PFM) - Dynamically discovered with fallback
# Try dynamic discovery first, fallback to known working values
AGORIC_OSMOSIS_DYNAMIC=$(ibc_find_channel_to_target "agoriclocal" "osmosis" "transfer" 2>/dev/null || echo "")
OSMOSIS_AGORIC_DYNAMIC=$(ibc_find_channel_to_target "osmosis" "agoriclocal" "transfer" 2>/dev/null || echo "")

export AGORIC_OSMOSIS_SRC_CHANNEL_ID="${AGORIC_OSMOSIS_DYNAMIC:-channel-0}"
export OSMOSIS_AGORIC_SRC_CHANNEL_ID="${OSMOSIS_AGORIC_DYNAMIC:-channel-1}"

# Debug output (uncomment for troubleshooting)
if [[ "${IBC_DEBUG:-}" == "true" ]]; then
    echo "DEBUG: AGORIC_OSMOSIS_SRC_CHANNEL_ID=$AGORIC_OSMOSIS_SRC_CHANNEL_ID (dynamic: '$AGORIC_OSMOSIS_DYNAMIC')" >&2
    echo "DEBUG: OSMOSIS_AGORIC_SRC_CHANNEL_ID=$OSMOSIS_AGORIC_SRC_CHANNEL_ID (dynamic: '$OSMOSIS_AGORIC_DYNAMIC')" >&2
fi

#echo "pocket_agoric_src_channel_id: $POCKET_AGORIC_SRC_CHANNEL_ID"
#echo "pocket_axelar_src_channel_id: $POCKET_AXELAR_SRC_CHANNEL_ID"
#echo "pocket_osmosis_src_channel_id: $POCKET_OSMOSIS_SRC_CHANNEL_ID"
