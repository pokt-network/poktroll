#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# DEV_NOTE: For testing purposes, you can change the branch name before merging to master.
POCKET_NETWORK_GENESIS_BRANCH="master"

# Snapshot configuration
# The snapshot functionality allows users to quickly sync a node from a trusted snapshot
# instead of syncing from genesis, which can be time-consuming.
# Snapshots are stored at https://snapshots.us-nj.poktroll.com/
# Supports archival snapshots for all networks (testnet-alpha, testnet-beta, mainnet).

# Function to print colored output
print_color() {
    COLOR=$1
    MESSAGE=$2
    echo -e "${COLOR}${MESSAGE}${NC}"
}

# Function to check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        print_color $RED "This script must be run as root or with sudo privileges."
        exit 1
    fi
}

# Function to get and normalize architecture
get_normalized_arch() {
    local arch
    arch=$(uname -m)

    if [ "$arch" = "x86_64" ]; then
        echo "amd64"
    elif [ "$arch" = "aarch64" ] || [ "$arch" = "arm64" ]; then
        echo "arm64"
    else
        print_color $RED "Unsupported architecture: $arch"
        exit 1
    fi
}

check_os() {
    local os
    os=$(uname -s)

    if [ "$os" = "Darwin" ]; then
        print_color $RED "This script is not supported on macOS/Darwin."
        print_color $RED "Please use a Linux distribution."
        exit 1
    fi
}

get_os_type() {
    uname_out="$(uname -s)"

    if [ "$uname_out" = "Linux" ]; then
        echo "linux"
    elif [ "$uname_out" = "Darwin" ]; then
        echo "darwin"
    else
        echo "unsupported"
    fi
}

# Function to check and install dependencies
install_dependencies() {
    local missing_deps=0
    local deps=("jq" "curl" "tar" "wget" "zstd")
    local to_install=()

    print_color $YELLOW "About to start installing dependencies..."

    # Check which dependencies are missing
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &>/dev/null; then
            print_color $YELLOW "$dep is not installed."
            to_install+=("$dep")
            ((missing_deps++))
        else
            print_color $GREEN "$dep is already installed."
        fi
    done

    # If no dependencies are missing, we're done
    if [ $missing_deps -eq 0 ]; then
        print_color $GREEN "All dependencies are already installed."
        return 0
    fi

    # Try to install missing dependencies
    print_color $YELLOW "Installing missing dependencies: ${to_install[*]}"

    if [ -f /etc/debian_version ]; then
        apt-get update
        apt-get install -y "${to_install[@]}"
    elif [ -f /etc/redhat-release ]; then
        yum update -y
        yum install -y "${to_install[@]}"
    else
        print_color $RED "Unsupported distribution. Please install ${to_install[*]} manually."
        return 1
    fi

    # Verify all dependencies were installed successfully
    missing_deps=0
    for dep in "${to_install[@]}"; do
        if ! command -v "$dep" &>/dev/null; then
            print_color $RED "Failed to install $dep"
            ((missing_deps++))
        else
            print_color $GREEN "$dep installed successfully."
        fi
    done

    if [ $missing_deps -gt 0 ]; then
        print_color $RED "Some dependencies failed to install."
        return 1
    fi

    print_color $GREEN "All dependencies installed successfully."
    return 0
}

# Function to get user input
get_user_input() {
    # Ask user which network to install
    echo ""
    echo "Which network would you like to install?"
    echo "1) testnet-alpha (unstable)"
    echo "2) testnet-beta (recommended)"
    echo "3) mainnet (not launched yet)"
    read -p "Enter your choice (1-3): " network_choice

    case $network_choice in
    1) NETWORK="testnet-alpha" ;;
    2) NETWORK="testnet-beta" ;;
    3) NETWORK="mainnet" ;;
    *)
        print_color $RED "Invalid network choice. Exiting."
        exit 1
        ;;
    esac

    print_color $GREEN "Installing the $NETWORK network."
    echo ""

    # Ask if user wants to use a snapshot (for all networks)
    USE_SNAPSHOT=false
    echo "Do you want to sync from:"
    echo "1) Genesis (slower, but verifies the entire chain)"
    echo "2) Snapshot (faster, but requires trusting the snapshot provider)"
    read -p "Enter your choice (1-2): " sync_choice
    
    case $sync_choice in
    2) 
        USE_SNAPSHOT=true
        # Set snapshot base URL
        SNAPSHOT_BASE_URL="https://snapshots.us-nj.poktroll.com"
        
        # Check if the network endpoint exists
        if ! curl --output /dev/null --silent --head --fail "$SNAPSHOT_BASE_URL/$NETWORK-latest-archival.txt"; then
            print_color $RED "No snapshots available for $NETWORK. Falling back to genesis sync."
            print_color $YELLOW "Snapshots may not be provided for all networks, especially new or test networks."
            USE_SNAPSHOT=false
        else
            # Get latest snapshot height
            LATEST_SNAPSHOT_HEIGHT=$(curl -s "$SNAPSHOT_BASE_URL/$NETWORK-latest-archival.txt")
            if [ -z "$LATEST_SNAPSHOT_HEIGHT" ]; then
                print_color $RED "Failed to fetch latest snapshot height for $NETWORK. Falling back to genesis sync."
                print_color $YELLOW "This may happen if snapshots are not yet available for this network."
                USE_SNAPSHOT=false
            else
                print_color $GREEN "Latest snapshot height for $NETWORK: $LATEST_SNAPSHOT_HEIGHT"
                # Get version from snapshot
                # Note: When using a snapshot, we must use the version that the snapshot was created with,
                # not the version from genesis. This is because the snapshot may have been created with
                # a different version than what's in the genesis file.
                SNAPSHOT_VERSION=$(curl -s "$SNAPSHOT_BASE_URL/$NETWORK-$LATEST_SNAPSHOT_HEIGHT-version.txt")
                if [ -z "$SNAPSHOT_VERSION" ]; then
                    print_color $RED "Failed to fetch snapshot version. Falling back to genesis sync."
                    USE_SNAPSHOT=false
                else
                    print_color $GREEN "Snapshot version: $SNAPSHOT_VERSION"
                    
                    # Check if the archival snapshot exists
                    SNAPSHOT_URL="$SNAPSHOT_BASE_URL/$NETWORK-$LATEST_SNAPSHOT_HEIGHT-archival.tar.zst"
                    if curl --output /dev/null --silent --head --fail "$SNAPSHOT_URL"; then
                        print_color $GREEN "Found archival snapshot at: $SNAPSHOT_URL"
                    else
                        # Try alternative format (.tar.gz)
                        SNAPSHOT_URL="$SNAPSHOT_BASE_URL/$NETWORK-$LATEST_SNAPSHOT_HEIGHT-archival.tar.gz"
                        if curl --output /dev/null --silent --head --fail "$SNAPSHOT_URL"; then
                            print_color $GREEN "Found archival snapshot at: $SNAPSHOT_URL"
                            # Set flag for .tar.gz format
                            SNAPSHOT_FORMAT="tar.gz"
                        else
                            print_color $RED "Could not find a valid snapshot for $NETWORK. Falling back to genesis sync."
                            USE_SNAPSHOT=false
                        fi
                    fi
                    
                    if [ "$USE_SNAPSHOT" = true ]; then
                        print_color $YELLOW "Will use snapshot from: $SNAPSHOT_URL"
                    fi
                fi
            fi
        fi
        ;;
    *) 
        USE_SNAPSHOT=false
        print_color $GREEN "Will sync from genesis."
        ;;
    esac

    print_color $YELLOW "(NOTE: If you're on a macOS, enter the name of an existing user)"
    read -p "Enter the desired username to run poktrolld (default: poktroll): " POKTROLL_USER
    POKTROLL_USER=${POKTROLL_USER:-poktroll}

    read -p "Enter the node moniker (default: $(hostname)): " NODE_MONIKER
    NODE_MONIKER=${NODE_MONIKER:-$(hostname)}

    # Update URLs to use the branch constant
    BASE_URL="https://raw.githubusercontent.com/pokt-network/pocket-network-genesis/${POCKET_NETWORK_GENESIS_BRANCH}/shannon/$NETWORK"
    SEEDS_URL="$BASE_URL/seeds"
    GENESIS_URL="$BASE_URL/genesis.json"

    # Download genesis.json and store it
    GENESIS_FILE="/tmp/genesis.json"
    curl -s -o "$GENESIS_FILE" "$GENESIS_URL"
    if [ $? -ne 0 ]; then
        print_color $RED "Failed to download genesis file. Please check your internet connection and try again."
        exit 1
    fi

    # Extract chain_id from genesis.json
    CHAIN_ID=$(jq -r '.chain_id' <"$GENESIS_FILE")
    if [ -z "$CHAIN_ID" ]; then
        print_color $RED "Failed to extract chain_id from genesis file."
        exit 1
    fi
    echo ""
    print_color $GREEN "Using chain_id: $CHAIN_ID from genesis file"

    # Extract version from genesis.json if not using snapshot
    if [ "$USE_SNAPSHOT" = false ]; then
        # When syncing from genesis, we must use the version specified in the genesis file
        # to ensure compatibility with the chain from the beginning.
        POKTROLLD_VERSION=$(jq -r '.app_version' <"$GENESIS_FILE")
        print_color $YELLOW "Detected version from genesis: $POKTROLLD_VERSION"
        if [ -z "$POKTROLLD_VERSION" ]; then
            print_color $RED "Failed to extract version information from genesis file."
            exit 1
        fi
    else
        # When using a snapshot, we must use the version that the snapshot was created with
        # to ensure compatibility with the snapshot data.
        POKTROLLD_VERSION=$SNAPSHOT_VERSION
        print_color $YELLOW "Using version from snapshot: $POKTROLLD_VERSION"
    fi

    # Fetch seeds from the provided URL
    SEEDS=$(curl -s "$SEEDS_URL")
    if [ -z "$SEEDS" ]; then
        print_color $RED "Failed to fetch seeds from $SEEDS_URL. Please check your internet connection and try again."
        exit 1
    fi
    print_color $GREEN "Successfully fetched seeds: $SEEDS"

    # Ask user for confirmation
    read -p "Do you want to use these seeds? (Y/n): " confirm
    if [[ $confirm =~ ^[Nn] ]]; then
        read -p "Enter custom seeds: " custom_seeds
        SEEDS=${custom_seeds:-$SEEDS}
    fi
    echo ""
}

# Function to create user
create_user() {
    if id "$POKTROLL_USER" &>/dev/null; then
        print_color $YELLOW "User $POKTROLL_USER already exists. Skipping user creation."
    else
        useradd -m -s /bin/bash "$POKTROLL_USER"
        print_color $YELLOW "User $POKTROLL_USER created. Please set a password for this user."
        while true; do
            if passwd "$POKTROLL_USER"; then
                break
            else
                print_color $RED "Password change failed. Please try again."
            fi
        done
        usermod -aG sudo "$POKTROLL_USER"
        print_color $GREEN "User $POKTROLL_USER created successfully and added to sudo group."
    fi
}

# TODO_TECHDEBT(@okdas): Use `.poktrollrc` across the board to create a clean
# separation of concerns for pocket specific configurations and debugging.
# Function to set up environment variables
setup_env_vars() {
    print_color $YELLOW "Setting up environment variables..."
    sudo -u "$POKTROLL_USER" bash <<EOF
    echo "export DAEMON_NAME=poktrolld" >> \$HOME/.profile
    echo "export DAEMON_HOME=\$HOME/.poktroll" >> \$HOME/.profile
    echo "export DAEMON_RESTART_AFTER_UPGRADE=true" >> \$HOME/.profile
    echo "export DAEMON_ALLOW_DOWNLOAD_BINARIES=true" >> \$HOME/.profile
    echo "export UNSAFE_SKIP_BACKUP=false" >> \$HOME/.profile
    source \$HOME/.profile
EOF
    print_color $GREEN "Environment variables set up successfully."
    echo ""
}

# Function to download and set up Cosmovisor
setup_cosmovisor() {
    print_color $YELLOW "Setting up Cosmovisor..."

    ARCH=$(get_normalized_arch)
    OS_TYPE=$(get_os_type)

    if [ "$OS_TYPE" = "unsupported" ]; then
        echo "Unsupported OS: $(uname -s)"
        exit 1
    fi

    COSMOVISOR_VERSION="v1.6.0"
    # Note that cosmosorvisor only support linux, which is why OS_TYPE is not used in the URL.
    COSMOVISOR_URL="https://github.com/cosmos/cosmos-sdk/releases/download/cosmovisor%2F${COSMOVISOR_VERSION}/cosmovisor-${COSMOVISOR_VERSION}-linux-${ARCH}.tar.gz"
    print_color $YELLOW "Attempting to download from: $COSMOVISOR_URL"

    sudo -u "$POKTROLL_USER" bash <<EOF
    mkdir -p \$HOME/bin
    curl -L "$COSMOVISOR_URL" | tar -zxvf - -C \$HOME/bin
    echo 'export PATH=\$HOME/bin:\$PATH' >> \$HOME/.profile
    source \$HOME/.profile
EOF
    print_color $GREEN "Cosmovisor set up successfully."
    echo ""
}

# Function to download and set up Poktrolld
setup_poktrolld() {
    print_color $YELLOW "Setting up Poktrolld..."

    ARCH=$(get_normalized_arch)
    OS_TYPE=$(get_os_type)

    # Note: Version is now extracted in get_user_input() function
    # and stored in POKTROLLD_VERSION variable
    print_color $YELLOW "Using poktrolld version: $POKTROLLD_VERSION"

    # TODO_TECHDEBT(@okdas): Consolidate this business logic with what we have
    # in `user_guide/install.md` to avoid duplication and have consistency.

    # Construct the release URL with proper version format
    RELEASE_URL="https://github.com/pokt-network/poktroll/releases/download/v${POKTROLLD_VERSION}/poktroll_${OS_TYPE}_${ARCH}.tar.gz"
    print_color $YELLOW "Attempting to download from: $RELEASE_URL"

    # Download and extract directly as the POKTROLL_USER
    sudo -u "$POKTROLL_USER" bash <<EOF
    mkdir -p \$HOME/.poktroll/cosmovisor/genesis/bin
    mkdir -p \$HOME/.local/bin
    curl -L "$RELEASE_URL" | tar -zxvf - -C \$HOME/.poktroll/cosmovisor/genesis/bin
    if [ \$? -ne 0 ]; then
        echo "Failed to download or extract binary"
        exit 1
    fi
    chmod +x \$HOME/.poktroll/cosmovisor/genesis/bin/poktrolld
    ln -sf \$HOME/.poktroll/cosmovisor/genesis/bin/poktrolld \$HOME/.local/bin/poktrolld
    source \$HOME/.profile
EOF

    if [ $? -ne 0 ]; then
        print_color $RED "Failed to set up Poktrolld"
        exit 1
    fi

    print_color $GREEN "Poktrolld set up successfully."
    echo ""
}

# Function to configure Poktrolld
configure_poktrolld() {
    print_color $YELLOW "Configuring Poktrolld..."

    # Ask for confirmation to use the downloaded genesis file
    print_color $YELLOW "The script has downloaded the genesis file from:"
    print_color $YELLOW "$GENESIS_URL"
    read -p "Are you OK with using this genesis file? (Y/n): " confirm_genesis
    if [[ $confirm_genesis =~ ^[Nn] ]]; then
        print_color $RED "Genesis file usage cancelled. Exiting."
        exit 1
    fi

    # Detect external IP address
    EXTERNAL_IP=$(curl -s https://api.ipify.org)
    print_color $YELLOW "Detected external IP address: $EXTERNAL_IP"
    read -p "Is this your correct external IP address? (Y/n): " confirm_ip
    if [[ $confirm_ip =~ ^[Nn] ]]; then
        read -p "Please enter your external IP address: " custom_ip
        EXTERNAL_IP=${custom_ip:-$EXTERNAL_IP}
    fi
    echo ""

    sudo -u "$POKTROLL_USER" bash <<EOF
    source \$HOME/.profile

    # Check poktrolld version
    POKTROLLD_VERSION=\$(poktrolld version)
    echo "Poktrolld version: \$POKTROLLD_VERSION"

    poktrolld init "$NODE_MONIKER" --chain-id="$CHAIN_ID" --home=\$HOME/.poktroll
    cp "$GENESIS_FILE" \$HOME/.poktroll/config/genesis.json
    sed -i -e "s|^seeds *=.*|seeds = \"$SEEDS\"|" \$HOME/.poktroll/config/config.toml
    sed -i -e "s|^external_address *=.*|external_address = \"$EXTERNAL_IP:26656\"|" \$HOME/.poktroll/config/config.toml
EOF
    if [ $? -eq 0 ]; then
        print_color $GREEN "Poktrolld configured successfully."
    else
        print_color $RED "Failed to configure Poktrolld. Please check the error messages above."
        exit 1
    fi
}

# Function to download and apply snapshot
setup_from_snapshot() {
    print_color $YELLOW "Setting up node from snapshot..."
    print_color $YELLOW "Using snapshot for $NETWORK at height $LATEST_SNAPSHOT_HEIGHT"
    print_color $YELLOW "Snapshot URL: $SNAPSHOT_URL"
    
    # Create a temporary directory for the snapshot
    SNAPSHOT_DIR="/tmp/poktroll_snapshot"
    mkdir -p "$SNAPSHOT_DIR"
    
    # Download and extract the snapshot
    print_color $YELLOW "Downloading and extracting snapshot. This may take a while..."
    print_color $YELLOW "Depending on the network, this could take several minutes to hours."
    print_color $YELLOW "Large snapshots may require significant bandwidth and disk space."
    
    # Note: We're using zstd to decompress the snapshot as it's more efficient than gzip
    # for large files. The archival snapshots can be quite large, so this is important.
    # We extract directly to the data directory to avoid using extra disk space.
    
    # Print start time
    START_TIME=$(date +"%T")
    print_color $YELLOW "Starting snapshot extraction at: $START_TIME"
    
    # Download and extract directly as the POKTROLL_USER
    sudo -u "$POKTROLL_USER" bash <<EOF
    # Stop if any command fails
    set -e
    
    # Create data directory if it doesn't exist
    mkdir -p \$HOME/.poktroll/data
    
    # Download and extract the snapshot based on format
    # The snapshot URL format is determined by the network and height
    # For example: https://snapshots.us-nj.poktroll.com/testnet-beta-66398-archival.tar.zst
    if [[ "$SNAPSHOT_URL" == *.tar.zst ]]; then
        echo "Extracting .tar.zst format snapshot..."
        curl -L "$SNAPSHOT_URL" | zstd -d | tar -xf - -C \$HOME/.poktroll/data
    elif [[ "$SNAPSHOT_URL" == *.tar.gz ]]; then
        echo "Extracting .tar.gz format snapshot..."
        curl -L "$SNAPSHOT_URL" | tar -zxf - -C \$HOME/.poktroll/data
    else
        echo "Unknown snapshot format. Expected .tar.zst or .tar.gz"
        exit 1
    fi
    
    # Check if extraction was successful
    if [ \$? -ne 0 ]; then
        echo "Failed to download or extract snapshot"
        exit 1
    fi
    
    echo "Snapshot extracted successfully"
EOF
    
    # Print end time
    END_TIME=$(date +"%T")
    print_color $YELLOW "Finished snapshot extraction at: $END_TIME"
    
    if [ $? -eq 0 ]; then
        print_color $GREEN "Snapshot for $NETWORK applied successfully."
    else
        print_color $RED "Failed to apply snapshot for $NETWORK. Falling back to genesis sync."
        USE_SNAPSHOT=false
        # Clean up any partial data
        sudo -u "$POKTROLL_USER" bash <<EOF
        rm -rf \$HOME/.poktroll/data/*
EOF
    fi
    
    # Clean up
    rm -rf "$SNAPSHOT_DIR"
    echo ""
}

# TODO_IMPROVE(@okdas): Use the fields from `setup_env_vars` to maintain a single source of truth
# for the values. Specifically, everything starting with `Environment=` is duplicated in the env var helper.
# Function to set up systemd service
setup_systemd() {
    print_color $YELLOW "Setting up systemd service..."
    cat >/etc/systemd/system/cosmovisor.service <<EOF
[Unit]
Description=Cosmovisor daemon for poktrolld
After=network-online.target

[Service]
User=$POKTROLL_USER
ExecStart=/home/$POKTROLL_USER/bin/cosmovisor run start --home=/home/$POKTROLL_USER/.poktroll
Restart=always
RestartSec=3
LimitNOFILE=infinity
LimitNPROC=infinity
Environment="DAEMON_NAME=poktrolld"
Environment="DAEMON_HOME=/home/$POKTROLL_USER/.poktroll"
Environment="DAEMON_RESTART_AFTER_UPGRADE=true"
Environment="DAEMON_ALLOW_DOWNLOAD_BINARIES=true"
Environment="UNSAFE_SKIP_BACKUP=true"

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable cosmovisor.service
    systemctl start cosmovisor.service
    print_color $GREEN "Systemd service set up and started successfully."
}

# Function to check if ufw is installed and open port 26656. We need to open the port to keep the network healthy.
# By default, at least on Debian vultr, this port is not open to the internet.
configure_ufw() {
    if command -v ufw &>/dev/null; then
        print_color $YELLOW "ufw is installed."

        # Check if rule already exists
        if ufw status | grep -q "26656"; then
            print_color $YELLOW "Port 26656 is already allowed in ufw rules."
            return
        fi

        read -p "Do you want to open port 26656 for p2p communication? (Y/n): " open_port
        if [[ $open_port =~ ^[Yy] ]]; then
            ufw allow 26656
            print_color $GREEN "Port 26656 opened successfully."
            print_color $YELLOW "To remove this rule later, run: sudo ufw delete allow 26656"
        else
            print_color $YELLOW "No firewall rules modified."
        fi
    else
        print_color $YELLOW "ufw is not installed. Skipping firewall configuration."
    fi
}

# Main function
main() {
    print_color $GREEN "Welcome to the Poktroll Full Node Install Script!"
    echo ""
    check_os
    check_root
    install_dependencies
    get_user_input
    create_user
    setup_env_vars
    setup_cosmovisor
    setup_poktrolld
    configure_poktrolld
    
    # Apply snapshot if user chose to use it
    if [ "$USE_SNAPSHOT" = true ]; then
        setup_from_snapshot
    fi
    
    setup_systemd
    configure_ufw
    
    # Print completion message with appropriate details
    print_color $GREEN "Poktroll Full Node installation for $NETWORK completed successfully!"
    if [ "$USE_SNAPSHOT" = true ]; then
        print_color $GREEN "Node was set up using snapshot at height $LATEST_SNAPSHOT_HEIGHT with version $SNAPSHOT_VERSION"
        print_color $YELLOW "Note: The node will continue syncing from height $LATEST_SNAPSHOT_HEIGHT to the current chain height"
    else
        print_color $GREEN "Node was set up to sync from genesis with version $POKTROLLD_VERSION"
        print_color $YELLOW "Note: Syncing from genesis may take a significant amount of time"
    fi
    print_color $YELLOW "You can check the status of your node with: sudo systemctl status cosmovisor.service"
    print_color $YELLOW "View logs with: sudo journalctl -u cosmovisor.service -f"
}

main
