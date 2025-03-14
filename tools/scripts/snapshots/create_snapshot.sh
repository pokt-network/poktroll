#!/bin/bash

# TODO_IMPROVE(@okdas): Add support for this script in our helm-charts.
# Ref: https://github.com/pokt-network/poktroll/pull/1092
# Note that there's also an internal doc (mentioned in the PR):  I also submitted the script that we use to create snapshots in the same PR. I just want to preserve it in Git, along with the index page. I added a readme so it explains why these files are there. They don't necessarily need to be saved in that repo and can go somewhere else.

set -e

# Prerequisites:
# 1. This script must be run as the poktroll user
# 2. The poktroll user must have passwordless sudo access to systemctl commands for cosmovisor service
#    Add the following line to /etc/sudoers.d/poktroll-cosmovisor (using visudo):
#    poktroll ALL=(ALL) NOPASSWD: /bin/systemctl stop cosmovisor, /bin/systemctl start cosmovisor
# 3. curl must be installed (usually installed by default)
#    If not: sudo apt-get update && sudo apt-get install -y curl
# 4. zstd must be installed for compression
#    Install: sudo apt-get update && sudo apt-get install -y zstd
# 5. mktorrent must be installed for creating torrent files
#    Install: sudo apt-get update && sudo apt-get install -y mktorrent
# 6. WebDAV credentials must be configured:
#    - WEBDAV_USER environment variable must be set
#    - WEBDAV_PASS environment variable must be set
#    - NETWORK environment variable must be set (e.g., mainnet, testnet-beta)
#    Example:
#    export WEBDAV_USER="uploader"
#    export WEBDAV_PASS="your-secure-password"
#    export NETWORK="testnet-beta"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Replace the Vultr variables with WebDAV configuration
WEBDAV_UPLOAD_URL="https://upload-snapshots.us-nj.poktroll.com"
SNAPSHOT_URL="https://snapshots.us-nj.poktroll.com"
WEBDAV_USER="xxx"
WEBDAV_PASS="xxx"                   # This should be passed as an environment variable in production
NETWORK="testnet-beta"              # This should be passed as an environment variable: mainnet, testnet-beta, etc.
LISTING_URL="${SNAPSHOT_URL}/list/" # URL for directory listing

# Torrent trackers
TRACKERS=(
    "udp://tracker.opentrackr.org:1337/announce"
    "udp://tracker.openbittorrent.com:80/announce"
)

# Function to print colored output
print_color() {
    COLOR=$1
    MESSAGE=$2
    echo -e "${COLOR}${MESSAGE}${NC}"
}

# Check if running as poktroll user
check_user() {
    if [[ $USER != "poktroll" ]]; then
        print_color $RED "This script must be run as the poktroll user."
        print_color $YELLOW "Please switch to the poktroll user with: su - poktroll"
        exit 1
    fi
}

# Function to stop the node
stop_node() {
    # We need to stop the node before taking snapshots to ensure data consistency
    print_color $YELLOW "Stopping poktrolld node..."
    sudo systemctl stop cosmovisor

    # Important: Wait for process to fully stop to ensure clean shutdown
    while pgrep poktrolld >/dev/null; do
        print_color $YELLOW "Waiting for poktrolld to stop..."
        sleep 5
    done

    print_color $GREEN "Node stopped successfully"
}

# Function to create snapshots
create_snapshots() {
    print_color $YELLOW "Creating snapshots..."

    # Create snapshot directory if it doesn't exist
    SNAPSHOT_DIR="$HOME/snapshots"
    mkdir -p "$SNAPSHOT_DIR"

    # Clean up old snapshot files in the snapshot directory - simplified approach
    print_color $YELLOW "Cleaning up old snapshot files in $SNAPSHOT_DIR..."
    # Remove everything in the snapshot directory
    rm -rf "$SNAPSHOT_DIR"/*
    print_color $GREEN "Old snapshot files cleaned up"

    cd "$HOME"

    # First check if snapshots directory exists and is writable
    if [ ! -d "$HOME/.poktroll/snapshots" ]; then
        print_color $YELLOW "Creating snapshots directory..."
        mkdir -p "$HOME/.poktroll/snapshots"
    fi

    # List and delete any existing snapshots
    EXISTING_SNAPSHOTS=$(~/.poktroll/cosmovisor/current/bin/poktrolld snapshots list)
    if [ -n "$EXISTING_SNAPSHOTS" ]; then
        while read -r line; do
            if [ -n "$line" ]; then
                HEIGHT=$(echo "$line" | grep -o 'height: [0-9]*' | grep -o '[0-9]*')
                print_color $YELLOW "Deleting existing snapshot at height $HEIGHT..."
                ~/.poktroll/cosmovisor/current/bin/poktrolld snapshots delete "$HEIGHT" 3
                # Wait a moment for deletion to complete
                sleep 2
            fi
        done <<<"$EXISTING_SNAPSHOTS"

        # Verify snapshots are deleted
        VERIFY_EMPTY=$(~/.poktroll/cosmovisor/current/bin/poktrolld snapshots list)
        if [ -n "$VERIFY_EMPTY" ]; then
            print_color $RED "Failed to delete existing snapshots"
            exit 1
        fi
    fi

    # Try to create new snapshot
    print_color $YELLOW "Creating new pruned snapshot..."
    EXPORT_OUTPUT=$(~/.poktroll/cosmovisor/current/bin/poktrolld snapshots export 2>&1)
    EXPORT_STATUS=$?

    if [ $EXPORT_STATUS -eq 0 ]; then
        CURRENT_HEIGHT=$(~/.poktroll/cosmovisor/current/bin/poktrolld snapshots list | tail -1 | grep -o 'height: [0-9]*' | grep -o '[0-9]*')
        print_color $GREEN "Successfully exported snapshot at height $CURRENT_HEIGHT"
    else
        # If snapshot already exists, extract the height from the error message
        if echo "$EXPORT_OUTPUT" | grep -q "snapshot already exists at height"; then
            CURRENT_HEIGHT=$(echo "$EXPORT_OUTPUT" | grep -o "height [0-9]*" | grep -o "[0-9]*")
            print_color $YELLOW "Using existing snapshot at height $CURRENT_HEIGHT"
        else
            print_color $RED "Failed to create snapshot:"
            print_color $RED "$EXPORT_OUTPUT"
            exit 1
        fi
    fi

    # Create both pruned and archival snapshots
    # Pruned snapshot: Created by poktrolld, contains only the latest state
    # Archival snapshot: Full copy of .poktroll directory, contains all historical data
    print_color $YELLOW "Creating pruned snapshot archive..."
    if ! ~/.poktroll/cosmovisor/current/bin/poktrolld snapshots dump "$CURRENT_HEIGHT" 3 \
        --output "$SNAPSHOT_DIR/${CURRENT_HEIGHT}-pruned.tar.gz"; then
        print_color $RED "Failed to dump snapshot"
        exit 1
    fi

    print_color $YELLOW "Creating archival snapshot..."
    # Use zstd for better compression ratio and speed compared to gzip
    cd "$HOME/.poktroll/data"
    tar --zstd -cf "$SNAPSHOT_DIR/${CURRENT_HEIGHT}-archival.tar.zst" .
    cd "$HOME"

    # Store version information for compatibility checking
    BINARY_VERSION=$(~/.poktroll/cosmovisor/current/bin/poktrolld version)
    echo "$BINARY_VERSION" >"$SNAPSHOT_DIR/${CURRENT_HEIGHT}-version.txt"

    print_color $GREEN "Snapshots created successfully in $SNAPSHOT_DIR:"
    ls -lh "$SNAPSHOT_DIR"
}

# Function to create torrent files
create_torrents() {
    local height=$1
    print_color $YELLOW "Creating torrent files for snapshots..."

    # Create torrent for archival snapshot
    print_color $YELLOW "Creating torrent for archival snapshot..."
    local archival_file="${SNAPSHOT_DIR}/${height}-archival.tar.zst"
    local archival_torrent="${SNAPSHOT_DIR}/${height}-archival.torrent"
    local archival_web_url="${SNAPSHOT_URL}/${NETWORK}-${height}-archival.tar.zst"

    # Remove existing torrent file if it exists
    if [ -f "$archival_torrent" ]; then
        print_color $YELLOW "Removing existing archival torrent file: $archival_torrent"
        rm -f "$archival_torrent"
    fi

    # Build tracker arguments
    local tracker_args=""
    for tracker in "${TRACKERS[@]}"; do
        tracker_args+=" -a $tracker"
    done

    # Create archival torrent
    if mktorrent \
        $tracker_args \
        -w "$archival_web_url" \
        -o "$archival_torrent" \
        "$archival_file"; then
        print_color $GREEN "Created archival torrent: $archival_torrent"
    else
        print_color $RED "Failed to create archival torrent"
        return 1
    fi

    # Create torrent for pruned snapshot
    print_color $YELLOW "Creating torrent for pruned snapshot..."
    local pruned_file="${SNAPSHOT_DIR}/${height}-pruned.tar.gz"
    local pruned_torrent="${SNAPSHOT_DIR}/${height}-pruned.torrent"
    local pruned_web_url="${SNAPSHOT_URL}/${NETWORK}-${height}-pruned.tar.gz"

    # Remove existing torrent file if it exists
    if [ -f "$pruned_torrent" ]; then
        print_color $YELLOW "Removing existing pruned torrent file: $pruned_torrent"
        rm -f "$pruned_torrent"
    fi

    # Create pruned torrent
    if mktorrent \
        $tracker_args \
        -w "$pruned_web_url" \
        -o "$pruned_torrent" \
        "$pruned_file"; then
        print_color $GREEN "Created pruned torrent: $pruned_torrent"
    else
        print_color $RED "Failed to create pruned torrent"
        return 1
    fi

    print_color $GREEN "Torrent files created successfully"
    return 0
}

# Function to create RSS feed for torrents
create_rss_feed() {
    local height=$1
    local timestamp=$(date -u +"%a, %d %b %Y %H:%M:%S GMT")
    local rss_file="${SNAPSHOT_DIR}/torrents.xml"

    print_color $YELLOW "Creating RSS feed for torrent files..."

    # Get list of remote torrent files using directory listing page
    print_color $YELLOW "Fetching remote torrent list from directory listing..."

    # Save the listing response for parsing
    local listing_response=$(curl -s "${LISTING_URL}/")

    # Debug output - show a snippet of the listing response
    print_color $YELLOW "Listing response snippet (first 200 chars):"
    echo "${listing_response:0:200}..."

    # Extract torrent filenames from the HTML directory listing
    local remote_torrents=$(echo "$listing_response" | grep -o "href=\"${NETWORK}-[0-9]\+-.*\.torrent\"" |
        sed 's/href="\(.*\)"/\1/g' | sort -r | head -10)

    # If that didn't work, display a warning but proceed with just the current torrent
    if [ -z "$remote_torrents" ]; then
        print_color $RED "WARNING: Could not retrieve list of existing torrents. RSS feed will only contain the current snapshot."
        print_color $YELLOW "Please check server configuration and connectivity."
    fi

    # Debug output
    print_color $YELLOW "Found remote torrents:"
    echo "$remote_torrents"

    # Create RSS feed header
    cat >"$rss_file" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
  <channel>
    <title>Poktroll ${NETWORK} Snapshots</title>
    <link>${SNAPSHOT_URL}/</link>
    <description>Torrent files for Poktroll ${NETWORK} blockchain snapshots</description>
    <language>en-us</language>
    <pubDate>${timestamp}</pubDate>
    <lastBuildDate>${timestamp}</lastBuildDate>
    <atom:link href="${SNAPSHOT_URL}/${NETWORK}-torrents.xml" rel="self" type="application/rss+xml" />
EOF

    # Add current snapshot torrents as items
    cat >>"$rss_file" <<EOF
    <item>
      <title>Poktroll ${NETWORK} Archival Snapshot (Height: ${height})</title>
      <description>Archival snapshot of Poktroll ${NETWORK} at block height ${height}</description>
      <pubDate>${timestamp}</pubDate>
      <guid>${SNAPSHOT_URL}/${NETWORK}-${height}-archival.torrent</guid>
      <enclosure url="${SNAPSHOT_URL}/${NETWORK}-${height}-archival.torrent" type="application/x-bittorrent" />
    </item>
    <item>
      <title>Poktroll ${NETWORK} Pruned Snapshot (Height: ${height})</title>
      <description>Pruned snapshot of Poktroll ${NETWORK} at block height ${height}</description>
      <pubDate>${timestamp}</pubDate>
      <guid>${SNAPSHOT_URL}/${NETWORK}-${height}-pruned.torrent</guid>
      <enclosure url="${SNAPSHOT_URL}/${NETWORK}-${height}-pruned.torrent" type="application/x-bittorrent" />
    </item>
EOF

    # Add previous snapshot torrents from remote server
    if [ -n "$remote_torrents" ]; then
        echo "$remote_torrents" | while read -r torrent; do
            # Debug output
            print_color $YELLOW "Processing torrent: $torrent"

            # Skip current height torrents (already added above)
            if [[ "$torrent" == *"${height}"* ]]; then
                print_color $YELLOW "Skipping current height torrent: $torrent"
                continue
            fi

            # Extract height and type from filename
            local t_height=$(echo "$torrent" | grep -o '[0-9]\+' | head -1)
            local t_type=$(echo "$torrent" | grep -o '\(archival\|pruned\)')

            print_color $YELLOW "Extracted height: $t_height, type: $t_type"

            if [ -n "$t_height" ] && [ -n "$t_type" ]; then
                # Use a more portable approach for generating timestamps
                local t_timestamp=$(date -u +"%a, %d %b %Y %H:%M:%S GMT")

                print_color $GREEN "Adding torrent to RSS feed: $torrent"
                cat >>"$rss_file" <<EOF
    <item>
      <title>Poktroll ${NETWORK} ${t_type^} Snapshot (Height: ${t_height})</title>
      <description>${t_type^} snapshot of Poktroll ${NETWORK} at block height ${t_height}</description>
      <pubDate>${t_timestamp}</pubDate>
      <guid>${SNAPSHOT_URL}/${torrent}</guid>
      <enclosure url="${SNAPSHOT_URL}/${torrent}" type="application/x-bittorrent" />
    </item>
EOF
            fi
        done
    fi

    # Close RSS feed
    cat >>"$rss_file" <<EOF
  </channel>
</rss>
EOF

    print_color $GREEN "RSS feed created: $rss_file"
    return 0
}

# Function to start the node
start_node() {
    print_color $YELLOW "Starting poktrolld node..."
    sudo systemctl start cosmovisor
    print_color $GREEN "Node started successfully"
}

# Function to clean old snapshots
clean_old_snapshots() {
    print_color $YELLOW "Cleaning snapshots..."

    # Clean all local snapshots
    print_color $YELLOW "Cleaning local snapshot files..."
    # Simply remove all files - we've already created and uploaded the new ones
    rm -rf "$SNAPSHOT_DIR"/*
    print_color $GREEN "Local snapshot files cleaned"

    # Clean snapshots from poktrolld store
    print_color $YELLOW "Cleaning snapshots from poktrolld store..."
    SNAPSHOTS_LIST=$(~/.poktroll/cosmovisor/current/bin/poktrolld snapshots list)

    # Parse and delete old snapshots, keeping only the latest
    echo "$SNAPSHOTS_LIST" | while read -r line; do
        if [ -n "$line" ]; then
            HEIGHT=$(echo "$line" | grep -o 'height: [0-9]*' | grep -o '[0-9]*')
            FORMAT=$(echo "$line" | grep -o 'format: [0-9]*' | grep -o '[0-9]*')

            # Don't delete the current height
            if [ "$HEIGHT" != "$CURRENT_HEIGHT" ]; then
                print_color $YELLOW "Deleting snapshot at height $HEIGHT format $FORMAT"
                ~/.poktroll/cosmovisor/current/bin/poktrolld snapshots delete "$HEIGHT" "$FORMAT"
            fi
        fi
    done
    print_color $GREEN "Poktrolld snapshots cleaned"

    # Clean remote snapshots
    print_color $YELLOW "Cleaning old snapshots from WebDAV server..."

    # Get list of remote files for this network using directory listing page
    print_color $YELLOW "Fetching remote file list from directory listing..."

    # Save the listing response for parsing
    local listing_response=$(curl -s "${LISTING_URL}/")

    # Debug output - show a snippet of the listing response
    print_color $YELLOW "Listing response snippet (first 200 chars):"
    echo "${listing_response:0:200}..."

    # Extract filenames from the HTML directory listing
    REMOTE_FILES=$(echo "$listing_response" | grep -o "href=\"${NETWORK}-[0-9]\+.*\"" |
        sed 's/href="\(.*\)"/\1/g' | sort)

    # If that didn't work, display a warning
    if [ -z "$REMOTE_FILES" ]; then
        print_color $RED "WARNING: Could not retrieve list of existing remote files. Cannot clean old snapshots."
        print_color $YELLOW "Please check server configuration and connectivity."
        return 1
    fi

    # Debug output
    print_color $YELLOW "Found remote files:"
    echo "$REMOTE_FILES"

    # Function to delete old snapshots of a specific type
    delete_old_snapshots() {
        local pattern=$1
        # Use grep -e to properly handle patterns starting with a dash
        local files=$(echo "$REMOTE_FILES" | grep -e "$pattern" | sort -r)
        local count=0

        echo "$files" | while read -r file; do
            count=$((count + 1))
            if [ $count -gt 3 ]; then
                if curl -u "${WEBDAV_USER}:${WEBDAV_PASS}" \
                    -X DELETE \
                    "${WEBDAV_UPLOAD_URL}/${file}" \
                    --fail --silent --show-error; then
                    print_color $GREEN "Deleted old snapshot: ${file}"
                else
                    print_color $RED "Failed to delete: ${file}"
                fi
            fi
        done
    }

    # Delete old snapshots of each type
    delete_old_snapshots "-archival.tar.zst"
    delete_old_snapshots "-pruned.tar.gz"
    delete_old_snapshots "-archival.torrent"
    delete_old_snapshots "-pruned.torrent"

    print_color $GREEN "Old remote snapshots cleaned successfully (keeping latest 3)"
}

# Replace check_s3cmd with check_curl
check_curl() {
    if ! command -v curl &>/dev/null; then
        print_color $RED "curl is not installed. Please install it first:"
        print_color $YELLOW "sudo apt-get update && sudo apt-get install -y curl"
        exit 1
    fi
}

# Check if mktorrent is installed
check_mktorrent() {
    if ! command -v mktorrent &>/dev/null; then
        print_color $RED "mktorrent is not installed. Please install it first:"
        print_color $YELLOW "sudo apt-get update && sudo apt-get install -y mktorrent"
        exit 1
    fi
}

# Replace upload_snapshots function
upload_snapshots() {
    local height=$1
    print_color $YELLOW "Uploading snapshots to WebDAV server..."

    # Function to upload a file using curl
    upload_file() {
        local file=$1
        local remote_name=$2

        if [ -z "$remote_name" ]; then
            remote_name="${NETWORK}-$(basename "$file")" # Add network prefix to filename
        fi

        if curl -u "${WEBDAV_USER}:${WEBDAV_PASS}" \
            -T "$file" \
            "${WEBDAV_UPLOAD_URL}/${remote_name}" \
            --fail --silent --show-error \
            --max-time 3600 \
            --retry 3 \
            --retry-delay 5 \
            --retry-max-time 60 \
            --connect-timeout 60; then
            print_color $GREEN "Uploaded: ${remote_name}"
            return 0
        else
            print_color $RED "Failed to upload: ${remote_name}"
            return 1
        fi
    }

    # Upload all files
    local failed=0
    upload_file "$SNAPSHOT_DIR/${height}-archival.tar.zst" || failed=1
    upload_file "$SNAPSHOT_DIR/${height}-pruned.tar.gz" || failed=1
    upload_file "$SNAPSHOT_DIR/${height}-version.txt" || failed=1

    # Upload torrent files
    upload_file "$SNAPSHOT_DIR/${height}-archival.torrent" "${NETWORK}-${height}-archival.torrent" || failed=1
    upload_file "$SNAPSHOT_DIR/${height}-pruned.torrent" "${NETWORK}-${height}-pruned.torrent" || failed=1

    if [ $failed -eq 0 ]; then
        # Create latest text files only after successful upload
        print_color $YELLOW "Creating latest height indicator files..."
        echo "$height" >"$SNAPSHOT_DIR/latest-archival.txt"
        echo "$height" >"$SNAPSHOT_DIR/latest-pruned.txt"

        # Create latest torrent symlinks only after successful upload
        print_color $YELLOW "Creating latest torrent symlinks..."
        ln -sf "${height}-archival.torrent" "${SNAPSHOT_DIR}/latest-archival.torrent"
        ln -sf "${height}-pruned.torrent" "${SNAPSHOT_DIR}/latest-pruned.torrent"

        # Create RSS feed for torrents
        create_rss_feed "$height"

        # Upload the latest indicator files and RSS feed
        upload_file "$SNAPSHOT_DIR/latest-archival.txt" || failed=1
        upload_file "$SNAPSHOT_DIR/latest-pruned.txt" || failed=1
        upload_file "$SNAPSHOT_DIR/latest-archival.torrent" "${NETWORK}-latest-archival.torrent" || failed=1
        upload_file "$SNAPSHOT_DIR/latest-pruned.torrent" "${NETWORK}-latest-pruned.torrent" || failed=1
        upload_file "$SNAPSHOT_DIR/torrents.xml" "${NETWORK}-torrents.xml" || failed=1

        print_color $GREEN "Snapshots uploaded successfully"
        print_color $GREEN "Archival snapshot: ${SNAPSHOT_URL}/${NETWORK}-${height}-archival.tar.zst"
        print_color $GREEN "Pruned snapshot: ${SNAPSHOT_URL}/${NETWORK}-${height}-pruned.tar.gz"
        print_color $GREEN "Torrent files:"
        print_color $GREEN "  ${SNAPSHOT_URL}/${NETWORK}-${height}-archival.torrent"
        print_color $GREEN "  ${SNAPSHOT_URL}/${NETWORK}-${height}-pruned.torrent"
        print_color $GREEN "Latest pointers:"
        print_color $GREEN "  ${SNAPSHOT_URL}/${NETWORK}-latest-archival.txt"
        print_color $GREEN "  ${SNAPSHOT_URL}/${NETWORK}-latest-pruned.txt"
        print_color $GREEN "  ${SNAPSHOT_URL}/${NETWORK}-latest-archival.torrent"
        print_color $GREEN "  ${SNAPSHOT_URL}/${NETWORK}-latest-pruned.torrent"
        print_color $GREEN "RSS feed:"
        print_color $GREEN "  ${SNAPSHOT_URL}/${NETWORK}-torrents.xml"
    else
        print_color $RED "Some uploads failed"
        exit 1
    fi
}

# Main function
main() {
    print_color $GREEN "Starting snapshot creation process..."

    # Check requirements
    check_user
    check_curl
    check_mktorrent

    # Stop the node
    stop_node

    # Create snapshots
    create_snapshots

    # Create torrent files
    create_torrents "$CURRENT_HEIGHT"

    # Upload snapshots
    upload_snapshots "$CURRENT_HEIGHT"

    # Clean old snapshots (both local and remote)
    clean_old_snapshots

    # Start the node
    start_node

    print_color $GREEN "Snapshot process completed successfully!"
    print_color $YELLOW "Snapshots are available locally in $HOME/snapshots/"
    print_color $YELLOW "and remotely at ${SNAPSHOT_URL}/"
    print_color $YELLOW "Latest snapshot heights are stored in:"
    print_color $YELLOW "  - latest-archival.txt"
    print_color $YELLOW "  - latest-pruned.txt"
    print_color $YELLOW "Version information is stored in ${CURRENT_HEIGHT}-version.txt"
    print_color $YELLOW "Torrent files are available at:"
    print_color $YELLOW "  - ${SNAPSHOT_URL}/${NETWORK}-${CURRENT_HEIGHT}-archival.torrent"
    print_color $YELLOW "  - ${SNAPSHOT_URL}/${NETWORK}-${CURRENT_HEIGHT}-pruned.torrent"
    print_color $YELLOW "  - ${SNAPSHOT_URL}/${NETWORK}-latest-archival.torrent"
    print_color $YELLOW "  - ${SNAPSHOT_URL}/${NETWORK}-latest-pruned.torrent"
    print_color $YELLOW "RSS feed for torrents is available at:"
    print_color $YELLOW "  - ${SNAPSHOT_URL}/${NETWORK}-torrents.xml"
}

# Run main function
main
