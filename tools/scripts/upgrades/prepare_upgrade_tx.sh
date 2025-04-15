#!/bin/bash

# Script to generate software upgrade proposal transactions for different environments.
#
# Usage: ./prepare_upgrade_tx.sh <version> [options]
#   <version>: Required. The release version tag (e.g., v0.1.2).
#   [options]: Optional flags to skip checksums for specific environments.
#              --no-checksum-alpha: Skip checksum for alpha environment.
#              --no-checksum-beta:  Skip checksum for beta environment.
#              --no-checksum-local: Skip checksum for local environment.
#              --no-checksum-main:  Skip checksum for main environment.
#
# Rationale for optional checksums:
# Omitting checksums (e.g., for Alpha network) provides flexibility.
# It allows the possibility of replacing the release binary *after* the upgrade
# height is reached if unforeseen issues arise with the initial binary.
# While checksums enhance security, this option prioritizes rapid iteration
# and recovery during early network phases or testing.

# Check if version input is provided
if [ -z "$1" ] || [[ "$1" == --* ]]; then
  echo "Error: Version parameter is required as the first argument (e.g., v0.1.2)" >&2
  exit 1
fi

VERSION="$1"
shift # Remove version from arguments

RELEASE_URL="https://github.com/pokt-network/poktroll/releases/download/$VERSION"
CHECKSUM_URL="$RELEASE_URL/release_checksum"
OUTPUT_DIR="tools/scripts/upgrades"

# Default checksum inclusion to true
INCLUDE_CHECKSUM_ALPHA=true
INCLUDE_CHECKSUM_BETA=true
INCLUDE_CHECKSUM_LOCAL=true
INCLUDE_CHECKSUM_MAIN=true

# Parse optional arguments for skipping checksums
while [[ "$#" -gt 0 ]]; do
    case $1 in
        --no-checksum-alpha) INCLUDE_CHECKSUM_ALPHA=false ;;
        --no-checksum-beta) INCLUDE_CHECKSUM_BETA=false ;;
        --no-checksum-local) INCLUDE_CHECKSUM_LOCAL=false ;;
        --no-checksum-main) INCLUDE_CHECKSUM_MAIN=false ;;
        *) echo "Unknown parameter passed: $1"; exit 1 ;;
    esac
    shift
done

# Create output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

# Create a temporary file for the checksum
TEMP_CHECKSUM=$(mktemp)

# Download checksum file with wget
echo "Downloading checksum file from $CHECKSUM_URL..."
wget -q -O "$TEMP_CHECKSUM" "$CHECKSUM_URL"

if [ ! -s "$TEMP_CHECKSUM" ]; then
  echo "Error: Failed to download checksum file" >&2
  rm -f "$TEMP_CHECKSUM"
  exit 1
fi

# Read the checksums file
CHECKSUMS=$(cat "$TEMP_CHECKSUM")
rm -f "$TEMP_CHECKSUM"

# Extract checksums with correct filenames
LINUX_AMD64_CHECKSUM=$(echo "$CHECKSUMS" | grep "pocket_linux_amd64.tar.gz" | awk '{print $1}')
LINUX_ARM64_CHECKSUM=$(echo "$CHECKSUMS" | grep "pocket_linux_arm64.tar.gz" | awk '{print $1}')
DARWIN_AMD64_CHECKSUM=$(echo "$CHECKSUMS" | grep "pocket_darwin_amd64.tar.gz" | awk '{print $1}')
DARWIN_ARM64_CHECKSUM=$(echo "$CHECKSUMS" | grep "pocket_darwin_arm64.tar.gz" | awk '{print $1}')

# Check if any checksum is missing
if [ -z "$LINUX_AMD64_CHECKSUM" ] || [ -z "$LINUX_ARM64_CHECKSUM" ] ||
  [ -z "$DARWIN_AMD64_CHECKSUM" ] || [ -z "$DARWIN_ARM64_CHECKSUM" ]; then
  echo "Error: Missing checksums in file" >&2
  echo "Available checksums:"
  echo "$CHECKSUMS"
  exit 1
fi

# Define authorities for each environment
ALPHA_AUTHORITY="pokt1r6ja6rz6rpae58njfrsgs5n5sp3r36r2q9j04h"
BETA_AUTHORITY="pokt1f0c9y7mahf2ya8tymy8g4rr75ezh3pkklu4c3e"
LOCAL_AUTHORITY="pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw"
MAIN_AUTHORITY="pokt18808wvw0h4t450t06uvauny8lvscsxjfyua7vh"

# Function to generate JSON file for a specific environment
generate_json_file() {
  local env=$1
  local authority=$2
  local output_file="$OUTPUT_DIR/upgrade_tx_${VERSION}_${env}.json"
  local include_checksum_var="INCLUDE_CHECKSUM_$(echo "$env" | tr '[:lower:]' '[:upper:]')"
  local include_checksum=${!include_checksum_var} # Indirect variable reference

  local linux_amd64_url="$RELEASE_URL/pocket_linux_amd64.tar.gz"
  local linux_arm64_url="$RELEASE_URL/pocket_linux_arm64.tar.gz"
  local darwin_amd64_url="$RELEASE_URL/pocket_darwin_amd64.tar.gz"
  local darwin_arm64_url="$RELEASE_URL/pocket_darwin_arm64.tar.gz"

  local checksum_message=""
  if [ "$include_checksum" = true ]; then
    linux_amd64_url+="?checksum=sha256:$LINUX_AMD64_CHECKSUM"
    linux_arm64_url+="?checksum=sha256:$LINUX_ARM64_CHECKSUM"
    darwin_amd64_url+="?checksum=sha256:$DARWIN_AMD64_CHECKSUM"
    darwin_arm64_url+="?checksum=sha256:$DARWIN_ARM64_CHECKSUM"
    checksum_message="including checksums"
  else
    checksum_message="omitting checksums (allows binary replacement post-upgrade if needed)"
  fi

  # Escape slashes for sed
  local escaped_info
  escaped_info=$(printf '{"binaries":{"linux/amd64":"%s","linux/arm64":"%s","darwin/amd64":"%s","darwin/arm64":"%s"}}' \
    "$linux_amd64_url" "$linux_arm64_url" "$darwin_amd64_url" "$darwin_arm64_url" | sed 's/"/\\"/g')

  cat >"$output_file" <<EOF
{
  "body": {
    "messages": [
      {
        "@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
        "authority": "$authority",
        "plan": {
          "name": "$VERSION",
          "height": "UPDATE_ME",
          "info": "$escaped_info"
        }
      }
    ]
  }
}
EOF

  echo "Created $output_file for $env environment, $checksum_message."
}

# Generate JSON files for each environment
generate_json_file "alpha" "$ALPHA_AUTHORITY"
generate_json_file "beta" "$BETA_AUTHORITY"
generate_json_file "local" "$LOCAL_AUTHORITY"
generate_json_file "main" "$MAIN_AUTHORITY"

echo "All upgrade transaction files created successfully."
