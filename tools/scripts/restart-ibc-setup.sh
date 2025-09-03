#!/bin/bash
set -e

# Check if yq is installed
if ! command -v yq &> /dev/null; then
    echo "❌ yq is required but not installed. Please install it using:"
    echo "  Linux: sudo snap install yq"
    echo "  macOS: brew install yq"
    exit 1
fi

# Check if localnet_config.yaml exists
if [ ! -f "localnet_config.yaml" ]; then
    echo "❌ localnet_config.yaml not found in current directory"
    exit 1
fi

echo "🔄 Starting dynamic IBC restart and setup sequence..."

# Function to wait for pod to be ready
wait_for_pod_ready() {
    local pod_pattern="$1"
    local timeout=120
    local count=0
    
    echo "⏳ Waiting for pod matching '$pod_pattern' to be ready..."
    
    while [ $count -lt $timeout ]; do
        if kubectl get pods | grep "$pod_pattern" | grep -q "1/1.*Running"; then
            echo "✅ Pod matching '$pod_pattern' is ready"
            return 0
        fi
        sleep 1
        count=$((count + 1))
        if [ $((count % 10)) -eq 0 ]; then
            echo "   ... still waiting ($count/${timeout}s)"
        fi
    done
    
    echo "❌ Timeout waiting for pod matching '$pod_pattern'"
    return 1
}

# Function to trigger and wait for IBC setup job
trigger_and_wait_ibc_setup() {
    local setup_name="$1"
    echo "🏗️  Triggering IBC setup: $setup_name"
    
    # Trigger the setup job
    tilt trigger "$setup_name"
    
    # Wait a moment for the job to start
    sleep 10
    
    # Wait for the job to complete (either success or failure)
    local timeout=180  # 3 minutes
    local count=0
    
    while [ $count -lt $timeout ]; do
        # Find the most recent setup pod by creation time and check its status
        local recent_pod=$(kubectl get pods --sort-by=.metadata.creationTimestamp -o wide | grep "ibc-relayer-setup" | grep -v "agoric-osmosis" | tail -1)
        
        if [[ -n "$recent_pod" ]]; then
            local pod_status=$(echo "$recent_pod" | awk '{print $3}')
            local pod_name=$(echo "$recent_pod" | awk '{print $1}')
            
            if [[ "$pod_status" == "Completed" ]]; then
                echo "✅ IBC setup completed successfully (pod: $pod_name)"
                return 0
            elif [[ "$pod_status" == "Error" ]] || [[ "$pod_status" == "Failed" ]]; then
                echo "❌ IBC setup failed (pod: $pod_name)"
                # Show logs for debugging
                echo "📋 Last few lines of logs:"
                kubectl logs "$pod_name" | tail -10
                return 1
            fi
        fi
        
        sleep 3
        count=$((count + 3))
        if [ $((count % 15)) -eq 0 ]; then
            echo "   ... setup still running ($count/${timeout}s)"
        fi
    done
    
    echo "⚠️  Timeout waiting for IBC setup '$setup_name'"
    return 1
}

# Get enabled validator configs from localnet_config.yaml
echo "📋 Reading IBC configuration from localnet_config.yaml..."

# Check if IBC is enabled
ibc_enabled=$(yq '.ibc.enabled' localnet_config.yaml)
if [ "$ibc_enabled" != "true" ]; then
    echo "❌ IBC is not enabled in localnet_config.yaml"
    exit 1
fi

# Get list of enabled validators
echo "🔍 Finding enabled IBC validators..."
enabled_validators=()
validator_names=()

# Get all validator config keys and check if they're enabled
for validator in $(yq '.ibc.validator_configs | keys | .[]' localnet_config.yaml); do
    # Remove quotes from validator name
    validator_name=$(echo "$validator" | tr -d '"')
    enabled=$(yq ".ibc.validator_configs.$validator_name.enabled" localnet_config.yaml)
    tilt_ui_name=$(yq ".ibc.validator_configs.$validator_name.tilt_ui_name" localnet_config.yaml | tr -d '"')
    
    if [ "$enabled" = "true" ]; then
        echo "  ✅ Found enabled validator: $validator_name (UI: $tilt_ui_name)"
        enabled_validators+=("$validator_name")
        validator_names+=("$tilt_ui_name")
    fi
done

if [ ${#enabled_validators[@]} -eq 0 ]; then
    echo "❌ No enabled IBC validators found in configuration"
    exit 1
fi

# Step 1: Restart all counterparty validators at once
echo "🔄 Restarting ${#enabled_validators[@]} counterparty validators..."
for tilt_name in "${validator_names[@]}"; do
    echo "  🔄 Triggering restart for: $tilt_name"
    tilt trigger "$tilt_name"
done

# Wait for all to be ready
for i in "${!enabled_validators[@]}"; do
    validator="${enabled_validators[$i]}"
    echo "⏳ Waiting for $validator validator to be ready..."
    wait_for_pod_ready "$validator-validator"
done

echo "⏸️  Waiting 30s for all validators to stabilize..."
sleep 30

# Step 2: Wait for pocket validator to be stable
echo "🔍 Checking Pocket validator status..."
wait_for_pod_ready "validator-pocket-validator"

# Step 3: Set up IBC connections based on localnet_config.yaml
echo "🔗 Setting up IBC connections..."

# Get enabled connections from config
connections_count=$(yq '.ibc.connections | length' localnet_config.yaml)
echo "📋 Found $connections_count connection(s) in configuration"

setup_names=()
failed_setups=()

for i in $(seq 0 $((connections_count - 1))); do
    connection_enabled=$(yq ".ibc.connections[$i].enabled" localnet_config.yaml)
    
    if [ "$connection_enabled" = "true" ]; then
        chain_a=$(yq ".ibc.connections[$i].chain_a" localnet_config.yaml | tr -d '"')
        chain_b=$(yq ".ibc.connections[$i].chain_b" localnet_config.yaml | tr -d '"')
        
        # Check if there's a custom setup resource name
        custom_name=$(yq ".ibc.connections[$i].setup_resource_name" localnet_config.yaml 2>/dev/null | tr -d '"')
        
        if [ "$custom_name" != "null" ] && [ -n "$custom_name" ]; then
            setup_name="$custom_name"
        else
            # Generate default setup name based on chain pair
            if [ "$chain_a" = "pocket" ]; then
                setup_name="🏗️ Pokt->$(echo "$chain_b" | sed 's/^./\U&/')"
            elif [ "$chain_b" = "pocket" ]; then
                setup_name="🏗️ $(echo "$chain_a" | sed 's/^./\U&/')->Pokt"
            else
                setup_name="🏗️ $(echo "$chain_a" | sed 's/^./\U&/')<->$(echo "$chain_b" | sed 's/^./\U&/')"
            fi
        fi
        
        echo "🔗 Setting up connection: $chain_a ↔ $chain_b (setup: $setup_name)"
        setup_names+=("$setup_name")
        
        if ! trigger_and_wait_ibc_setup "$setup_name"; then
            echo "❌ Failed to setup $chain_a ↔ $chain_b connection"
            failed_setups+=("$chain_a ↔ $chain_b")
        else
            echo "✅ Successfully setup $chain_a ↔ $chain_b connection"
        fi
        
        # Wait between setups to avoid overwhelming the system
        if [ $i -lt $((connections_count - 1)) ]; then
            echo "⏸️  Waiting 10s before next connection setup..."
            sleep 10
        fi
    else
        chain_a=$(yq ".ibc.connections[$i].chain_a" localnet_config.yaml | tr -d '"')
        chain_b=$(yq ".ibc.connections[$i].chain_b" localnet_config.yaml | tr -d '"')
        echo "⏭️  Skipping disabled connection: $chain_a ↔ $chain_b"
    fi
done

# Report results
if [ ${#failed_setups[@]} -eq 0 ]; then
    echo "🎉 All IBC connections setup completed successfully!"
else
    echo "⚠️  Some IBC connections failed to setup:"
    for failed in "${failed_setups[@]}"; do
        echo "  ❌ $failed"
    done
fi

# Step 6: Test channel discovery
echo "🔍 Testing channel discovery..."
IBC_DEBUG=true source tools/scripts/ibc-channels.sh

echo "✅ IBC restart and setup sequence completed!"