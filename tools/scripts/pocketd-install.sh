#!/usr/bin/env bash

# This script installs the pocketd binary if not already installed.
#
# Usage:
#
# - To install the latest release:
#     curl -sSL https://raw.githubusercontent.com/pokt-network/poktroll/main/tools/scripts/pocketd-install.sh | bash
#
# - If pocketd is already installed and you want to upgrade to the latest version, use:
#     curl -sSL https://raw.githubusercontent.com/pokt-network/poktroll/main/tools/scripts/pocketd-install.sh | bash -s -- --upgrade
#
# - To install a specific release (including dev-releases), use the --tag flag. For example:
#     curl -sSL https://raw.githubusercontent.com/pokt-network/poktroll/main/tools/scripts/pocketd-install.sh | bash -s -- --tag v0.1.12-dev1 --upgrade
#
#   (You can find available versions, including dev-releases, at https://github.com/pokt-network/poktroll/releases)
#
# Flags:
#   -u, --upgrade   Force reinstallation of the latest (or specified) version by removing the existing binary first.
#   --tag <tag>     Install a specific release version (e.g., v0.1.12-dev1).

UPGRADE=false
TAG=""

# Process command line arguments
while [[ "$#" -gt 0 ]]; do
    case $1 in
    -u | --upgrade)
        UPGRADE=true
        ;;
    --tag)
        if [[ -n $2 && $2 != -* ]]; then
            TAG="$2"
            shift
        else
            echo "❌ Error: --tag requires a value."
            exit 1
        fi
        ;;
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
#
# - Checks if pocketd is installed; if not, downloads the correct binary for the system's OS and architecture
# - Extracts it, makes it executable, and verifies installation with 'pocketd version'
install_pocketd() {
    echo "🛠️  Starting pocketd installation script..."

    if command_exists pocketd && [ "$UPGRADE" = false ]; then
        echo "✅ pocketd already installed. To upgrade to the latest version, use the --upgrade flag like so:"
        echo ""
        echo "  curl -sSL https://raw.githubusercontent.com/pokt-network/poktroll/main/tools/scripts/pocketd-install.sh | bash -s -- --upgrade"
        echo ""
        echo "You can also install a specific version by using the --tag flag like so:"
        echo ""
        echo "  curl -sSL https://raw.githubusercontent.com/pokt-network/poktroll/main/tools/scripts/pocketd-install.sh | bash -s -- --tag v0.1.12-dev1"
        echo ""
        echo "You can find available versions, including dev-releases, at https://github.com/pokt-network/poktroll/releases"
        return
    fi

    if command_exists pocketd && [ "$UPGRADE" = true ]; then
        echo "🔄 Upgrading pocketd..."
        sudo rm -f "$(which pocketd)"
    else
        echo "🚀 Installing pocketd..."
    fi

    # Detect OS (darwin for macOS, linux for Linux)
    OS=$(uname | tr '[:upper:]' '[:lower:]')

    # Detect architecture and convert to expected format (amd64 or arm64)
    case "$(uname -m)" in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64 | arm64)
        ARCH="arm64"
        ;;
    *)
        echo "❌ Unsupported architecture: $(uname -m). Expected x86_64, aarch64, or arm64."
        exit 1
        ;;
    esac

    # Validate OS
    if [ "$OS" != "darwin" ] && [ "$OS" != "linux" ]; then
        echo "❌ Unsupported operating system: $OS. Expected darwin or linux."
        exit 1
    fi

    # Construct tarball name based on detected OS and architecture
    TARBALL="pocket_${OS}_${ARCH}.tar.gz"

    # Determine download base URL
    if [[ -n "$TAG" ]]; then
        BASE_URL="https://github.com/pokt-network/poktroll/releases/download/${TAG}"
        echo "🔖 Using tag: $TAG"
    else
        BASE_URL="https://github.com/pokt-network/poktroll/releases/latest/download"
        echo "🔖 Using latest release"
    fi

    echo "🔍 Detected OS: $OS, Architecture: $ARCH"
    echo "📥 Downloading ${BASE_URL}/${TARBALL}"

    # Download the appropriate tarball
    curl -LO "${BASE_URL}/${TARBALL}"

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
    pocketd version
}

install_pocketd
