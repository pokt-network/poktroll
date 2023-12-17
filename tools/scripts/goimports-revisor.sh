#!/usr/bin/env bash

FLAGS=(
    -rm-unused
    -use-cache
    -imports-order "std,general,company,project,blanked,dotted"
    -project-name "github.com/pokt-network/poktroll"
)

# String to check within the import block
EXCLUDE_STRING="this line is used by starport scaffolding"

# Define a function to check for the exclusion comment within the import block and run goimports-reviser
process_file() {
    local file=$1
    echo "Checking $file"

    # Extract the import block and check if it contains the exclude string
    if ! awk '/import \(/{flag=1;next}/\)/{flag=0}flag' "$file" | grep -q "$EXCLUDE_STRING"; then
        echo "Processing $file"
        goimports-reviser "${FLAGS[@]}" "$file"
    else
        echo "Skipping $file due to exclusion string within import block"
    fi
}

export -f process_file

# Find all .go files, excluding specified patterns, and process them
find . -type f -name '*.go' \
    ! -path "./vendors/*" \
    ! -path "./ts-client/*" \
    ! -path "./.git/*" \
    ! -path "./.github/*" \
    ! -path "./bin/*" \
    ! -path "./docs/*" \
    ! -path "./localnet/*" \
    ! -path "./proto/*" \
    ! -path "./tools/*" \
    ! -name "*mock.go" \
    ! -name "*pb.go" \
    -exec bash -c 'process_file "$0"' {} \;
