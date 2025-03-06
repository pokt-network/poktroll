#!/bin/bash

# Set error handling
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_color() {
    COLOR=$1
    MESSAGE=$2
    echo -e "${COLOR}${MESSAGE}${NC}"
}

# Configuration
SNAPSHOT_DIR="/home/poktroll/snapshots"
NODE_HOME="/home/poktroll/.poktroll"
NETWORK="testnet-beta"  # Change this to match your network

# Function to stop the node
stop_node() {
    print_color $YELLOW "Stopping poktrolld node..."
    sudo systemctl stop cosmovisor-poktroll
    sleep 5
    print_color $GREEN "Node stopped successfully"
}

# Function to start the node
start_node() {
    print_color $YELLOW "Starting poktrolld node..."
    sudo systemctl start cosmovisor-poktroll
    print_color $GREEN "Node started successfully"
}

# Function to get current block height
get_block_height() {
    HEIGHT=$(curl -s http://localhost:26657/status | jq -r '.result.sync_info.latest_block_height')
    echo $HEIGHT
}

# Function to clean up existing snapshot files at a specific height
cleanup_existing_snapshot() {
    local height=$1
    print_color $YELLOW "Checking for existing snapshots at height $height..."
    
    # Check if files exist before attempting to delete
    if ls $SNAPSHOT_DIR/$height-* 1> /dev/null 2>&1; then
        print_color $YELLOW "Deleting existing snapshot files at height $height..."
        rm -f $SNAPSHOT_DIR/$height-*
        print_color $GREEN "Existing snapshot files deleted successfully"
    else
        print_color $GREEN "No existing snapshot files found at height $height"
    fi
}

# Function to create snapshots
create_snapshots() {
    # Get current block height
    HEIGHT=$(get_block_height)
    print_color $YELLOW "Creating snapshots at height $HEIGHT..."
    
    # Create snapshots directory if it doesn't exist
    mkdir -p $SNAPSHOT_DIR
    
    # Clean up existing snapshots at this height
    cleanup_existing_snapshot $HEIGHT
    
    # Create version file
    POKTROLLD_VERSION=$(poktrolld version)
    echo $POKTROLLD_VERSION > $SNAPSHOT_DIR/$HEIGHT-version.txt
    
    # Create pruned snapshot
    print_color $YELLOW "Creating pruned snapshot..."
    poktrolld export --home=$NODE_HOME --height=$HEIGHT 2> /dev/null > $SNAPSHOT_DIR/$HEIGHT-pruned.json
    if [ $? -eq 0 ]; then
        print_color $GREEN "Successfully exported pruned snapshot at height $HEIGHT"
        # Create pruned snapshot archive
        print_color $YELLOW "Creating pruned snapshot archive..."
        tar -czf $SNAPSHOT_DIR/$HEIGHT-pruned.tar.gz -C $SNAPSHOT_DIR $HEIGHT-pruned.json
        rm $SNAPSHOT_DIR/$HEIGHT-pruned.json
    else
        print_color $RED "Failed to export pruned snapshot"
        exit 1
    fi
    
    # Create archival snapshot
    print_color $YELLOW "Creating archival snapshot..."
    tar -cf - -C $NODE_HOME data | zstd -T0 -19 > $SNAPSHOT_DIR/$HEIGHT-archival.tar.zst
    if [ $? -eq 0 ]; then
        print_color $GREEN "Successfully created archival snapshot at height $HEIGHT"
    else
        print_color $RED "Failed to create archival snapshot"
        exit 1
    fi
    
    # Update latest symlinks and text files
    print_color $YELLOW "Updating latest snapshot references..."
    
    # Remove old symlinks if they exist
    rm -f $SNAPSHOT_DIR/latest-pruned.torrent
    rm -f $SNAPSHOT_DIR/latest-archival.torrent
    
    # Update latest height text files
    echo $HEIGHT > $SNAPSHOT_DIR/latest-pruned.txt
    echo $HEIGHT > $SNAPSHOT_DIR/latest-archival.txt
    
    print_color $GREEN "Snapshots created successfully in $SNAPSHOT_DIR:"
    ls -lh $SNAPSHOT_DIR
}

# Function to create torrent files
create_torrents() {
    print_color $YELLOW "Creating torrent files for snapshots..."
    
    # Get current block height from latest-archival.txt
    HEIGHT=$(cat $SNAPSHOT_DIR/latest-archival.txt)
    
    # Clean up existing torrent files at this height
    rm -f $SNAPSHOT_DIR/$HEIGHT-*.torrent
    
    # Create torrent for pruned snapshot
    print_color $YELLOW "Creating torrent for pruned snapshot..."
    mktorrent -v -a udp://tracker.opentrackr.org:1337 -w https://snapshots.us-nj.poktroll.com/$NETWORK-$HEIGHT-pruned.tar.gz -o $SNAPSHOT_DIR/$HEIGHT-pruned.torrent $SNAPSHOT_DIR/$HEIGHT-pruned.tar.gz
    if [ $? -eq 0 ]; then
        print_color $GREEN "Pruned torrent created successfully"
        # Create symlink to latest pruned torrent
        ln -sf $HEIGHT-pruned.torrent $SNAPSHOT_DIR/latest-pruned.torrent
    else
        print_color $RED "Failed to create pruned torrent"
    fi
    
    # Create torrent for archival snapshot
    print_color $YELLOW "Creating torrent for archival snapshot..."
    mktorrent -v -a udp://tracker.opentrackr.org:1337 -w https://snapshots.us-nj.poktroll.com/$NETWORK-$HEIGHT-archival.tar.zst -o $SNAPSHOT_DIR/$HEIGHT-archival.torrent $SNAPSHOT_DIR/$HEIGHT-archival.tar.zst
    if [ $? -eq 0 ]; then
        print_color $GREEN "Archival torrent created successfully"
        # Create symlink to latest archival torrent
        ln -sf $HEIGHT-archival.torrent $SNAPSHOT_DIR/latest-archival.torrent
    else
        print_color $RED "Failed to create archival torrent"
    fi
    
    # Create torrents.xml file for RSS feed
    print_color $YELLOW "Creating torrents.xml file..."
    cat > $SNAPSHOT_DIR/torrents.xml <<EOF
<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0">
  <channel>
    <title>Poktroll $NETWORK Snapshots</title>
    <link>https://snapshots.us-nj.poktroll.com/</link>
    <description>Poktroll $NETWORK node snapshots</description>
    <item>
      <title>Poktroll $NETWORK Pruned Snapshot (Height: $HEIGHT)</title>
      <link>https://snapshots.us-nj.poktroll.com/$NETWORK-$HEIGHT-pruned.torrent</link>
      <pubDate>$(date -R)</pubDate>
    </item>
    <item>
      <title>Poktroll $NETWORK Archival Snapshot (Height: $HEIGHT)</title>
      <link>https://snapshots.us-nj.poktroll.com/$NETWORK-$HEIGHT-archival.torrent</link>
      <pubDate>$(date -R)</pubDate>
    </item>
  </channel>
</rss>
EOF
    
    print_color $GREEN "Torrent files created successfully"
}

# Main function
main() {
    print_color $GREEN "Starting snapshot creation process..."
    
    # Stop the node
    stop_node
    
    # Create snapshots
    create_snapshots
    
    # Create torrent files
    create_torrents
    
    # Start the node
    start_node
    
    print_color $GREEN "Snapshot creation process completed successfully!"
}

# Run the main function
main 