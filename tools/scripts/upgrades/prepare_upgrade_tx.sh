#!/bin/bash

# Check if version input is provided
if [ -z "$1" ]; then
  echo "Error: Version parameter is required (e.g., v0.1.2)" >&2
  exit 1
fi

VERSION="$1"
RELEASE_URL="https://github.com/pokt-network/poktroll/releases/download/$VERSION"
CHECKSUM_URL="$RELEASE_URL/release_checksum"
OUTPUT_DIR="tools/scripts/upgrades"

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
          "info": "{\"binaries\":{\"linux/amd64\":\"$RELEASE_URL/pocket_linux_amd64.tar.gz?checksum=sha256:$LINUX_AMD64_CHECKSUM\",\"linux/arm64\":\"$RELEASE_URL/pocket_linux_arm64.tar.gz?checksum=sha256:$LINUX_ARM64_CHECKSUM\",\"darwin/amd64\":\"$RELEASE_URL/pocket_darwin_amd64.tar.gz?checksum=sha256:$DARWIN_AMD64_CHECKSUM\",\"darwin/arm64\":\"$RELEASE_URL/pocket_darwin_arm64.tar.gz?checksum=sha256:$DARWIN_ARM64_CHECKSUM\"}}"
        }
      }
    ]
  }
}
EOF

  echo "Created $output_file"
}

# Generate JSON files for each environment
generate_json_file "alpha" "$ALPHA_AUTHORITY"
generate_json_file "beta" "$BETA_AUTHORITY"
generate_json_file "local" "$LOCAL_AUTHORITY"
generate_json_file "main" "$MAIN_AUTHORITY"

echo "All upgrade transaction files created successfully."
