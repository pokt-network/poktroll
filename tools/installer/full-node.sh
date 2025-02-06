#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# DEV_NOTE: For testing purposes, you can change the branch name before merging to master.
POCKET_NETWORK_GENESIS_BRANCH="master"

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
    local deps=("jq" "curl" "tar" "wget")
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

    print_color $YELLOW "All dependencies installed successfully."
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
    # COSMOVISOR_URL="https://github.com/cosmos/cosmos-sdk/releases/download/cosmovisor%2F${COSMOVISOR_VERSION}/cosmovisor-${COSMOVISOR_VERSION}-${OS_TYPE}-${ARCH}.tar.gz"
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

    # Extract the version from genesis.json using jq
    POKTROLLD_VERSION=$(jq -r '.app_version' <"$GENESIS_FILE")
    print_color $YELLOW "Detected version from genesis: $POKTROLLD_VERSION"

    if [ -z "$POKTROLLD_VERSION" ]; then
        print_color $RED "Failed to extract version information from genesis file."
        exit 1
    fi

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
    setup_systemd
    configure_ufw
    print_color $GREEN "Poktroll Full Node installation for $NETWORK completed successfully!"
    print_color $YELLOW "You can check the status of your node with: sudo systemctl status cosmovisor.service"
    print_color $YELLOW "View logs with: sudo journalctl -u cosmovisor.service -f"
}

main
