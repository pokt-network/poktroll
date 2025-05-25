#!/usr/bin/env bash

# This script installs the pocketd binary if not already installed.
# Use with -u or --upgrade flag to force reinstallation of the latest version by removing the existing binary first.

# Example Usage:
# curl -sSL https://raw.githubusercontent.com/pokt-network/poktroll/main/tools/scripts/pocketd-install.sh | bash
# curl -sSL https://raw.githubusercontent.com/pokt-network/poktroll/main/tools/scripts/pocketd-install.sh | bash -s -- --upgrade

UPGRADE=false

# Process command line arguments
while [[ "$#" -gt 0 ]]; do
    case $1 in
    -u | --upgrade) UPGRADE=true ;;
    *)
        echo "Unknown parameter: $1"
        exit 1
        ;;
    esac
    shift
done

command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to install pocketd if not present
# This function checks if pocketd is installed. If not, it downloads the correct binary for the system's OS and architecture,
# extracts it, makes it executable, and verifies with 'pocketd version'.
install_pocketd() {
    echo "🛠️  Starting pocketd installation script..."

    if command_exists pocketd && [ "$UPGRADE" = false ]; then
        echo "✅ pocketd already installed. To upgrade to the latest version, use the --upgrade flag."
    else
        if command_exists pocketd && [ "$UPGRADE" = true ]; then
            echo "🔄 Upgrading pocketd..."
            sudo rm -f $(which pocketd)
        else
            echo "🚀 Installing pocketd..."
        fi

        # Detect OS (darwin for macOS, linux for Linux)
        OS=$(uname | tr '[:upper:]' '[:lower:]')

        # Detect architecture and convert to expected format (amd64 or arm64)
        if [ "$(uname -m)" == "x86_64" ]; then
            ARCH="amd64"
        elif [ "$(uname -m)" == "aarch64" ] || [ "$(uname -m)" == "arm64" ]; then
            ARCH="arm64"
        else
            echo "❌ Unsupported architecture: $(uname -m). Expected x86_64, aarch64, or arm64."
            exit 1
        fi

        # Validate OS
        if [ "$OS" != "darwin" ] && [ "$OS" != "linux" ]; then
            echo "❌ Unsupported operating system: $OS. Expected darwin or linux."
            exit 1
        fi

        # Construct tarball name based on detected OS and architecture
        TARBALL="pocket_${OS}_${ARCH}.tar.gz"

        echo "🔍 Detected OS: $OS, Architecture: $ARCH"
        echo "📥 Downloading $TARBALL..."

        # Download the appropriate tarball
        curl -LO "https://github.com/pokt-network/poktroll/releases/latest/download/${TARBALL}"

        # Create directory for binary if it doesn't exist
        sudo mkdir -p /usr/local/bin

        # Extract the tarball to /usr/local/bin
        echo "📦 Extracting files..."
        sudo tar -zxf "${TARBALL}" -C /usr/local/bin

        # Make the binary executable
        sudo chmod +x /usr/local/bin/pocketd

        # Clean up the downloaded tarball
        rm "${TARBALL}"

        echo "🌿 Successfully installed pocketd version:"
        echo "$(pocketd version)"
    fi
}

install_pocketd
