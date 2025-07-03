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
            pocketd q ibc channel channels --node=${POCKET_NODE} --network=${NETWORK} -o json #--home=${POCKETD_HOME}
            ;;
        "agoriclocal")
            kubectl_exec_grep_pod "agoric-validator" agd q ibc channel channels -o json
            ;;
        "axelar")
            kubectl_exec_grep_pod "axelar-validator" axelard q ibc channel channels -o json
            ;;
        "osmosis")
            kubectl_exec_grep_pod "osmosis-validator" osmosisd q ibc channel channels -o json
            ;;
          *)
            echo "Unsupported chain ID: $chain_id"
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
    local source_port_id="$3"
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
        dest_channel_id=$(echo "$channel" | jq -r '.counterparty.channel_id')
        dest_port_id=$(echo "$channel" | jq -r '.counterparty.port_id')

        # Check if both the destination channel ID and port match.
        if [ "$dest_channel_id" = "$dest_channel_id" ] && [ "$dest_port_id" = "$dest_port_id" ]; then
            echo "$dest_channel_id"
            return
        fi
    done
}

# Assign source channel ID variables for each path.
# TODO_IN_THIS_COMMIT: comment and/or update - assumes star topology and only transfer port (no ICA)
export AGORIC_POCKET_SRC_CHANNEL_ID="channel-0"
export AXELAR_POCKET_SRC_CHANNEL_ID="channel-0"
export OSMOSIS_POCKET_SRC_CHANNEL_ID="channel-0"
export POCKET_AGORIC_SRC_CHANNEL_ID=$(ibc_get_counterparty_channel_id "agoriclocal" "channel-0" "transfer")
export POCKET_AXELAR_SRC_CHANNEL_ID=$(ibc_get_counterparty_channel_id "axelar" "channel-0" "transfer")
#export POCKET_OSMOSIS_SRC_CHANNEL_ID=$(ibc_get_counterparty_channel_id "osmosis" "channel-0" "transfer")

#echo "pocket_agoric_src_channel_id: $POCKET_AGORIC_SRC_CHANNEL_ID"
#echo "pocket_axelar_src_channel_id: $POCKET_AXELAR_SRC_CHANNEL_ID"
#echo "pocket_osmosis_src_channel_id: $POCKET_OSMOSIS_SRC_CHANNEL_ID"
