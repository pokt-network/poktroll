#!/bin/bash

# Check if correct number of arguments provided
if [ $# -ne 2 ]; then
    echo "Usage: $0 <path_to_json_file> <comma_separated_validator_list>"
    echo "Example: $0 accounts.json 'C409A9E0D1BE8780FE0B29DCDF72F8A879FB110C,08e5727cd7fbc4bc97ef3246da7379043f949f70'"
    exit 1
fi

JSON_FILE="$1"
VALIDATOR_LIST="$2"

# Check if JSON file exists
if [ ! -f "$JSON_FILE" ]; then
    echo "Error: JSON file '$JSON_FILE' not found!"
    exit 1
fi

# Check if jq is installed
if ! command -v jq &>/dev/null; then
    echo "Error: jq is required but not installed. Please install jq first."
    echo "On Ubuntu/Debian: sudo apt-get install jq"
    echo "On macOS: brew install jq"
    exit 1
fi

# Create backup of original file
BACKUP_FILE="${JSON_FILE}.backup.$(date +%Y%m%d_%H%M%S)"
cp "$JSON_FILE" "$BACKUP_FILE"
echo "Created backup: $BACKUP_FILE"

# Convert comma-separated list to array
IFS=',' read -ra VALIDATORS <<<"$VALIDATOR_LIST"

# Create a temporary file for processing
TEMP_FILE=$(mktemp)

# Process each validator
for validator in "${VALIDATORS[@]}"; do
    # Trim whitespace
    validator=$(echo "$validator" | xargs)
    echo "Processing validator: $validator"

    # Use jq to find and modify the account
    jq --arg validator "$validator" '
        .morse_account_state.accounts |= map(
            if .morse_src_address == $validator then
                . as $account |
                ($account.unstaked_balance.amount | tonumber) as $unstaked |
                ($account.supplier_stake.amount | tonumber) as $supplier |
                ($account.application_stake.amount | tonumber) as $application |
                ($unstaked + $supplier + $application) as $new_balance |
                .unstaked_balance.amount = ($new_balance | tostring) |
                .supplier_stake.amount = "0" |
                .application_stake.amount = "0"
            else
                .
            end
        )
    ' "$JSON_FILE" >"$TEMP_FILE"

    # Check if jq command was successful
    if [ $? -eq 0 ]; then
        mv "$TEMP_FILE" "$JSON_FILE"
        echo "Successfully processed validator: $validator"
    else
        echo "Error processing validator: $validator"
        rm -f "$TEMP_FILE"
        exit 1
    fi
done

# Clean up
rm -f "$TEMP_FILE"

echo "Processing complete!"
echo "Original file backed up to: $BACKUP_FILE"
echo "Modified file: $JSON_FILE"

# Optional: Show summary of changes
echo ""
echo "Summary of processed accounts:"
for validator in "${VALIDATORS[@]}"; do
    validator=$(echo "$validator" | xargs)
    echo "=== $validator ==="
    jq --arg validator "$validator" '
        .morse_account_state.accounts[] |
        select(.morse_src_address == $validator) |
        {
            morse_src_address: .morse_src_address,
            unstaked_balance: .unstaked_balance.amount,
            supplier_stake: .supplier_stake.amount,
            application_stake: .application_stake.amount
        }
    ' "$JSON_FILE"
    echo ""
done
