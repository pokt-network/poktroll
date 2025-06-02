#!/bin/bash

# Initialize variables
OUTPUT_FILE=""
JSON_FILE=""
VALIDATOR_LIST=""
IN_PLACE=false
NO_BACKUP=false

# Parse command line options
while getopts "o:ihb" opt; do
    case $opt in
    o)
        OUTPUT_FILE="$OPTARG"
        ;;
    i)
        IN_PLACE=true
        ;;
    b)
        NO_BACKUP=true
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
    echo "###                      Morse Manual Unstaker (Fast)                     ###"
    echo "###############################################################################"
    echo ""
    echo "DESCRIPTION:"
    echo "    Fast version that processes all validators in a single jq pass"
    echo ""
    echo "USAGE:"
    echo "    $0 [-o output_file] [-i] [-b] <path_to_json_file> <comma_separated_validator_list>"
    echo ""
    echo "OPTIONS:"
    echo "    -o output_file              Specify custom output file path"
    echo "    -i                          Modify the original file in-place"
    echo "    -b                          Skip creating backup files"
    echo "    -h                          Show this help message"
    echo ""
    echo "###############################################################################"
    echo ""
    exit 1
fi

JSON_FILE="$1"
VALIDATOR_LIST="$2"

# Determine output file
if [ "$IN_PLACE" = true ]; then
    OUTPUT_FILE="$JSON_FILE"
elif [ -z "$OUTPUT_FILE" ]; then
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
    echo "Error: jq is required but not installed."
    exit 1
fi

# Create backup (unless skipped or doing in-place without backup)
BACKUP_FILE=""
if [ "$NO_BACKUP" != true ] && [ "$IN_PLACE" = true ]; then
    BACKUP_FILE="${JSON_FILE}.backup.$(date +%Y%m%d_%H%M%S)"
    cp "$JSON_FILE" "$BACKUP_FILE"
    echo "Created backup: $BACKUP_FILE"
elif [ "$NO_BACKUP" != true ] && [ "$IN_PLACE" != true ]; then
    BACKUP_FILE="${JSON_FILE}.backup.$(date +%Y%m%d_%H%M%S)"
    cp "$JSON_FILE" "$BACKUP_FILE"
    echo "Created backup: $BACKUP_FILE"
fi

# Convert comma-separated list to space-separated for bash array
IFS=',' read -ra VALIDATORS <<<"$VALIDATOR_LIST"

# Clean up validators (trim whitespace, convert to lowercase)
CLEANED_VALIDATORS=()
for validator in "${VALIDATORS[@]}"; do
    cleaned=$(echo "$validator" | xargs | tr '[:upper:]' '[:lower:]')
    CLEANED_VALIDATORS+=("$cleaned")
done

echo "Processing ${#CLEANED_VALIDATORS[@]} validators in a single pass..."

# Build JSON array of validators for jq
VALIDATORS_JSON="["
for i in "${!CLEANED_VALIDATORS[@]}"; do
    if [ $i -gt 0 ]; then
        VALIDATORS_JSON+=","
    fi
    VALIDATORS_JSON+="\"${CLEANED_VALIDATORS[$i]}\""
done
VALIDATORS_JSON+="]"

# For in-place modification, we need to use a temp file
if [ "$IN_PLACE" = true ]; then
    TEMP_FILE="${JSON_FILE}.tmp"
else
    TEMP_FILE="$OUTPUT_FILE"
fi

# Execute single jq command with validators as argument
jq --argjson validators "$VALIDATORS_JSON" '
.morse_account_state.accounts |= map(
    if (.morse_src_address | ascii_downcase) as $addr | ($validators | index($addr)) then
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
)' "$JSON_FILE" >"$TEMP_FILE"

if [ $? -ne 0 ]; then
    echo "Error: jq processing failed!"
    rm -f "$TEMP_FILE"
    exit 1
fi

# If doing in-place modification, move temp file to original
if [ "$IN_PLACE" = true ]; then
    mv "$TEMP_FILE" "$JSON_FILE"
    echo "File modified in-place: $JSON_FILE"
fi

# Count processed and missing validators in a single jq call
echo "Checking results..."
RESULTS=$(jq --argjson validators "$VALIDATORS_JSON" '
{
    "processed": [
        .morse_account_state.accounts[] |
        select((.morse_src_address | ascii_downcase) as $addr | ($validators | index($addr))) |
        select(.supplier_stake.amount == "0" and .application_stake.amount == "0") |
        .morse_src_address | ascii_downcase
    ],
    "missing": ($validators - [
        .morse_account_state.accounts[] |
        .morse_src_address | ascii_downcase
    ])
}' "$OUTPUT_FILE")

PROCESSED_COUNT=$(echo "$RESULTS" | jq -r '.processed | length')
MISSING_VALIDATORS=($(echo "$RESULTS" | jq -r '.missing[]' 2>/dev/null))

echo ""
echo "==============================================================================="
echo "Processing complete!"
if [ -n "$BACKUP_FILE" ]; then
    echo "Original file backed up to: $BACKUP_FILE"
fi
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

echo ""
echo "Processing completed in a single jq pass - much faster than the original!"
