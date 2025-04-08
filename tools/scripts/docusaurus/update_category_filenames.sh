#!/bin/bash
#
# update_category_filenames.sh - Rename Docusaurus markdown files based on sidebar_position
#
# This script renames markdown files in a Docusaurus documentation directory by:
#   1. Reading the sidebar_position from the frontmatter of each file
#   2. Converting filenames from kebab-case to snake_case
#   3. Adding the position number as a prefix to each filename
#
# Requirements:
#   - Directory must contain a _category_.json file (verifies it's a valid category)
#   - Each markdown file must have a sidebar_position specified in its frontmatter
#
# Usage:
#   tools/scripts/docusaurus/update_category_filenames.sh PATH_TO_DIRECTORY
#
# Example:
#   tools/scripts/docusaurus/update_category_filenames.sh ./docusaurus/docs/tools/user_guide/
#
# Before:
#   _category_.json
#   app-transfer.md
#   check-balance.md
#   create-new-wallet.md
#   pocketd_cli.md
#   recover-with-mnemonic.md
#   send-tokens.md
#   user_keyring.md
#
# After:
#   _category_.json
#   1_pocketd_cli.md
#   2_create_new_wallet.md
#   3_recover_with_mnemonic.md
#   4_check_balance.md
#   5_send_tokens.md
#   6_app_transfer.md
#

# Check if path is provided
if [ -z "$1" ]; then
    echo "Error: Please provide a directory path"
    exit 1
fi

# Path to the directory
dir_path="$1"

# Check if directory exists
if [ ! -d "$dir_path" ]; then
    echo "Error: Directory not found: $dir_path"
    exit 1
fi

# Check if _category_.json exists
if [ ! -f "$dir_path/_category_.json" ]; then
    echo "Error: _category_.json file not found in $dir_path"
    exit 1
fi

# Process each markdown file
for file in "$dir_path"/*.md; do
    # Skip if not a regular file
    if [ ! -f "$file" ]; then
        continue
    fi

    # Get the filename without path
    filename=$(basename "$file")

    # Skip processing already renamed files (files starting with a number followed by underscore)
    if [[ $filename =~ ^[0-9]+_ ]]; then
        continue
    fi

    # Extract the sidebar_position
    position=$(grep -m 1 "sidebar_position:" "$file" | sed 's/sidebar_position: *//')

    # Check if sidebar_position was found
    if [ -z "$position" ]; then
        echo "Error: sidebar_position not found in $filename"
        continue
    fi

    # Convert dash to underscore and extract basename without extension
    base_name=$(basename "$filename" .md | tr '-' '_')

    # Create new filename with position prefix
    new_filename="${position}_${base_name}.md"

    # Rename the file
    mv "$file" "$dir_path/$new_filename"
    echo "Renamed: $filename -> $new_filename"
done

echo "Processing complete."
