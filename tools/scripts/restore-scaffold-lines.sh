#!/usr/bin/env bash

# The line to be restored
RESTORE_LINE="// this line is used by starport scaffolding # root/moduleImport"

# The pattern after which the line should be restored (modify this as needed)
PATTERN="import ("

# Function to restore the line in a file
restore_line() {
    local file=$1
    local temp_file=$(mktemp)

    # Flag to indicate if the line has been restored
    local restored=0

    # Read the file line by line
    while IFS= read -r line; do
        echo "$line" >> "$temp_file"
        # Check if the line matches the pattern
        if [[ $line == *"$PATTERN"* ]]; then
            # Add the RESTORE_LINE after the pattern
            echo "$RESTORE_LINE" >> "$temp_file"
            restored=1
        fi
    done < "$file"

    # Replace the original file with the temp file
    mv "$temp_file" "$file"
}

# Process each file changed in the git diff
git diff --name-only | while IFS= read -r file; do
    # Check if the file is a .go file and the specific line was deleted
    if [[ $file == *.go ]] && git diff "$file" | grep -q "^\-.*$RESTORE_LINE"; then
        echo "Restoring line in $file"
        restore_line "$file"
    fi
done
