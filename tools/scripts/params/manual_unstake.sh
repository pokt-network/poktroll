#!/bin/bash

# Initialize variables
OUTPUT_FILE=""
JSON_FILE=""
VALIDATOR_LIST=""

# Parse command line options
while getopts "o:h" opt; do
    case $opt in
    o)
        OUTPUT_FILE="$OPTARG"
        ;;
    h)
        show_help=true
        ;;
    \?)
        echo "Invalid option: -$OPTARG" >&2
        exit 1
        ;;
    esac
done

# Shift to get remaining arguments
shift $((OPTIND - 1))

# Check if correct number of arguments provided
if [ $# -ne 2 ] || [ "$show_help" = true ]; then
    echo ""
    echo "###############################################################################"
    echo "###                      Morse Manual Unstaker                            ###"
    echo "###############################################################################"
    echo ""
    echo "DESCRIPTION:"
    echo "    The Morse Manual Unstaker processes morse account state JSON files."
    echo "    It manually unstakes specified accounts by:"
    echo "    - Setting supplier_stake to 0"
    echo "    - Setting application_stake to 0"
    echo "    - Setting unstaked_balance to the sum of original values"
    echo ""
    echo "USAGE:"
    echo "    $0 [-o output_file] <path_to_json_file> <comma_separated_validator_list>"
    echo ""
    echo "OPTIONS:"
    echo "    -o output_file              Specify custom output file path"
    echo "    -h                          Show this help message"
    echo ""
    echo "PARAMETERS:"
    echo "    path_to_json_file           Path to the JSON file to process"
    echo "    comma_separated_validator_list   List of morse_src_address values to update"
    echo ""
    echo "EXAMPLE:"
    echo "    $0 tools/scripts/migration/msg_import_morse_accounts_165497_2025-04-15.json \\"
    echo "       'c409a9e0d1be8780fe0b29dcdf72f8a879fb110c,08e5727cd7fbc4bc97ef3246da7379043f949f70,278654d9daf0e0be2c4e4da5a26c3b4149c5f6d0,81522de7711246fca147a34173dd2a462dc77a5a,c86b27e72c32b64db3eae137ffa84fec007a9062,79cbe645f2b4fa767322faf59a0093e6b73a2383,a86b6a5517630a23aec3dc4e3479a5818c575ac2,882f3f23687a9f3dddf6c65d66e9e3184ca67573,96f2c414b6f3afbba7ba571b7de360709d614e62,05db988509a25dd812dfd1a421cbf47078301a16'"
    echo ""
    echo "    With custom output:"
    echo "    $0 -o unstaked_accounts.json tools/scripts/migration/msg_import_morse_accounts_165497_2025-04-15.json \\"
    echo "       'c409a9e0d1be8780fe0b29dcdf72f8a879fb110c,08e5727cd7fbc4bc97ef3246da7379043f949f70'"
    echo ""
    echo "NOTES:"
    echo "    - A timestamped backup will be created automatically"
    echo "    - Requires 'jq' to be installed for JSON processing"
    echo "    - Validator addresses are case-insensitive"
    echo "    - If -o is not specified, creates output file with '_unstaked' suffix"
    echo "    - Missing validator addresses will be logged as errors"
    echo ""
    echo "###############################################################################"
    echo ""
    exit 1
fi

JSON_FILE="$1"
VALIDATOR_LIST="$2"

# Determine output file
if [ -z "$OUTPUT_FILE" ]; then
    # Extract directory, filename, and extension
    DIR=$(dirname "$JSON_FILE")
    BASENAME=$(basename "$JSON_FILE" .json)
    OUTPUT_FILE="${DIR}/${BASENAME}_unstaked.json"
fi

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

# Copy original to output file for processing
cp "$JSON_FILE" "$OUTPUT_FILE"
echo "Processing into output file: $OUTPUT_FILE"

# Convert comma-separated list to array
IFS=',' read -ra VALIDATORS <<<"$VALIDATOR_LIST"

# Create a temporary file for processing
TEMP_FILE=$(mktemp)

# Track processed and missing validators
PROCESSED_COUNT=0
MISSING_VALIDATORS=()

# Process each validator
for validator in "${VALIDATORS[@]}"; do
    # Trim whitespace and convert to lowercase for comparison
    validator=$(echo "$validator" | xargs | tr '[:upper:]' '[:lower:]')
    echo "Processing validator: $validator"

    # Check if validator exists in the file
    VALIDATOR_EXISTS=$(jq --arg validator "$validator" '
        .morse_account_state.accounts |
        any(.morse_src_address | ascii_downcase == $validator)
    ' "$OUTPUT_FILE")

    if [ "$VALIDATOR_EXISTS" = "false" ]; then
        echo "ERROR: Validator $validator not found in the JSON file"
        MISSING_VALIDATORS+=("$validator")
        continue
    fi

    # Use jq to find and modify the account
    jq --arg validator "$validator" '
        .morse_account_state.accounts |= map(
            if (.morse_src_address | ascii_downcase) == $validator then
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
    ' "$OUTPUT_FILE" >"$TEMP_FILE"

    # Check if jq command was successful
    if [ $? -eq 0 ]; then
        mv "$TEMP_FILE" "$OUTPUT_FILE"
        echo "Successfully processed validator: $validator"
        ((PROCESSED_COUNT++))
    else
        echo "Error processing validator: $validator"
        rm -f "$TEMP_FILE"
        exit 1
    fi
done

# Clean up
rm -f "$TEMP_FILE"

echo ""
echo "==============================================================================="
echo "Processing complete!"
echo "Original file backed up to: $BACKUP_FILE"
echo "Output file: $OUTPUT_FILE"
echo "Processed validators: $PROCESSED_COUNT"

# Report missing validators
if [ ${#MISSING_VALIDATORS[@]} -gt 0 ]; then
    echo ""
    echo "WARNING: The following validators were not found:"
    for missing in "${MISSING_VALIDATORS[@]}"; do
        echo "  - $missing"
    done
fi

# Optional: Show summary of changes
echo ""
echo "Summary of processed accounts:"
for validator in "${VALIDATORS[@]}"; do
    validator=$(echo "$validator" | xargs | tr '[:upper:]' '[:lower:]')

    # Skip if validator was missing
    if [[ " ${MISSING_VALIDATORS[@]} " =~ " $validator " ]]; then
        continue
    fi

    echo "=== $validator ==="
    jq --arg validator "$validator" '
        .morse_account_state.accounts[] |
        select((.morse_src_address | ascii_downcase) == $validator) |
        {
            morse_src_address: .morse_src_address,
            unstaked_balance: .unstaked_balance.amount,
            supplier_stake: .supplier_stake.amount,
            application_stake: .application_stake.amount
        }
    ' "$OUTPUT_FILE"
    echo ""
done
