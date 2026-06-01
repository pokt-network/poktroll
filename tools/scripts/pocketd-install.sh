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
# SECURITY: This script is piped from the internet into your shell. To inspect it
# before running, download it first and read it:
#
#     curl -sSL https://raw.githubusercontent.com/pokt-network/poktroll/main/tools/scripts/pocketd-install.sh -o pocketd-install.sh
#     less pocketd-install.sh   # review, then:
#     bash pocketd-install.sh
#
# The downloaded release tarball is verified against the published `release_checksum`
# (SHA256) before installation; a checksum mismatch aborts the install.
#
# SCOPE: This installs ONLY the `pocketd` CLI binary into /usr/local/bin. It does
# NOT set up a full node, Cosmovisor, or a systemd service. To run a full node, see:
#     https://dev.poktroll.com/operate/cheat_sheets/full_node_cheatsheet
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

# Function to verify a downloaded tarball against the published SHA256 checksum.
#
# - Downloads the `release_checksum` file (standard sha256sum format) from the same release.
# - If the checksum file (or a matching entry) is absent (e.g. older releases predating
#   published checksums), warns and continues.
# - If an entry exists but does NOT match, aborts the installation.
verify_checksum() {
    local tarball="$1"
    local checksum_file="release_checksum"

    echo "🔐 Verifying download against published SHA256 checksum..."

    if ! curl -sSfLO "${BASE_URL}/${checksum_file}" 2>/dev/null; then
        echo "⚠️  Could not download ${checksum_file} from ${BASE_URL}."
        echo "    Skipping checksum verification (this release may predate published checksums)."
        return 0
    fi

    local expected
    expected=$(awk -v f="${tarball}" '$2 == f {print $1}' "${checksum_file}")

    if [ -z "${expected}" ]; then
        echo "⚠️  No checksum entry for ${tarball} in ${checksum_file}; skipping verification."
        rm -f "${checksum_file}"
        return 0
    fi

    local actual
    if command_exists sha256sum; then
        actual=$(sha256sum "${tarball}" | awk '{print $1}')
    elif command_exists shasum; then
        actual=$(shasum -a 256 "${tarball}" | awk '{print $1}')
    else
        echo "❌ Neither sha256sum nor shasum is available; cannot verify checksum. Aborting."
        rm -f "${checksum_file}" "${tarball}"
        exit 1
    fi

    if [ "${expected}" != "${actual}" ]; then
        echo "❌ Checksum verification FAILED for ${tarball}"
        echo "    expected: ${expected}"
        echo "    actual:   ${actual}"
        echo "    Refusing to install a tarball that does not match the published checksum."
        rm -f "${checksum_file}" "${tarball}"
        exit 1
    fi

    echo "✅ Checksum verified: ${actual}"
    rm -f "${checksum_file}"
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
        echo "  curl -sSL https://raw.githubusercontent.com/pokt-network/poktroll/main/tools/scripts/pocketd-install.sh | bash -s -- --tag v0.1.12-dev1 --upgrade"
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

    # Verify the downloaded tarball against the published SHA256 checksum before installing.
    verify_checksum "${TARBALL}"

    # Create directory for binary if it doesn't exist
    sudo mkdir -p /usr/local/bin

    # Extract the tarball to /usr/local/bin
    echo "📦 Extracting files..."
    sudo tar -zxf "${TARBALL}" -C /usr/local/bin

    # Make the binary executable
    sudo chmod +x /usr/local/bin/pocketd

    # Clean up the downloaded tarball
    rm "${TARBALL}"

    echo "🌿 Successfully installed the pocketd CLI version:"
    pocketd version
    echo ""
    echo "ℹ️  This installed the pocketd CLI binary ONLY."
    echo "   It did NOT set up a full node, Cosmovisor, or a systemd service."
    echo "   To run a full node, follow: https://dev.poktroll.com/operate/cheat_sheets/full_node_cheatsheet"
}

install_pocketd
