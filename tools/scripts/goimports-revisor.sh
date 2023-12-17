#!/usr/bin/env bash

FLAGS=(
    -rm-unused
    -use-cache
    -imports-order "std,general,company,project,blanked,dotted"
    -project-name "github.com/pokt-network/poktrolld"
)

# Define a function to check for the exclusion comment and run goimports-reviser
process_file() {
    local file=$1
    if ! grep -q "//go:build ignore" "$file"; then
        echo "Processing $file"
        goimports-reviser "${FLAGS[@]}" "$file"
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
