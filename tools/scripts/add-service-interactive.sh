#!/bin/bash
# Interactive script to add or modify services on Pocket Network
# Usage: ./add-service-interactive.sh [--dry-run]
#
# Reference: docusaurus/docs/1_operate/1_cheat_sheets/1_service_cheatsheet.md

set -e

# Colors for better UX
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Initialize variables
DRY_RUN=false
SERVICES=()

# Service constraints from documentation
MAX_SERVICE_ID_LENGTH=42
MAX_SERVICE_DESCRIPTION_LENGTH=169

# Parse options
while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [--dry-run]"
            echo ""
            echo "Interactive script to add or modify services on Pocket Network"
            echo ""
            echo "Options:"
            echo "  --dry-run    Show commands that would be executed without running them"
            echo "  -h, --help   Show this help message and exit"
            echo ""
            echo "Reference: docusaurus/docs/1_operate/1_cheat_sheets/1_service_cheatsheet.md"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Function to print section headers (always to terminal)
print_header() {
    {
        echo ""
        echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${BOLD}${CYAN}  $1${NC}"
        echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo ""
    } > /dev/tty
}

# Function to print info (always to terminal)
print_info() {
    echo -e "${CYAN}ℹ${NC}  $1" > /dev/tty
}

# Function to print success (always to terminal)
print_success() {
    echo -e "${GREEN}✓${NC}  $1" > /dev/tty
}

# Function to print warning (always to terminal)
print_warning() {
    echo -e "${YELLOW}⚠${NC}  $1" > /dev/tty
}

# Function to print error (always to terminal)
print_error() {
    echo -e "${RED}✗${NC}  $1" > /dev/tty
}

# Function to prompt with default value
# Note: Prompt is sent to /dev/tty so it displays even when output is captured
prompt_with_default() {
    local prompt="$1"
    local default="$2"
    local result

    if [ -n "$default" ]; then
        read -rp "$(echo -e "${BOLD}$prompt${NC} [${GREEN}$default${NC}]: ")" result < /dev/tty > /dev/tty
        echo "${result:-$default}"
    else
        read -rp "$(echo -e "${BOLD}$prompt${NC}: ")" result < /dev/tty > /dev/tty
        echo "$result"
    fi
}

# Function to prompt yes/no
# Note: Prompt is sent to /dev/tty so it displays even when output is captured
prompt_yes_no() {
    local prompt="$1"
    local default="$2"
    local result

    while true; do
        if [ "$default" = "y" ]; then
            read -rp "$(echo -e "${BOLD}$prompt${NC} [${GREEN}Y${NC}/n]: ")" result < /dev/tty > /dev/tty
            result="${result:-y}"
        else
            read -rp "$(echo -e "${BOLD}$prompt${NC} [y/${GREEN}N${NC}]: ")" result < /dev/tty > /dev/tty
            result="${result:-n}"
        fi

        # Convert to lowercase (compatible with bash 3.x)
        result=$(echo "$result" | tr '[:upper:]' '[:lower:]')

        case "$result" in
            y|yes) echo "y"; return ;;
            n|no) echo "n"; return ;;
            *) echo -e "${RED}✗${NC}  Please answer 'y' or 'n'" > /dev/tty ;;
        esac
    done
}

# Function to select from options
# Note: Menu is printed to /dev/tty so it displays even when output is captured
select_option() {
    local prompt="$1"
    shift
    local options=("$@")
    local selection

    # Print menu to /dev/tty so it shows even when stdout is captured
    {
        echo -e "${BOLD}$prompt${NC}"
        echo ""
        for i in "${!options[@]}"; do
            echo -e "  ${CYAN}$((i+1))${NC}) ${options[$i]}"
        done
        echo ""
    } > /dev/tty

    while true; do
        read -rp "$(echo -e "${BOLD}Enter choice [1-${#options[@]}]:${NC} ")" selection < /dev/tty > /dev/tty
        if [[ "$selection" =~ ^[0-9]+$ ]] && [ "$selection" -ge 1 ] && [ "$selection" -le "${#options[@]}" ]; then
            echo "$((selection-1))"
            return
        fi
        echo -e "${RED}✗${NC}  Invalid selection. Please enter a number between 1 and ${#options[@]}" > /dev/tty
    done
}

# Function to validate service ID
validate_service_id() {
    local id="$1"
    if [[ -z "$id" ]]; then
        print_error "Service ID cannot be empty"
        return 1
    fi
    if [[ ${#id} -gt $MAX_SERVICE_ID_LENGTH ]]; then
        print_error "Service ID cannot exceed $MAX_SERVICE_ID_LENGTH characters (got ${#id})"
        return 1
    fi
    if [[ ! "$id" =~ ^[a-zA-Z0-9_-]+$ ]]; then
        print_error "Service ID can only contain letters, numbers, underscores, and hyphens"
        return 1
    fi
    return 0
}

# Function to validate service description
validate_service_description() {
    local desc="$1"
    if [[ -z "$desc" ]]; then
        print_error "Service description cannot be empty"
        return 1
    fi
    if [[ ${#desc} -gt $MAX_SERVICE_DESCRIPTION_LENGTH ]]; then
        print_error "Service description cannot exceed $MAX_SERVICE_DESCRIPTION_LENGTH characters (got ${#desc})"
        return 1
    fi
    return 0
}

# Function to validate compute_units_per_relay
validate_compute_units() {
    local units="$1"
    if [[ ! "$units" =~ ^[0-9]+$ ]]; then
        print_error "Compute units per relay must be a positive integer"
        return 1
    fi
    if [ "$units" -lt 1 ]; then
        print_error "Compute units per relay must be at least 1"
        return 1
    fi
    return 0
}

# Function to validate metadata file
validate_metadata_file() {
    local file="$1"
    if [[ -n "$file" ]]; then
        if [[ ! -f "$file" ]]; then
            print_error "Metadata file not found: $file"
            return 1
        fi
        if [[ ! -r "$file" ]]; then
            print_error "Cannot read metadata file: $file"
            return 1
        fi
        # Check file size (256 KiB max when decoded)
        local file_size
        file_size=$(wc -c < "$file")
        if [ "$file_size" -gt 262144 ]; then
            print_error "Metadata file exceeds 256 KiB limit (got $file_size bytes)"
            return 1
        fi
    fi
    return 0
}

# Function to collect service details
collect_service() {
    local service_id service_description compute_units metadata_file

    print_header "Add New Service"

    # Service ID
    echo -e "${CYAN}Service ID Constraints:${NC}"
    echo "  - Maximum $MAX_SERVICE_ID_LENGTH characters"
    echo "  - Only letters, numbers, underscores, and hyphens allowed"
    echo ""

    while true; do
        service_id=$(prompt_with_default "Service ID (unique identifier, e.g., 'eth', 'polygon')" "")
        if validate_service_id "$service_id"; then
            break
        fi
    done

    # Service Description
    echo ""
    echo -e "${CYAN}Service Description Constraints:${NC}"
    echo "  - Maximum $MAX_SERVICE_DESCRIPTION_LENGTH characters"
    echo ""

    while true; do
        service_description=$(prompt_with_default "Service Description (human-readable name)" "$service_id")
        if [ -z "$service_description" ]; then
            service_description="$service_id"
        fi
        if validate_service_description "$service_description"; then
            break
        fi
    done

    # Compute Units Per Relay
    echo ""
    echo -e "${CYAN}About Compute Units Per Relay:${NC}"
    echo "  This value represents how many compute units each relay request costs."
    echo "  Higher values indicate more resource-intensive operations."
    echo "  Default is 1 for simple RPC calls."
    echo ""

    while true; do
        compute_units=$(prompt_with_default "Compute Units Per Relay" "1")
        if validate_compute_units "$compute_units"; then
            break
        fi
    done

    # Metadata (optional)
    echo ""
    echo -e "${CYAN}About Service Metadata (Optional):${NC}"
    echo "  You can attach an API specification (OpenAPI, OpenRPC, etc.) to the service."
    echo "  This is stored on-chain and helps describe the service's interface."
    echo "  Maximum size: 256 KiB"
    echo ""

    add_metadata=$(prompt_yes_no "Do you want to add API metadata from a file?" "n")
    if [ "$add_metadata" = "y" ]; then
        while true; do
            metadata_file=$(prompt_with_default "Path to metadata file (e.g., ./openapi.json)" "")
            if [[ -z "$metadata_file" ]]; then
                print_info "Skipping metadata"
                break
            fi
            if validate_metadata_file "$metadata_file"; then
                break
            fi
        done
    else
        metadata_file=""
    fi

    # Store the service (using | as delimiter, with metadata path)
    SERVICES+=("$service_id|$service_description|$compute_units|$metadata_file")

    echo ""
    print_success "Service added: $service_id"
    print_info "  Description: $service_description"
    print_info "  Compute Units Per Relay: $compute_units"
    if [[ -n "$metadata_file" ]]; then
        print_info "  Metadata file: $metadata_file"
    fi
}

# Function to display all collected services
display_services() {
    if [ ${#SERVICES[@]} -eq 0 ]; then
        print_warning "No services have been added yet"
        return
    fi

    echo ""
    echo -e "${BOLD}Services to be added/modified:${NC}"
    echo ""
    printf "  ${CYAN}%-4s %-20s %-35s %-8s %s${NC}\n" "#" "Service ID" "Description" "Units" "Metadata"
    echo "  ──────────────────────────────────────────────────────────────────────────────────"

    local i=1
    for service in "${SERVICES[@]}"; do
        IFS='|' read -r sid sdesc sunits smeta <<< "$service"
        local meta_display="none"
        if [[ -n "$smeta" ]]; then
            meta_display="$(basename "$smeta")"
        fi
        # Truncate description if too long for display
        if [[ ${#sdesc} -gt 32 ]]; then
            sdesc="${sdesc:0:29}..."
        fi
        printf "  %-4s %-20s %-35s %-8s %s\n" "$i" "$sid" "$sdesc" "$sunits" "$meta_display"
        ((i++))
    done
    echo ""
}

# Function to remove a service from the list
remove_service() {
    if [ ${#SERVICES[@]} -eq 0 ]; then
        print_warning "No services to remove"
        return
    fi

    display_services

    read -rp "$(echo -e "${BOLD}Enter the number of the service to remove (or 0 to cancel):${NC} ")" selection

    if [[ "$selection" =~ ^[0-9]+$ ]] && [ "$selection" -ge 1 ] && [ "$selection" -le "${#SERVICES[@]}" ]; then
        local removed="${SERVICES[$((selection-1))]}"
        IFS='|' read -r sid _ _ _ <<< "$removed"
        unset 'SERVICES[$((selection-1))]'
        SERVICES=("${SERVICES[@]}")  # Re-index array
        print_success "Removed service: $sid"
    elif [ "$selection" != "0" ]; then
        print_error "Invalid selection"
    fi
}

# Function to execute the service additions
execute_services() {
    local network="$1"
    local address="$2"
    local success_count=0
    local error_count=0
    local total=${#SERVICES[@]}

    print_header "Executing Service Additions"

    if [ "$DRY_RUN" = true ]; then
        print_warning "DRY RUN MODE - Commands will be displayed but not executed"
        echo ""
    fi

    local i=1
    for service in "${SERVICES[@]}"; do
        IFS='|' read -r service_id service_description compute_units metadata_file <<< "$service"

        # Build the command
        cmd="pocketd tx service add-service \"$service_id\" \"$service_description\" $compute_units"

        # Add metadata flag if present
        if [[ -n "$metadata_file" ]]; then
            cmd="$cmd --experimental-metadata-file \"$metadata_file\""
        fi

        cmd="$cmd --gas auto --gas-adjustment 1.5 --from $address --network=$network --yes"

        if [ "$DRY_RUN" = true ]; then
            echo -e "${CYAN}[$i/$total]${NC} $cmd"
            echo ""
        else
            echo -e "${CYAN}[$i/$total]${NC} Adding/modifying service: ${BOLD}$service_id${NC}"
            echo "  Description: $service_description"
            echo "  Compute Units: $compute_units"

            output=$(eval "$cmd" 2>&1)
            exit_code=$?

            if [ $exit_code -eq 0 ] && (! echo "$output" | grep -q "raw_log" || echo "$output" | grep -q 'raw_log: ""'); then
                ((success_count++))
                print_success "Success"
                tx_hash=$(echo "$output" | grep -o 'txhash: [A-Fa-f0-9]*' | cut -d' ' -f2)
                if [ -n "$tx_hash" ]; then
                    echo -e "  Transaction hash: ${GREEN}$tx_hash${NC}"
                fi
            else
                ((error_count++))
                print_error "Failed"
                if echo "$output" | grep -q "raw_log" && ! echo "$output" | grep -q 'raw_log: ""'; then
                    echo "  Error: $(echo "$output" | grep "raw_log" | head -1)"
                elif [ $exit_code -ne 0 ]; then
                    echo "  Command failed with exit code: $exit_code"
                    echo "  Error: $(echo "$output" | head -3 | tr '\n' ' ')"
                fi
            fi
            echo ""

            # Wait between transactions (except for the last one)
            if [ $i -lt $total ]; then
                echo -n "  Waiting 5 seconds before next transaction "
                for j in {1..5}; do
                    echo -n "."
                    sleep 1
                done
                echo " done"
                echo ""
            fi
        fi

        ((i++))
    done

    # Summary
    print_header "Summary"
    echo "  Total services processed: $total"
    if [ "$DRY_RUN" = false ]; then
        echo -e "  Successful: ${GREEN}$success_count${NC}"
        if [ $error_count -gt 0 ]; then
            echo -e "  Failed: ${RED}$error_count${NC}"
        else
            echo "  Failed: 0"
        fi
    fi

    if [ "$DRY_RUN" = true ]; then
        echo ""
        print_info "DRY RUN completed. Run without --dry-run to execute the commands."
    elif [ $error_count -gt 0 ]; then
        echo ""
        print_warning "Some services failed. Please check the output above for details."
        return 1
    else
        echo ""
        print_success "All services have been added/modified successfully!"
        echo ""
        print_info "You can query your service with:"
        echo "  pocketd query service show-service <SERVICE_ID> --network=$network"
    fi
}

# Main script starts here
clear
print_header "Pocket Network Service Manager"

echo "This interactive script will guide you through adding or modifying"
echo "services on the Pocket Network."
echo ""
echo -e "${CYAN}Reference:${NC} docusaurus/docs/1_operate/1_cheat_sheets/1_service_cheatsheet.md"
echo ""

if [ "$DRY_RUN" = true ]; then
    print_warning "DRY RUN MODE ENABLED - No changes will be made"
    echo ""
fi

# Step 1: Select Network
print_header "Step 1: Select Network"

networks=("Beta TestNet (pocket-lego-testnet)" "Mainnet (pocket)")
network_idx=$(select_option "Which network do you want to use?" "${networks[@]}")

case $network_idx in
    0)
        NETWORK="beta"
        NETWORK_NAME="Beta TestNet"
        ;;
    1)
        NETWORK="main"
        NETWORK_NAME="Mainnet"
        print_warning "You selected Mainnet. Please ensure you have the correct permissions."
        echo ""
        ;;
esac

print_success "Selected: $NETWORK_NAME"

# Step 2: Configure Wallet
print_header "Step 2: Configure Wallet"

echo "You'll need to specify the signing key/address."
echo -e "${YELLOW}Important:${NC} The signer becomes the owner of the service."
echo "           Only the owner can update the service later."
echo ""

# List available keys if possible
echo -e "${CYAN}Checking for available keys...${NC}"
if command -v pocketd &> /dev/null; then
    keys_output=$(pocketd keys list 2>/dev/null || echo "")
    if [ -n "$keys_output" ] && [ "$keys_output" != "[]" ]; then
        echo ""
        echo -e "${CYAN}Available keys (showing first few):${NC}"
        echo "$keys_output" | head -20
        echo ""
        print_info "Run 'pocketd keys list' to see all available keys"
    fi
fi

ADDRESS=$(prompt_with_default "Signing key name or address (will be the service owner)" "")
while [ -z "$ADDRESS" ]; do
    print_error "Address cannot be empty"
    ADDRESS=$(prompt_with_default "Signing key name or address" "")
done

print_success "Wallet configured"
print_info "Owner Address/Key: $ADDRESS"

# Step 3: Fee Information
print_header "Step 3: Fee Information"

echo "There are two types of fees involved:" > /dev/tty
echo "" > /dev/tty
echo "  1. Transaction fee: Small fee paid to validators (auto-calculated)" > /dev/tty
echo "  2. Add Service fee: Module fee automatically deducted when adding a service" > /dev/tty
echo "" > /dev/tty

# Query and display service params for informational purposes
if command -v pocketd &> /dev/null; then
    echo "Checking current service module parameters..." > /dev/tty
    params_output=$(pocketd query service params --network="$NETWORK" 2>/dev/null || echo "")
    if [ -n "$params_output" ]; then
        echo "" > /dev/tty
        echo -e "${CYAN}Service module parameters:${NC}" > /dev/tty
        echo "$params_output" > /dev/tty
        echo "" > /dev/tty
    fi
fi

print_info "Transaction fees will be auto-calculated using: --gas auto --gas-adjustment 1.5"
print_info "The add_service_fee shown above will be deducted automatically from your account"
echo "" > /dev/tty

read -rp "$(echo -e "${BOLD}Press Enter to continue...${NC}")" < /dev/tty > /dev/tty

# Step 4: Add Services
while true; do
    print_header "Step 4: Manage Services"

    echo "Current services to add: ${#SERVICES[@]}" > /dev/tty
    echo "" > /dev/tty

    options=("Add a new service" "View all services" "Remove a service" "Continue to execution")
    choice=$(select_option "What would you like to do?" "${options[@]}")

    case $choice in
        0) collect_service ;;
        1) display_services ;;
        2) remove_service ;;
        3)
            if [ ${#SERVICES[@]} -eq 0 ]; then
                print_warning "No services have been added. Please add at least one service."
            else
                break
            fi
            ;;
    esac
done

# Step 5: Review and Confirm
print_header "Step 5: Review and Confirm"

echo -e "${BOLD}Configuration Summary:${NC}" > /dev/tty
echo "" > /dev/tty
echo "  Network:        $NETWORK_NAME" > /dev/tty
echo "  Owner/Signer:   $ADDRESS" > /dev/tty
echo "  TX Fee:         auto-calculated (--gas auto --gas-adjustment 1.5)" > /dev/tty
echo "" > /dev/tty

display_services

echo -e "  ${YELLOW}Note: add_service_fee will also be deducted per service (see params above)${NC}" > /dev/tty
echo "" > /dev/tty

if [ "$DRY_RUN" = true ]; then
    print_warning "DRY RUN MODE - Commands will be displayed but not executed"
    echo ""
fi

confirm=$(prompt_yes_no "Do you want to proceed with these service additions?" "n")

if [ "$confirm" = "y" ]; then
    execute_services "$NETWORK" "$ADDRESS"
else
    echo ""
    print_info "Operation cancelled by user."
    exit 0
fi
