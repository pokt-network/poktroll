#!/bin/bash
# Fix protobuf RPC comments to have consistent module.Operation format

set -euo pipefail

# List of modules to update
modules=(
    "application"
    "session"
    "shared"
    "supplier"
    "tokenomics"
    "migration"
)

for module in "${modules[@]}"; do
    file="proto/pocket/$module/tx.proto"
    if [ -f "$file" ]; then
        echo "Updating $file..."

        # Replace the verbose UpdateParams comment
        sed -i.bak '
        /\/\/ UpdateParams defines a (governance) operation for updating the module/,/\/\/ parameters\. The authority defaults to the x\/gov module account\./ {
            c\
  // '"$module"'.MsgUpdateParams updates all module parameters via governance.
        }
        ' "$file"

        # Add comment for UpdateParam if it doesn't have one
        sed -i.bak '
        /rpc UpdateParam.*returns.*MsgUpdateParamResponse/ {
            i\
  // '"$module"'.MsgUpdateParam updates a single module parameter via governance.
        }
        ' "$file"

        # Remove backup file
        rm -f "$file.bak"
        echo "Updated $module module"
    else
        echo "File not found: $file"
    fi
done

echo "All protobuf comments updated!"