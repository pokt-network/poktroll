#!/bin/bash

# Set error handling
set -e

# Error handling function
handle_error() {
    print_color $RED "An error occurred during installation at line $1"
    exit 1
}

# Set up trap to catch errors
trap 'handle_error $LINENO' ERR

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# DEV_NOTE: For testing purposes, you can change the branch name before merging to master.
POCKET_NETWORK_GENESIS_BRANCH="master"

# Define environment variables for pocketd (single source of truth)
DAEMON_NAME="pocketd"
DAEMON_RESTART_AFTER_UPGRADE="true"
DAEMON_ALLOW_DOWNLOAD_BINARIES="true"
UNSAFE_SKIP_BACKUP="true"

# Snapshot configuration
# The snapshot functionality allows users to quickly sync a node from a trusted snapshot
# instead of syncing from genesis, which can be time-consuming.
# Snapshots are stored at https://snapshots.us-nj.pocket.com/
# Supports archival snapshots for all networks (testnet-alpha, testnet-beta, mainnet).
#
# This script exclusively uses torrent downloads for snapshots:
# - Torrents provide faster, more reliable downloads for large files
# - Distributes bandwidth load across multiple peers
# - All torrents include web seeds, so they'll work even without peers
# - Uses aria2c with optimized settings for maximum download performance

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
    # Temporarily disable exit on error for dependency installation
    set +e
    
    local missing_deps=0
    local deps=("jq" "curl" "tar" "wget" "zstd" "aria2c")
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
        # Re-enable exit on error
        set -e
        return 0
    fi

    # Try to install missing dependencies
    print_color $YELLOW "Installing missing dependencies: ${to_install[*]}"

    # Simple package manager detection and installation
    if command -v apt-get &>/dev/null; then
        print_color $GREEN "Using apt-get to install packages."
        apt-get update
        
        # Install packages
        for dep in "${to_install[@]}"; do
            if [ "$dep" = "aria2c" ]; then
                print_color $YELLOW "Installing aria2 package for aria2c command..."
                apt-get install -y aria2 || print_color $RED "Failed to install aria2. Will try to continue anyway."
            else
                apt-get install -y "$dep" || print_color $RED "Failed to install $dep. Will try to continue anyway."
            fi
        done
    elif command -v yum &>/dev/null; then
        print_color $GREEN "Using yum to install packages."
        yum update -y
        
        # Install packages
        for dep in "${to_install[@]}"; do
            if [ "$dep" = "aria2c" ]; then
                print_color $YELLOW "Installing aria2 package for aria2c command..."
                yum install -y aria2 || print_color $RED "Failed to install aria2. Will try to continue anyway."
            else
                yum install -y "$dep" || print_color $RED "Failed to install $dep. Will try to continue anyway."
            fi
        done
    elif command -v dnf &>/dev/null; then
        print_color $GREEN "Using dnf to install packages."
        dnf check-update || true  # Ignore non-zero exit code from check-update
        
        # Install packages
        for dep in "${to_install[@]}"; do
            if [ "$dep" = "aria2c" ]; then
                print_color $YELLOW "Installing aria2 package for aria2c command..."
                dnf install -y aria2 || print_color $RED "Failed to install aria2. Will try to continue anyway."
            else
                dnf install -y "$dep" || print_color $RED "Failed to install $dep. Will try to continue anyway."
            fi
        done
    else
        print_color $RED "Could not detect a supported package manager (apt-get, yum, or dnf)."
        print_color $RED "Please install the following dependencies manually: ${to_install[*]}"
        print_color $RED "For aria2c, the package name is usually 'aria2'"
        print_color $YELLOW "Continuing with installation, but some features may not work."
    fi

    # Verify all dependencies were installed successfully
    missing_deps=0
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &>/dev/null; then
            print_color $RED "Failed to install $dep"
            ((missing_deps++))
            
            # Special handling for aria2c - provide clear instructions
            if [ "$dep" = "aria2c" ]; then
                print_color $YELLOW "aria2c is required for torrent downloads. You can install it manually with:"
                print_color $YELLOW "  sudo apt-get install aria2    # For Debian/Ubuntu"
                print_color $YELLOW "  sudo yum install aria2        # For RHEL/CentOS"
                print_color $YELLOW "  sudo dnf install aria2        # For Fedora"
            fi
        else
            print_color $GREEN "$dep installed successfully."
        fi
    done

    if [ $missing_deps -gt 0 ]; then
        print_color $RED "Some dependencies failed to install."
        print_color $YELLOW "Continuing with installation, but some features may not work."
    else
        print_color $GREEN "All required dependencies installed successfully."
    fi
    
    # Re-enable exit on error
    set -e
    return 0  # Always return success to continue script execution
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

    # Extract version from genesis.json
    GENESIS_VERSION=$(jq -r '.app_version' <"$GENESIS_FILE")
    print_color $YELLOW "Detected version from genesis: $GENESIS_VERSION"
    if [ -z "$GENESIS_VERSION" ]; then
        print_color $RED "Failed to extract version information from genesis file."
        exit 1
    fi

    # Ask if user wants to use a snapshot (for all networks)
    USE_SNAPSHOT=false
    echo "Do you want to sync from:"
    echo "1) Genesis (slower, but verifies the entire chain)"
    echo "2) Snapshot via torrent (faster, distributed download)"
    read -p "Enter your choice (1-2): " sync_choice
    
    case $sync_choice in
    2) 
        USE_SNAPSHOT=true
        # Set snapshot base URL
        SNAPSHOT_BASE_URL="https://snapshots.us-nj.pocket.com"
        
        # Always use torrent
        USE_TORRENT=true
        print_color $GREEN "Will use torrent for snapshot download (faster and more reliable)."
        
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
                SNAPSHOT_VERSION=$(curl -s "$SNAPSHOT_BASE_URL/$NETWORK-$LATEST_SNAPSHOT_HEIGHT-version.txt")
                if [ -z "$SNAPSHOT_VERSION" ]; then
                    print_color $RED "Failed to fetch snapshot version. Falling back to genesis sync."
                    USE_SNAPSHOT=false
                else
                    print_color $GREEN "Snapshot version: $SNAPSHOT_VERSION"
                    
                    # First try latest torrent
                    TORRENT_URL="$SNAPSHOT_BASE_URL/$NETWORK-latest-archival.torrent"
                    if curl --output /dev/null --silent --head --fail "$TORRENT_URL"; then
                        print_color $GREEN "Found torrent file at: $TORRENT_URL"
                        SNAPSHOT_URL="$TORRENT_URL"
                        # Set the version to use for installation
                        POKTROLLD_VERSION=$SNAPSHOT_VERSION
                        print_color $YELLOW "Will use version $POKTROLLD_VERSION from snapshot"
                    else
                        # Try specific height torrent
                        TORRENT_URL="$SNAPSHOT_BASE_URL/$NETWORK-$LATEST_SNAPSHOT_HEIGHT-archival.torrent"
                        if curl --output /dev/null --silent --head --fail "$TORRENT_URL"; then
                            print_color $GREEN "Found torrent file at: $TORRENT_URL"
                            SNAPSHOT_URL="$TORRENT_URL"
                            # Set the version to use for installation
                            POKTROLLD_VERSION=$SNAPSHOT_VERSION
                            print_color $YELLOW "Will use version $POKTROLLD_VERSION from snapshot"
                        else
                            print_color $RED "Could not find a valid torrent file. Falling back to genesis sync."
                            print_color $YELLOW "This may happen if torrents are not yet available for this network."
                            USE_SNAPSHOT=false
                        fi
                    fi
                    
                    if [ "$USE_SNAPSHOT" = true ]; then
                        print_color $YELLOW "Will use snapshot from: $SNAPSHOT_URL"
                        print_color $GREEN "Using torrent download method with aria2c (faster and more reliable)"
                        print_color $YELLOW "Torrents include web seeds, so they'll work even without peers"
                    fi
                fi
            fi
        fi
        ;;
    *) 
        USE_SNAPSHOT=false
        print_color $GREEN "Will sync from genesis."
        # Set the version to use for installation
        POKTROLLD_VERSION=$GENESIS_VERSION
        print_color $YELLOW "Will use version $POKTROLLD_VERSION from genesis file"
        ;;
    esac

    print_color $YELLOW "(NOTE: If you're on a macOS, enter the name of an existing user)"
    read -p "Enter the desired username to run pocketd (default: pocket): " POKTROLL_USER
    POKTROLL_USER=${POKTROLL_USER:-pocket}

    read -p "Enter the node moniker (default: $(hostname)): " NODE_MONIKER
    NODE_MONIKER=${NODE_MONIKER:-$(hostname)}

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

# TODO_TECHDEBT(@okdas): Use `.pocketrc` across the board to create a clean
# separation of concerns for pocket specific configurations and debugging.
# Function to set up environment variables
setup_env_vars() {
    print_color $YELLOW "Setting up environment variables..."
    
    # Create a .bashrc file if it doesn't exist
    sudo -u "$POKTROLL_USER" bash -c "touch \$HOME/.bashrc"
    
    sudo -u "$POKTROLL_USER" bash <<EOF
    # Add environment variables to both .profile and .bashrc for better compatibility
    echo "export DAEMON_NAME=$DAEMON_NAME" >> \$HOME/.profile
    echo "export DAEMON_HOME=\$HOME/.pocket" >> \$HOME/.profile
    echo "export DAEMON_RESTART_AFTER_UPGRADE=$DAEMON_RESTART_AFTER_UPGRADE" >> \$HOME/.profile
    echo "export DAEMON_ALLOW_DOWNLOAD_BINARIES=$DAEMON_ALLOW_DOWNLOAD_BINARIES" >> \$HOME/.profile
    echo "export UNSAFE_SKIP_BACKUP=$UNSAFE_SKIP_BACKUP" >> \$HOME/.profile
    
    # Add Cosmovisor and pocketd to PATH
    echo "export PATH=\$HOME/.local/bin:\$HOME/.pocket/cosmovisor/current/bin:\$PATH" >> \$HOME/.profile
    
    # Also add to .bashrc to ensure they're available in non-login shells
    echo "export DAEMON_NAME=$DAEMON_NAME" >> \$HOME/.bashrc
    echo "export DAEMON_HOME=\$HOME/.pocket" >> \$HOME/.bashrc
    echo "export DAEMON_RESTART_AFTER_UPGRADE=$DAEMON_RESTART_AFTER_UPGRADE" >> \$HOME/.bashrc
    echo "export DAEMON_ALLOW_DOWNLOAD_BINARIES=$DAEMON_ALLOW_DOWNLOAD_BINARIES" >> \$HOME/.bashrc
    echo "export UNSAFE_SKIP_BACKUP=$UNSAFE_SKIP_BACKUP" >> \$HOME/.bashrc
    
    # Add Cosmovisor and pocketd to PATH in .bashrc as well
    echo "export PATH=\$HOME/.local/bin:\$HOME/.pocket/cosmovisor/current/bin:\$PATH" >> \$HOME/.bashrc
    
    # Source the profile to make variables available in this session
    source \$HOME/.profile
EOF

    # Export variables for the current script session as well
    export DAEMON_NAME=$DAEMON_NAME
    export DAEMON_HOME=/home/$POKTROLL_USER/.pocket
    export DAEMON_RESTART_AFTER_UPGRADE=$DAEMON_RESTART_AFTER_UPGRADE
    export DAEMON_ALLOW_DOWNLOAD_BINARIES=$DAEMON_ALLOW_DOWNLOAD_BINARIES
    export UNSAFE_SKIP_BACKUP=$UNSAFE_SKIP_BACKUP
    
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

    COSMOVISOR_VERSION="v1.7.1"
    # Note that cosmosorvisor only support linux, which is why OS_TYPE is not used in the URL.
    COSMOVISOR_URL="https://github.com/cosmos/cosmos-sdk/releases/download/cosmovisor%2F${COSMOVISOR_VERSION}/cosmovisor-${COSMOVISOR_VERSION}-linux-${ARCH}.tar.gz"
    print_color $YELLOW "Attempting to download from: $COSMOVISOR_URL"

    sudo -u "$POKTROLL_USER" bash <<EOF
    mkdir -p \$HOME/.local/bin
    mkdir -p \$HOME/.pocket/cosmovisor/genesis/bin
    mkdir -p \$HOME/.pocket/cosmovisor/upgrades
    
    curl -L "$COSMOVISOR_URL" | tar -zxvf - -C \$HOME/.local/bin
    chmod +x \$HOME/.local/bin/cosmovisor
    
    # Add to PATH in this session
    export PATH=\$HOME/.local/bin:\$PATH
    
    # Make sure the PATH is updated in .profile
    echo 'export PATH=\$HOME/.local/bin:\$PATH' >> \$HOME/.profile
    
    # Source the profile to make the PATH available in this session
    source \$HOME/.profile
EOF
    print_color $GREEN "Cosmovisor set up successfully."
    echo ""
}

# Function to download and set up Poktrolld
setup_pocketd() {
    print_color $YELLOW "Setting up Poktrolld..."

    ARCH=$(get_normalized_arch)
    OS_TYPE=$(get_os_type)

    # Note: Version is now extracted in get_user_input() function
    # and stored in POKTROLLD_VERSION variable
    print_color $YELLOW "Using pocketd version: $POKTROLLD_VERSION"

    # Construct the release URL with proper version format
    RELEASE_URL="https://github.com/pokt-network/poktroll/releases/download/v${POKTROLLD_VERSION}/pocket_${OS_TYPE}_${ARCH}.tar.gz"
    print_color $YELLOW "Attempting to download from: $RELEASE_URL"

    # Download and extract directly as the POKTROLL_USER
    sudo -u "$POKTROLL_USER" bash <<EOF
    # Ensure directories exist
    mkdir -p \$HOME/.pocket/cosmovisor/genesis/bin
    mkdir -p \$HOME/.pocket/cosmovisor/upgrades
    mkdir -p \$HOME/.local/bin
    
    # Download and extract the binary
    curl -L "$RELEASE_URL" | tar -zxvf - -C \$HOME/.pocket/cosmovisor/genesis/bin
    if [ \$? -ne 0 ]; then
        echo "Failed to download or extract binary"
        exit 1
    fi
    chmod +x \$HOME/.pocket/cosmovisor/genesis/bin/pocketd
    
    # Create the current symlink manually to ensure it exists
    ln -sf \$HOME/.pocket/cosmovisor/genesis \$HOME/.pocket/cosmovisor/current
    
    # Create a symlink to the binary in .local/bin for easier access
    ln -sf \$HOME/.pocket/cosmovisor/current/bin/pocketd \$HOME/.local/bin/pocketd
    
    # Initialize Cosmovisor with the pocketd binary
    export DAEMON_NAME=$DAEMON_NAME
    export DAEMON_HOME=\$HOME/.pocket
    \$HOME/.local/bin/cosmovisor init \$HOME/.pocket/cosmovisor/genesis/bin/pocketd
    
    # Source the profile to update the environment
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
configure_pocketd() {
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

    # Check pocketd version
    # Now we can use pocketd directly since Cosmovisor is initialized
    POKTROLLD_VERSION=\$(\$HOME/.pocket/cosmovisor/genesis/bin/pocketd version)
    echo "Poktrolld version: \$POKTROLLD_VERSION"

    # Initialize node using pocketd directly
    \$HOME/.pocket/cosmovisor/genesis/bin/pocketd init "$NODE_MONIKER" --chain-id="$CHAIN_ID" --home=\$HOME/.pocket
    cp "$GENESIS_FILE" \$HOME/.pocket/config/genesis.json
    sed -i -e "s|^seeds *=.*|seeds = \"$SEEDS\"|" \$HOME/.pocket/config/config.toml
    sed -i -e "s|^external_address *=.*|external_address = \"$EXTERNAL_IP:26656\"|" \$HOME/.pocket/config/config.toml
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
    
    # Create a temporary directory for the snapshot in the user's home directory
    SNAPSHOT_DIR="/home/$POKTROLL_USER/pocket_snapshot"
    sudo -u "$POKTROLL_USER" mkdir -p "$SNAPSHOT_DIR"
    
    # Download and extract the snapshot
    print_color $YELLOW "Downloading and extracting snapshot. This may take a while..."
    print_color $YELLOW "Depending on the network, this could take several minutes to hours."
    print_color $YELLOW "Large snapshots may require significant bandwidth and disk space."
    print_color $YELLOW "Torrent downloads use web seeds, so they'll work even without peers."
    
    # Print start time
    START_TIME=$(date +"%T")
    print_color $YELLOW "Starting snapshot download at: $START_TIME"
    
    # Download the torrent file
    TORRENT_FILE="$SNAPSHOT_DIR/snapshot.torrent"
    sudo -u "$POKTROLL_USER" curl -L -o "$TORRENT_FILE" "$SNAPSHOT_URL"
    
    if [ $? -ne 0 ]; then
        print_color $RED "Failed to download torrent file. Falling back to genesis sync."
        USE_SNAPSHOT=false
    else
        print_color $GREEN "Torrent file downloaded successfully."
        
        # Set the download directory
        DOWNLOAD_DIR="$SNAPSHOT_DIR/download"
        sudo -u "$POKTROLL_USER" mkdir -p "$DOWNLOAD_DIR"
        
        # Use aria2c to download the snapshot
        print_color $YELLOW "Starting torrent download with aria2c. This may take a while..."
        print_color $YELLOW "Download progress will be shown below:"
        
        # Run aria2c as the POKTROLL_USER
        sudo -u "$POKTROLL_USER" bash <<EOF
        # Stop if any command fails
        set -e
        
        # Create data directory if it doesn't exist
        mkdir -p \$HOME/.pocket/data
        
        # Download using aria2c with optimized settings
        # --seed-time=0: Don't seed after download completes
        # --file-allocation=none: Faster startup
        # --continue=true: Resume download if interrupted
        # --max-connection-per-server=4: Limit connections to web server to reduce load
        # --max-concurrent-downloads=16: Download multiple pieces simultaneously
        # --split=16: Split file into more segments for parallel download
        # --bt-enable-lpd=true: Enable Local Peer Discovery
        # --bt-max-peers=100: High number of peers for better distribution
        # --bt-prioritize-piece=head,tail: Download beginning and end first for verification
        # --bt-seed-unverified: Seed without verifying to help the network
        aria2c --seed-time=0 --dir="$DOWNLOAD_DIR" --file-allocation=none --continue=true \
               --max-connection-per-server=4 --max-concurrent-downloads=16 --split=16 \
               --bt-enable-lpd=true --bt-max-peers=100 --bt-prioritize-piece=head,tail \
               --bt-seed-unverified \
               "$TORRENT_FILE"
        
        # Check if download was successful
        if [ \$? -ne 0 ]; then
            echo "Failed to download snapshot via torrent"
            exit 1
        fi
        
        # Find the downloaded file
        DOWNLOADED_FILE=\$(find "$DOWNLOAD_DIR" -type f | head -n 1)
        
        if [ -z "\$DOWNLOADED_FILE" ]; then
            echo "No files downloaded by aria2c"
            exit 1
        fi
        
        echo "Downloaded file: \$DOWNLOADED_FILE"
        
        # Extract the snapshot based on format
        if [[ "\$DOWNLOADED_FILE" == *.tar.zst ]]; then
            echo "Extracting .tar.zst format snapshot..."
            zstd -d "\$DOWNLOADED_FILE" --stdout | tar -xf - -C \$HOME/.pocket/data
        elif [[ "\$DOWNLOADED_FILE" == *.tar.gz ]]; then
            echo "Extracting .tar.gz format snapshot..."
            tar -zxf "\$DOWNLOADED_FILE" -C \$HOME/.pocket/data
        else
            echo "Unknown snapshot format. Expected .tar.zst or .tar.gz"
            exit 1
        fi
        
        echo "Snapshot extracted successfully"
EOF
    fi
    
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
        rm -rf \$HOME/.pocket/data/*
EOF
    fi
    
    # Clean up
    print_color $YELLOW "Cleaning up temporary snapshot files..."
    sudo -u "$POKTROLL_USER" rm -rf "$SNAPSHOT_DIR"
    print_color $GREEN "Cleanup completed."
    echo ""
}

# Function to set up systemd service
setup_systemd() {
    # Create a unique service name based on user
    SERVICE_NAME="cosmovisor-${POKTROLL_USER}"
    print_color $YELLOW "Setting up systemd service as $SERVICE_NAME.service..."
    
    cat >/etc/systemd/system/$SERVICE_NAME.service <<EOF
[Unit]
Description=Cosmovisor daemon for pocketd ($POKTROLL_USER)
After=network-online.target

[Service]
User=$POKTROLL_USER
ExecStart=/home/$POKTROLL_USER/.local/bin/cosmovisor run start --home=/home/$POKTROLL_USER/.pocket
Restart=always
RestartSec=3
LimitNOFILE=infinity
LimitNPROC=infinity
Environment="DAEMON_NAME=$DAEMON_NAME"
Environment="DAEMON_HOME=/home/$POKTROLL_USER/.pocket"
Environment="DAEMON_RESTART_AFTER_UPGRADE=$DAEMON_RESTART_AFTER_UPGRADE"
Environment="DAEMON_ALLOW_DOWNLOAD_BINARIES=$DAEMON_ALLOW_DOWNLOAD_BINARIES"
Environment="UNSAFE_SKIP_BACKUP=$UNSAFE_SKIP_BACKUP"

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable $SERVICE_NAME.service
    systemctl start $SERVICE_NAME.service
    print_color $GREEN "Systemd service $SERVICE_NAME.service set up and started successfully."
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
    
    # Basic checks
    check_os
    check_root
    
    # Install dependencies - temporarily disable error trapping for this function
    trap - ERR
    install_dependencies
    # Restore error trapping
    trap 'handle_error $LINENO' ERR
    
    # Continue with installation
    get_user_input  # This now includes determining the correct pocketd version
    create_user
    setup_env_vars
    setup_cosmovisor
    setup_pocketd  # Now installs the correct version determined in get_user_input
    configure_pocketd
    
    # Apply snapshot if user chose to use it
    if [ "$USE_SNAPSHOT" = true ]; then
        # Temporarily disable error trapping for snapshot setup
        trap - ERR
        setup_from_snapshot
        # Restore error trapping
        trap 'handle_error $LINENO' ERR
    fi
    
    setup_systemd
    configure_ufw
    
    # Print completion message with appropriate details
    print_color $GREEN "Poktroll Full Node installation for $NETWORK completed successfully!"
    if [ "$USE_SNAPSHOT" = true ]; then
        print_color $GREEN "Node was set up using snapshot at height $LATEST_SNAPSHOT_HEIGHT with version $POKTROLLD_VERSION"
        print_color $YELLOW "Note: The node will continue syncing from height $LATEST_SNAPSHOT_HEIGHT to the current chain height"
    else
        print_color $GREEN "Node was set up to sync from genesis with version $POKTROLLD_VERSION"
        print_color $YELLOW "Note: Syncing from genesis may take a significant amount of time"
    fi
    print_color $YELLOW "You can check the status of your node with: sudo systemctl status $SERVICE_NAME.service"
    print_color $YELLOW "View logs with: sudo journalctl -u $SERVICE_NAME.service -f"
}

main
