#!/bin/bash
# Script to add or modify services on Pocket Network
# Usage: ./add-services.sh [OPTIONS] <SERVICES_FILE> <NETWORK> <ADDRESS> [HOME_DIR]
# Example: ./add-services.sh services.txt beta my-key
# Example: ./add-services.sh services.txt main my-key ~/.poktroll

# Function to display help
show_help() {
    echo "Usage: $0 [OPTIONS] <SERVICES_FILE> <NETWORK> <ADDRESS> [HOME_DIR]"
    echo
    echo "Add or modify services on Pocket Network using pocketd from a tab-separated or whitespace-separated file"
    echo
    echo "Arguments:"
    echo "  SERVICES_FILE  Path to tab-separated or whitespace-separated file with columns: service_id, service_description, CUTTM"
    echo "  NETWORK        Network to use: 'main' or 'beta'"
    echo "                 - main: https://shannon-grove-rpc.mainnet.poktroll.com (pocket)"
    echo "                 - beta: https://shannon-testnet-grove-rpc.beta.poktroll.com (pocket-beta)"
    echo "  ADDRESS        Address/key name for the --from flag (e.g., my-key)"
    echo "  HOME_DIR       Directory path for the --home flag (default: ~/.pocket)"
    echo
    echo "Options:"
    echo "  --dry-run      Show commands that would be executed without running them"
    echo "  -h, --help     Show this help message and exit"
    echo
    echo "Services file format (tab-separated or whitespace-separated):"
    echo "  service_id<TAB>service_description<TAB>CUTTM"
    echo "  OR"
    echo "  service_id service_description CUTTM"
    echo "  Example:"
    echo "    eth	Ethereum	1"
    echo "    bitcoin	Bitcoin	2"
    echo "    polygon \"Polygon Network\" 3"
    echo
    echo "IMPORTANT: Check current fees for adding/modifying services by running:"
    echo "  pocketd query service params --node <NODE_URL>"
    echo "  This script uses a default fee of 20000upokt which may not be current."
    echo
    echo "Example:"
    echo "  $0 services.txt beta my-key"
    echo "  $0 services.txt main my-key ~/.poktroll"
    echo "  $0 --dry-run services.txt beta my-key"
    echo
}

# Initialize variables
DRY_RUN=false

# Parse options
while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        -*)
            echo "Error: Unknown option $1"
            echo
            show_help
            exit 1
            ;;
        *)
            break
            ;;
    esac
done

# Check if minimum required arguments are provided
if [ $# -lt 3 ]; then
  echo "Error: Missing required arguments"
  echo
  show_help
  exit 1
fi

# Assign arguments to variables
SERVICES_FILE=$1
NETWORK=$2
ADDRESS=$3
HOME_DIR=${4:-~/.pocket}

# Validate network and set corresponding values
case $NETWORK in
    main)
        NODE_URL="https://shannon-grove-rpc.mainnet.poktroll.com"
        CHAIN_ID="pocket"
        ;;
    beta)
        NODE_URL="https://shannon-testnet-grove-rpc.beta.poktroll.com"
        CHAIN_ID="pocket-beta"
        ;;
    *)
        echo "Error: Invalid network '$NETWORK'. Must be 'main' or 'beta'"
        exit 1
        ;;
esac

# Check if services file exists
if [ ! -f "$SERVICES_FILE" ]; then
    echo "Error: Services file '$SERVICES_FILE' not found"
    exit 1
fi

# Check if services file is readable
if [ ! -r "$SERVICES_FILE" ]; then
    echo "Error: Cannot read services file '$SERVICES_FILE'"
    exit 1
fi

# Display configuration
if [ "$DRY_RUN" = true ]; then
    echo "DRY RUN MODE - Commands will be displayed but not executed"
    echo ""
fi

echo "Adding or modifying services on Pocket Network using pocketd..."
echo "Using SERVICES_FILE: $SERVICES_FILE"
echo "Using NETWORK: $NETWORK"
echo "Using NODE: $NODE_URL"
echo "Using CHAIN_ID: $CHAIN_ID"
echo "Using HOME: $HOME_DIR"
echo "Using ADDRESS: $ADDRESS"
echo ""

# Counter for tracking services
service_count=0
success_count=0
error_count=0

# Read the services file and process each line
while IFS=$'\t' read -r service_id service_description cuttm || [[ -n "$service_id" ]]; do
    # Skip empty lines and lines starting with #
    if [[ -z "$service_id" || "$service_id" =~ ^[[:space:]]*# ]]; then
        continue
    fi
    
    # If tab parsing didn't work (no tabs), try space parsing
    if [[ -z "$service_description" || -z "$cuttm" ]]; then
        # Fall back to space-separated parsing for lines without tabs
        read -r service_id service_description cuttm <<< "$service_id"
    fi
    
    # Skip if we don't have all required fields
    if [[ -z "$service_id" || -z "$service_description" || -z "$cuttm" ]]; then
        echo "Warning: Skipping invalid line: $service_id $service_description $cuttm"
        continue
    fi
    
    # Increment counter
    ((service_count++))
    
    # Build the command
    cmd="pocketd tx service add-service \"$service_id\" \"$service_description\" $cuttm --node $NODE_URL --fees 20000upokt --from $ADDRESS --chain-id $CHAIN_ID --home $HOME_DIR --unordered --timeout-duration=60s --yes"
    
    if [ "$DRY_RUN" = true ]; then
        echo "[$service_count] $cmd"
    else
        echo "[$service_count] Adding/modifying service: $service_id ($service_description) with CUTTM: $cuttm"
        
        # Capture the command output
        output=$(eval $cmd 2>&1)
        exit_code=$?
        
        # Check if the command succeeded and doesn't contain meaningful raw_log (which indicates an error)
        if [ $exit_code -eq 0 ] && (! echo "$output" | grep -q "raw_log" || echo "$output" | grep -q 'raw_log: ""'); then
            ((success_count++))
            echo "  ✅ Success"
            # Extract and display transaction hash
            tx_hash=$(echo "$output" | grep -o 'txhash: [A-Fa-f0-9]*' | cut -d' ' -f2)
            if [ -n "$tx_hash" ]; then
                echo "  Transaction hash: $tx_hash"
            fi
        else
            ((error_count++))
            echo "  ❌ Failed"
            if echo "$output" | grep -q "raw_log" && ! echo "$output" | grep -q 'raw_log: ""'; then
                echo "  Error details: $(echo "$output" | grep "raw_log" | head -1)"
            elif [ $exit_code -ne 0 ]; then
                echo "  Command failed with exit code: $exit_code"
                # Show some of the error output for debugging
                echo "  Error output: $(echo "$output" | head -3 | tr '\n' ' ')"
            fi
        fi
        echo ""
        
        # Wait 5 seconds between each transaction with a loader animation
        echo "  Waiting 5 seconds before next transaction..."
        for i in {1..5}; do
            printf "\r  [%d/5] %s" $i "$(printf '%*s' $((i % 4 + 1)) | tr ' ' '|/-\\')"
            sleep 1
        done
        printf "\r  [5/5] ✓ Ready for next transaction\n"
    fi
    
done < "$SERVICES_FILE"

# Display summary
echo ""
echo "Summary:"
echo "  Total services processed: $service_count"
if [ "$DRY_RUN" = false ]; then
    echo "  Successful operations: $success_count"
    echo "  Failed operations: $error_count"
fi

if [ "$DRY_RUN" = true ]; then
    echo ""
    echo "DRY RUN completed. Use the script without --dry-run to execute the commands."
elif [ $error_count -gt 0 ]; then
    echo ""
    echo "Some services failed to be added/modified. Please check the output above for details."
    exit 1
else
    echo ""
    echo "All services have been added/modified successfully!"
fi
