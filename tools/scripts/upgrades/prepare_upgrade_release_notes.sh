#!/bin/bash

# Protocol Release Notes Script
# Generates a table for the GitHub release by querying onchain data.

set -eo pipefail

# Script metadata
SCRIPT_NAME="$(basename "$0")"

# Default values
UPGRADE_VERSION=""
OUTPUT_FORMAT="markdown"
VERBOSE=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Network helper functions (avoiding associative arrays for bash 3.2 compatibility)
get_network_name() {
    case "$1" in
    alpha) echo "Alpha TestNet" ;;
    beta) echo "Beta TestNet" ;;
    main) echo "MainNet" ;;
    *) echo "Unknown" ;;
    esac
}

# Usage function
usage() {
    cat <<EOF
$SCRIPT_NAME - Protocol Upgrade Query Tool

DESCRIPTION:
    Prepare release notes for a specific upgrade version.
    Specifically, it:
    1. Accepts an upgrade version (e.g., v0.1.18)
    2. Queries for the upgrade height across our alpha, beta, and main networks
    3. Queries for the tx hash of each upgrade
    4. Prepares a copy-pasta table that can be put in the GitHub release notes

USAGE:
    $SCRIPT_NAME [OPTIONS] <upgrade_version>

ARGUMENTS:
    upgrade_version    The upgrade version to query (e.g., v0.1.18)

OPTIONS:
    -h, --help         Show this help message and exit
    -v, --verbose      Enable verbose output
    --version          Show script version
    --output FORMAT    Output format: json, markdown (default: markdown)

EXAMPLES:
    $SCRIPT_NAME v0.1.18
    $SCRIPT_NAME v0.1.18 --verbose --output json

NETWORKS:
    The script queries the following networks:
    - alpha (Alpha TestNet)
    - beta (Beta TestNet)
    - main (MainNet)

DEPENDENCIES:
    - pocketd command must be available in PATH
    - jq command for JSON parsing

EOF
}

# Logging functions
log_info() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "${BLUE}[INFO]${NC} $1" >&2
    fi
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

log_success() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "${GREEN}[SUCCESS]${NC} $1" >&2
    fi
}

# Check dependencies
check_dependencies() {
    local missing_deps=""

    if ! command -v pocketd &>/dev/null; then
        missing_deps="$missing_deps pocketd"
    fi

    if ! command -v jq &>/dev/null; then
        missing_deps="$missing_deps jq"
    fi

    if [[ -n "$missing_deps" ]]; then
        log_error "Missing required dependencies:$missing_deps"
        log_error "Please install the missing dependencies and try again."
        exit 1
    fi
}

# Query upgrade height for a specific network
query_network_upgrade() {
    local network="$1"
    local version="$2"

    log_info "Querying $network network for upgrade $version..."

    local cmd="pocketd query upgrade applied $version --network=$network -o json"

    if [[ "$VERBOSE" == "true" ]]; then
        log_info "Executing: $cmd"
    fi

    # Execute command and capture both stdout and stderr
    local result
    local exit_code=0

    result=$(eval "$cmd" 2>&1) || exit_code=$?

    if [[ $exit_code -eq 0 ]]; then
        # Try to parse as JSON and extract height
        local height
        height=$(echo "$result" | jq -r '.height // "N/A"' 2>/dev/null)

        if [[ "$height" != "N/A" && "$height" != "null" ]]; then
            log_success "Found upgrade height $height for $network"
            echo "$height"
        else
            log_warn "No height found in response for $network"
            echo "N/A"
        fi
    else
        log_warn "Failed to query $network: $result"
        echo "ERROR"
    fi
}

# Query block hash for a specific network and height
query_upgrade_block_hash() {
    local network="$1"
    local height="$2"
    local home_dir="${HOME}/.pocket_prod"

    # Skip if height is not valid
    if [[ "$height" == "N/A" || "$height" == "ERROR" ]]; then
        echo "N/A"
        return
    fi

    log_info "Querying block hash for upgrade at height $height on $network..."

    local cmd="pocketd query block --type=height $height --network=$network -o json --home=\"$home_dir\""

    if [[ "$VERBOSE" == "true" ]]; then
        log_info "Executing: $cmd"
    fi

    # Execute command and capture both stdout and stderr
    local result
    local exit_code=0

    result=$(eval "$cmd" 2>&1) || exit_code=$?

    if [[ $exit_code -eq 0 ]]; then
        # Parse JSON and extract block hash
        local block_hash
        # Try multiple possible locations for the block hash
        block_hash=$(echo "$result" | jq -r '.block_id.hash // .block.block_id.hash // .header.hash // "N/A"' 2>/dev/null)

        # If still not found, use the block height as a fallback identifier
        if [[ "$block_hash" == "N/A" || "$block_hash" == "null" ]]; then
            block_hash="$height"
        fi

        if [[ -n "$block_hash" && "$block_hash" != "N/A" && "$block_hash" != "null" ]]; then
            log_success "Found upgrade block hash $block_hash for $network"
            echo "$block_hash"
        else
            log_error "No block hash found for $network at height $height"
            echo "N/A"
        fi
    else
        log_error "Failed to query block for $network at height $height: $result"
        echo "ERROR"
    fi
}

# Get value by index from space-separated results
get_result_value() {
    local results="$1"
    local index="$2"
    echo "$results" | cut -d' ' -f$((index + 1))
}

# Generate markdown output
generate_markdown() {
    local version="$1"
    local results="$2"

    # Extract values from results string
    local alpha_height=$(get_result_value "$results" 0)
    local alpha_height_url="https://shannon-alpha.trustsoothe.io/block/${alpha_height}"
    local alpha_block=$(get_result_value "$results" 1)
    local alpha_block_url="https://shannon-alpha.trustsoothe.io/block/${alpha_block}"

    local beta_height=$(get_result_value "$results" 2)
    local beta_height_url="https://shannon-beta.trustsoothe.io/block/${beta_height}"
    local beta_block=$(get_result_value "$results" 3)
    local beta_block_url="https://shannon-beta.trustsoothe.io/block/${beta_block}"

    local main_height=$(get_result_value "$results" 4)
    local main_height_url="https://shannon-mainnet.trustsoothe.io/block/${main_height}"
    local main_block=$(get_result_value "$results" 5)
    local main_block_url="https://shannon-mainnet.trustsoothe.io/block/${main_block}"

    # Add query time header
    echo "Query Time: $(date)"
    echo ""

    cat <<EOF
## Pocket Network Protocol Upgrades

> [!IMPORTANT]
> TO PROTOCOL MAINTAINER: See the [example release template](https://dev.poktroll.com/develop/upgrades/release_procedure#93-example-release-template) for an example of what to write below.
> Remove this note afterwards.

## Upgrade Details

| Network       | Upgrade Block Height | Notes |
| ------------- | -------------- | ----- |
| Alpha TestNet | [${alpha_height:-⚪}](${alpha_height_url:-⚪}) | ${alpha_block:+🟢}${alpha_block:-⚪} |
| Beta TestNet  | [${beta_height:-⚪}](${beta_height_url:-⚪}) | ${beta_block:+🟢}${beta_block:-⚪} |
| MainNet       | [${main_height:-⚪}](${main_height_url:-⚪}) | ${main_block:+🟢}${main_block:-⚪} |

## What's changed? (Autogenerated release notes)

EOF
}

# Generate JSON output
generate_json_output() {
    local version="$1"
    local results="$2"

    # Extract values from results string
    local alpha_height=$(get_result_value "$results" 0)
    local alpha_height_url="https://shannon-alpha.trustsoothe.io/block/${alpha_height}"
    local alpha_block=$(get_result_value "$results" 1)
    local alpha_block_url="https://shannon-alpha.trustsoothe.io/block/${alpha_block}"

    local beta_height=$(get_result_value "$results" 2)
    local beta_height_url="https://shannon-beta.trustsoothe.io/block/${beta_height}"
    local beta_block=$(get_result_value "$results" 3)
    local beta_block_url="https://shannon-beta.trustsoothe.io/block/${beta_block}"

    local main_height=$(get_result_value "$results" 4)
    local main_height_url="https://shannon-mainnet.trustsoothe.io/block/${main_height}"
    local main_block=$(get_result_value "$results" 5)
    local main_block_url="https://shannon-mainnet.trustsoothe.io/block/${main_block}"

    cat <<EOF
{
  "upgrade_version": "$version",
  "query_timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "networks": {
    "alpha": {
      "name": "Alpha TestNet",
      "upgrade_height": "$alpha_height",
      "upgrade_height_url": "$alpha_height_url",
      "upgrade_block_hash": "$alpha_block",
      "upgrade_block_url": "$alpha_block_url"
    },
    "beta": {
      "name": "Beta TestNet",
      "upgrade_height": "$beta_height",
      "upgrade_height_url": "$beta_height_url",
      "upgrade_block_hash": "$beta_block",
      "upgrade_block_url": "$beta_block_url"
    },
    "main": {
      "name": "MainNet",
      "upgrade_height": "$main_height",
      "upgrade_height_url": "$main_height_url",
      "upgrade_block_hash": "$main_block",
      "upgrade_block_url": "$main_block_url"
    }
  }
}
EOF
}

# Main function
# Main function
main() {
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
        -h | --help)
            usage
            exit 0
            ;;
        -v | --verbose)
            VERBOSE=true
            shift
            ;;
        --output)
            OUTPUT_FORMAT="$2"
            shift 2
            ;;
        -*)
            log_error "Unknown option: $1"
            echo "Use --help for usage information."
            exit 1
            ;;
        *)
            if [[ -z "$UPGRADE_VERSION" ]]; then
                UPGRADE_VERSION="$1"
            else
                log_error "Too many arguments. Expected one upgrade version."
                exit 1
            fi
            shift
            ;;
        esac
    done

    # Validate arguments
    if [[ -z "$UPGRADE_VERSION" ]]; then
        log_error "Upgrade version is required."
        echo "Use --help for usage information."
        exit 1
    fi

    # Validate output format
    case "$OUTPUT_FORMAT" in
    json | markdown) ;;
    *)
        log_error "Invalid output format: $OUTPUT_FORMAT"
        log_error "Valid formats: json, markdown"
        exit 1
        ;;
    esac

    # Check dependencies
    check_dependencies

    log_info "Starting upgrade query for version: $UPGRADE_VERSION"
    log_info "Output format: $OUTPUT_FORMAT"

    # Query all networks for heights and transaction hashes
    local results=""
    local first=true
    for network in alpha beta main; do
        local height
        height=$(query_network_upgrade "$network" "$UPGRADE_VERSION")

        local block_hash
        block_hash=$(query_upgrade_block_hash "$network" "$height")

        if [[ "$first" == "true" ]]; then
            results="$height $block_hash"
            first=false
        else
            results="$results $height $block_hash"
        fi
    done

    # Generate output based on format
    case "$OUTPUT_FORMAT" in
    json)
        generate_json_output "$UPGRADE_VERSION" "$results"
        ;;
    markdown)
        generate_markdown "$UPGRADE_VERSION" "$results"
        ;;
    esac

    log_info "Query completed successfully"
}

# Run main function with all arguments
main "$@"
