#!/bin/bash

# Pocket Shannon Application Staking Script
# This script stakes applications on the Pocket network
# Usage: ./stake-apps.sh <address> <amount> <service_id> [flags]
#        ./stake-apps.sh --file <file> [flags]

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_color() {
    printf "${1}${2}${NC}\n"
}

# Function to show usage and help
show_help() {
    cat << EOF
$(print_color $BLUE "Pocket Shannon Application Staking Script")

$(print_color $YELLOW "USAGE:")
    Single mode:  $0 <address> <amount> <service_id> [OPTIONS]
    Batch mode:   $0 --file <file> [OPTIONS]

$(print_color $YELLOW "ARGUMENTS (Single mode):")
    address         Application address to stake from
    amount          Amount to stake (without upokt suffix)
    service_id      Service ID to stake for

$(print_color $YELLOW "OPTIONS:")
    --file FILE, -f FILE           Batch stake from file (format: address service_id amount per line)
    --delegate GATEWAY_ADDR        Delegate to gateway after staking (60s delay)
    --dry-run                     Show commands that would be executed without running them
    --node RPC_ENDPOINT           Custom RPC endpoint for pocketd
    --home DIR                    Home directory for pocketd (pocketd default if not specified)
    --keyring-backend BACKEND     Keyring backend (pocketd default if not specified)
    --chain-id CHAIN              Chain ID (default: pocket)
    --help, -h                    Show this help message

$(print_color $YELLOW "EXAMPLES:")
    Single stake:
    $0 pokt1abc123... 1000000 anvil

    Single stake with custom flags:
    $0 pokt1abc123... 1000000 anvil --node https://rpc.pocket.network --chain-id mainnet

    Batch stake from file:
    $0 --file stakes.txt

    Stake and delegate:
    $0 pokt1abc123... 1000000 anvil --delegate pokt1gateway456...

    Dry run:
    $0 pokt1abc123... 1000000 anvil --dry-run

    Batch dry run:
    $0 --file stakes.txt --dry-run

$(print_color $YELLOW "FILE FORMAT (for --file option):")
    Each line should contain: address service_id amount
    Example:
    pokt1abc123... anvil 1000000
    pokt1def456... ethereum 2000000
    pokt1ghi789... polygon 1500000

$(print_color $RED "NOTES:")
    - Amount should be specified without 'upokt' suffix (will be added automatically)
    - Fees of 200000upokt are automatically added to stake commands
    - Delegation fees of 20000upokt are automatically added to delegate commands
    - 60 second delay is enforced between staking and delegation
EOF
}

# Initialize variables
address=""
amount=""
service_id=""
file_path=""
delegate_addr=""
node_flag=""
home_dir=""
keyring_backend=""
chain_id="pocket"
dry_run=false

# Function to build pocketd flags
build_pocketd_flags() {
    local flags=""
    
    if [[ -n $node_flag ]]; then
        flags="$flags --node=$node_flag"
    fi
    
    if [[ -n $home_dir ]]; then
        flags="$flags --home=$home_dir"
    fi
    
    if [[ -n $keyring_backend ]]; then
        flags="$flags --keyring-backend=$keyring_backend"
    fi
    
    flags="$flags --chain-id=$chain_id"
    
    echo "$flags"
}

# Function to create stake config file
create_stake_config() {
    local stake_amount="$1"
    local stake_service_id="$2"
    local config_file="/tmp/stake_app_config.yaml"
    
    if [[ "$dry_run" == true ]]; then
        print_color $YELLOW "🔍 [DRY RUN] Would create stake configuration file: $config_file"
        print_color $BLUE "📄 Config content would be:"
        cat <<🚀
stake_amount: ${stake_amount}upokt
service_ids:
  - "${stake_service_id}"
🚀
        return 0
    fi
    
    print_color $BLUE "📝 Creating stake configuration file..."
    
    cat <<🚀 > "$config_file"
stake_amount: ${stake_amount}upokt
service_ids:
  - "${stake_service_id}"
🚀

    if [[ $? -eq 0 ]]; then
        print_color $GREEN "✅ Configuration file created successfully: $config_file"
        return 0
    else
        print_color $RED "❌ Failed to create configuration file"
        return 1
    fi
}

# Function to stake application
stake_application() {
    local from_addr="$1"
    local stake_amount="$2"
    local stake_service_id="$3"
    
    if [[ "$dry_run" == true ]]; then
        print_color $YELLOW "🔍 [DRY RUN] Would stake application for $from_addr"
        print_color $BLUE "   Amount: ${stake_amount}upokt"
        print_color $BLUE "   Service ID: $stake_service_id"
    else
        print_color $BLUE "🚀 Staking application for $from_addr..."
        print_color $YELLOW "   Amount: ${stake_amount}upokt"
        print_color $YELLOW "   Service ID: $stake_service_id"
    fi
    
    # Create config file
    if ! create_stake_config "$stake_amount" "$stake_service_id"; then
        return 1
    fi
    
    # Build command flags
    local pocketd_flags=$(build_pocketd_flags)
    
    # Execute stake command
    local stake_cmd="pocketd tx application stake-application --config=/tmp/stake_app_config.yaml --from=$from_addr $pocketd_flags --fees=200000upokt --yes"
    
    if [[ "$dry_run" == true ]]; then
        print_color $YELLOW "🔍 [DRY RUN] Would execute:"
        print_color $BLUE "   $stake_cmd"
        print_color $GREEN "✅ [DRY RUN] Would successfully stake application for $from_addr"
        return 0
    fi
    
    print_color $BLUE "🔨 Executing: $stake_cmd"
    
    if eval "$stake_cmd"; then
        print_color $GREEN "✅ Successfully staked application for $from_addr"
        return 0
    else
        print_color $RED "❌ Failed to stake application for $from_addr"
        return 1
    fi
}

# Function to delegate to gateway
delegate_to_gateway() {
    local from_addr="$1"
    local gateway_addr="$2"
    local skip_wait="${3:-false}"
    
    if [[ "$dry_run" == true ]]; then
        if [[ "$skip_wait" != true ]]; then
            print_color $YELLOW "🔍 [DRY RUN] Would wait 60 seconds before delegation..."
        fi
        print_color $YELLOW "🔍 [DRY RUN] Would delegate $from_addr to gateway $gateway_addr"
        
        # Build command flags
        local pocketd_flags=$(build_pocketd_flags)
        
        # Show delegate command
        local delegate_cmd="pocketd tx application delegate-to-gateway $gateway_addr --from=$from_addr $pocketd_flags --fees=20000upokt"
        
        print_color $YELLOW "🔍 [DRY RUN] Would execute:"
        print_color $BLUE "   $delegate_cmd"
        print_color $GREEN "✅ [DRY RUN] Would successfully delegate $from_addr to gateway $gateway_addr"
        return 0
    fi
    
    if [[ "$skip_wait" != true ]]; then
        print_color $BLUE "⏳ Waiting 60 seconds before delegation..."
        sleep 60
    fi
    
    print_color $BLUE "🔗 Delegating $from_addr to gateway $gateway_addr..."
    
    # Build command flags
    local pocketd_flags=$(build_pocketd_flags)
    
    # Execute delegate command
    local delegate_cmd="pocketd tx application delegate-to-gateway $gateway_addr --from=$from_addr $pocketd_flags --fees=20000upokt --yes"
    
    print_color $BLUE "🔨 Executing: $delegate_cmd"
    
    if eval "$delegate_cmd"; then
        print_color $GREEN "✅ Successfully delegated $from_addr to gateway $gateway_addr"
        return 0
    else
        print_color $RED "❌ Failed to delegate $from_addr to gateway $gateway_addr"
        return 1
    fi
}

# Function to process batch file
process_batch_file() {
    local file="$1"
    local total_lines=0
    local successful_stakes=0
    local failed_stakes=0
    local successful_delegations=0
    local failed_delegations=0
    local staked_addresses=()
    
    print_color $BLUE "📂 Processing batch file: $file"
    
    if [[ ! -f "$file" ]]; then
        print_color $RED "❌ File not found: $file"
        return 1
    fi
    
    # Count total lines
    total_lines=$(wc -l < "$file")
    print_color $YELLOW "📊 Total lines to process: $total_lines"
    echo
    
    print_color $BLUE "🚀 PHASE 1: STAKING APPLICATIONS"
    print_color $BLUE "================================"
    echo
    
    local line_num=0
    while IFS= read -r line; do
        line_num=$((line_num + 1))
        
        # Skip empty lines and comments
        if [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]]; then
            continue
        fi
        
        # Parse line
        read -r line_address line_service_id line_amount <<< "$line"
        
        if [[ -z "$line_address" || -z "$line_service_id" || -z "$line_amount" ]]; then
            print_color $RED "❌ Line $line_num: Invalid format - $line"
            failed_stakes=$((failed_stakes + 1))
            continue
        fi
        
        print_color $BLUE "🔄 Processing line $line_num/$total_lines..."
        
        # Stake application
        if stake_application "$line_address" "$line_amount" "$line_service_id"; then
            successful_stakes=$((successful_stakes + 1))
            staked_addresses+=("$line_address")
        else
            failed_stakes=$((failed_stakes + 1))
        fi
        
        echo
    done < "$file"
    
    # Delegation phase
    if [[ -n "$delegate_addr" && ${#staked_addresses[@]} -gt 0 ]]; then
        echo
        print_color $BLUE "🔗 PHASE 2: DELEGATING TO GATEWAY"
        print_color $BLUE "================================="
        echo
        
        print_color $YELLOW "Will delegate ${#staked_addresses[@]} successfully staked addresses to gateway: $delegate_addr"
        
        if [[ "$dry_run" == true ]]; then
            print_color $YELLOW "🔍 [DRY RUN] Would wait 60 seconds before starting delegations..."
        else
            print_color $BLUE "⏳ Waiting 60 seconds before starting delegations..."
            sleep 60
        fi
        
        echo
        
        for address in "${staked_addresses[@]}"; do
            if delegate_to_gateway "$address" "$delegate_addr" true; then
                successful_delegations=$((successful_delegations + 1))
            else
                failed_delegations=$((failed_delegations + 1))
            fi
            echo
        done
    elif [[ -n "$delegate_addr" && ${#staked_addresses[@]} -eq 0 ]]; then
        echo
        print_color $YELLOW "⚠️  No successful stakes to delegate"
    fi
    
    # Generate report
    echo "========================================="
    print_color $GREEN "📊 BATCH PROCESSING REPORT"
    echo "========================================="
    print_color $BLUE "Total lines processed: $total_lines"
    print_color $GREEN "Successful stakes: $successful_stakes"
    print_color $RED "Failed stakes: $failed_stakes"
    
    if [[ -n "$delegate_addr" ]]; then
        print_color $GREEN "Successful delegations: $successful_delegations"
        print_color $RED "Failed delegations: $failed_delegations"
    fi
    echo "========================================="
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        --file|-f)
            if [[ -n $2 && $2 != -* ]]; then
                file_path="$2"
                shift 2
            else
                print_color $RED "Error: --file requires a file path"
                exit 1
            fi
            ;;
        --delegate)
            if [[ -n $2 && $2 != -* ]]; then
                delegate_addr="$2"
                shift 2
            else
                print_color $RED "Error: --delegate requires a gateway address"
                exit 1
            fi
            ;;
        --dry-run)
            dry_run=true
            shift
            ;;
        --node)
            if [[ -n $2 && $2 != -* ]]; then
                node_flag="$2"
                shift 2
            else
                print_color $RED "Error: --node requires an RPC endpoint"
                exit 1
            fi
            ;;
        --node=*)
            node_flag="${1#*=}"
            shift
            ;;
        --home)
            if [[ -n $2 && $2 != -* ]]; then
                home_dir="$2"
                shift 2
            else
                print_color $RED "Error: --home requires a directory path"
                exit 1
            fi
            ;;
        --home=*)
            home_dir="${1#*=}"
            shift
            ;;
        --keyring-backend)
            if [[ -n $2 && $2 != -* ]]; then
                keyring_backend="$2"
                shift 2
            else
                print_color $RED "Error: --keyring-backend requires a backend type"
                exit 1
            fi
            ;;
        --keyring-backend=*)
            keyring_backend="${1#*=}"
            shift
            ;;
        --chain-id)
            if [[ -n $2 && $2 != -* ]]; then
                chain_id="$2"
                shift 2
            else
                print_color $RED "Error: --chain-id requires a chain ID"
                exit 1
            fi
            ;;
        --chain-id=*)
            chain_id="${1#*=}"
            shift
            ;;
        -*)
            print_color $RED "Error: Unknown option $1"
            print_color $BLUE "Use --help or -h for usage information"
            exit 1
            ;;
        *)
            # Positional arguments for single mode
            if [[ -z "$file_path" ]]; then
                if [[ -z $address ]]; then
                    address="$1"
                elif [[ -z $amount ]]; then
                    amount="$1"
                elif [[ -z $service_id ]]; then
                    service_id="$1"
                else
                    print_color $RED "Error: Too many positional arguments"
                    print_color $BLUE "Use --help or -h for usage information"
                    exit 1
                fi
            fi
            shift
            ;;
    esac
done

# Set default home directory if not specified
# (Removed - let pocketd use its own default)

# Validate mode and arguments
if [[ -n "$file_path" ]]; then
    # Batch mode
    if [[ -n "$address" || -n "$amount" || -n "$service_id" ]]; then
        print_color $RED "Error: Cannot use positional arguments with --file option"
        exit 1
    fi
    
    print_color $BLUE "======================================="
    print_color $BLUE "  Pocket Application Staking (Batch)"
    print_color $BLUE "======================================="
    echo
    
    print_color $YELLOW "Configuration:"
    print_color $BLUE "  Mode: Batch"
    print_color $BLUE "  File: $file_path"
    print_color $BLUE "  Home directory: $home_dir"
    print_color $BLUE "  Keyring backend: $keyring_backend"
    print_color $BLUE "  Chain ID: $chain_id"
    if [[ -n "$node_flag" ]]; then
        print_color $BLUE "  Node: $node_flag"
    fi
    if [[ -n "$delegate_addr" ]]; then
        print_color $BLUE "  Delegate to: $delegate_addr"
    fi
    echo
    
    process_batch_file "$file_path"
    
else
    # Single mode
    if [[ -z $address || -z $amount || -z $service_id ]]; then
        print_color $RED "Error: Missing required arguments for single mode"
        print_color $BLUE "Usage: $0 <address> <amount> <service_id> [OPTIONS]"
        print_color $BLUE "Use --help or -h for detailed usage information"
        exit 1
    fi
    
    # Validate amount is numeric
    if ! [[ $amount =~ ^[0-9]+$ ]]; then
        print_color $RED "Error: Amount must be a positive integer"
        exit 1
    fi
    
    print_color $BLUE "======================================="
    print_color $BLUE "  Pocket Application Staking (Single)"
    print_color $BLUE "======================================="
    echo
    
    print_color $YELLOW "Configuration:"
    print_color $BLUE "  Mode: Single"
    print_color $BLUE "  Address: $address"
    print_color $BLUE "  Amount: ${amount}upokt"
    print_color $BLUE "  Service ID: $service_id"
    print_color $BLUE "  Home directory: $home_dir"
    print_color $BLUE "  Keyring backend: $keyring_backend"
    print_color $BLUE "  Chain ID: $chain_id"
    if [[ -n "$node_flag" ]]; then
        print_color $BLUE "  Node: $node_flag"
    fi
    if [[ -n "$delegate_addr" ]]; then
        print_color $BLUE "  Delegate to: $delegate_addr"
    fi
    echo
    
    # Stake application
    if stake_application "$address" "$amount" "$service_id"; then
        # Delegate if specified
        if [[ -n "$delegate_addr" ]]; then
            delegate_to_gateway "$address" "$delegate_addr"
        fi
        
        echo
        print_color $GREEN "🎉 Single stake operation completed successfully!"
    else
        echo
        print_color $RED "💥 Single stake operation failed!"
        exit 1
    fi
fi

# Cleanup
if [[ "$dry_run" != true ]]; then
    rm -f /tmp/stake_app_config.yaml
else
    print_color $YELLOW "🔍 [DRY RUN] Would cleanup temporary files"
fi
