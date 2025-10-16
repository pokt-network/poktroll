#!/bin/bash

show_help() {
    cat << EOF
🔐 Key Import Script 🔐

Usage: $0 <keyfile> [OPTIONS]

Import private keys from a file containing hex private keys and names.

⚠️  🚨 CRITICAL SECURITY WARNING 🚨 ⚠️
This script imports UNENCRYPTED PRIVATE KEYS from plain text format!
The private keys in your file are sensitive and should be:
- 🔒 Stored securely and privately
- 🗑️  Deleted after import if no longer needed
- 👀 Never shared or exposed publicly
- 💾 Backed up safely before import

ONLY use this script if you:
- 🧠 Fully understand the security implications
- 🔒 Are in a secure, private environment
- 🎯 Have verified the private keys are valid
- 💪 Trust the source of the private keys

❌ DO NOT use this script on shared/public computers!
❌ DO NOT use this script over remote connections!
❌ DO NOT leave private keys in plain text files!

Arguments:
    <keyfile> 📄   Path to file containing private keys and names

Options:
    -h, --help 🆘                    Show this help message
    --keyring-backend <backend>      Select keyring's backend (os|file|kwallet|pass|test|memory)
    --key-type <type>               Private key signing algorithm (default: secp256k1)

Examples:
    $0 keys.txt 📁
    $0 keys.txt --keyring-backend file 🗂️
    $0 keys.txt --keyring-backend os --key-type secp256k1 💻

The keyfile should contain one private key and name per line:
    a1b2c3d4e5f6... big-wallet2
    f6e5d4c3b2a1... eth-app3
    1a2b3c4d5e6f... company-gateway

🔒 Remember: Secure key management is critical! 🔒

EOF
}

# Initialize variables
KEYFILE=""
KEYRING_BACKEND=""
KEY_TYPE="secp256k1"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        --keyring-backend)
            if [[ -n "$2" ]]; then
                KEYRING_BACKEND="$2"
                shift 2
            else
                echo "❌ Error: --keyring-backend requires a value"
                exit 1
            fi
            ;;
        --key-type)
            if [[ -n "$2" ]]; then
                KEY_TYPE="$2"
                shift 2
            else
                echo "❌ Error: --key-type requires a value"
                exit 1
            fi
            ;;
        *)
            if [[ -z "$KEYFILE" ]]; then
                KEYFILE="$1"
                shift
            else
                echo "❌ Error: Unknown argument '$1'"
                echo "💡 Use -h or --help for usage information"
                exit 1
            fi
            ;;
    esac
done

# Check if file argument is provided
if [[ -z "$KEYFILE" ]]; then
    echo "❌ Error: No keyfile provided"
    echo "💡 Use -h or --help for usage information"
    exit 1
fi

# Check if file exists
if [[ ! -f "$KEYFILE" ]]; then
    echo "❌ Error: File '$KEYFILE' not found"
    exit 1
fi

# Check if file is readable
if [[ ! -r "$KEYFILE" ]]; then
    echo "❌ Error: File '$KEYFILE' is not readable"
    exit 1
fi

# Check dependencies
if ! command -v pocketd &> /dev/null; then
    echo "❌ Error: pocketd command not found"
    exit 1
fi

# Count total keys to import
total_keys=0
while IFS= read -r line || [[ -n "$line" ]]; do
    # Skip empty lines and lines starting with #
    if [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]]; then
        continue
    fi
    ((total_keys++))
done < "$KEYFILE"

echo "🔑 Importing $total_keys keys..."
echo "📁 Reading from: $KEYFILE"
if [[ -n "$KEYRING_BACKEND" ]]; then
    echo "🗂️  Using keyring backend: $KEYRING_BACKEND"
fi
echo "🔐 Using key type: $KEY_TYPE"
echo ""

# Initialize counters
imported_count=0
failed_count=0
skipped_count=0

# Read file and import each key
while IFS= read -r line || [[ -n "$line" ]]; do
    # Skip empty lines and lines starting with #
    if [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]]; then
        continue
    fi

    # Parse line: <private_key> <name>
    if [[ "$line" =~ ^([a-fA-F0-9]+)[[:space:]]+(.+)$ ]]; then
        private_key="${BASH_REMATCH[1]}"
        keyname="${BASH_REMATCH[2]}"

        # Trim whitespace from keyname
        keyname=$(echo "$keyname" | xargs)

        echo "🔑 Importing key: $keyname"

        # Build the import command with optional keyring backend
        import_cmd="pocketd keys import-hex \"$keyname\" \"$private_key\""
        if [[ -n "$KEYRING_BACKEND" ]]; then
            import_cmd="$import_cmd --keyring-backend $KEYRING_BACKEND"
        fi
        if [[ -n "$KEY_TYPE" ]]; then
            import_cmd="$import_cmd --key-type $KEY_TYPE"
        fi

        # Execute the import command
        if eval "$import_cmd" 2>/dev/null; then
            echo "✅ Successfully imported: $keyname"
            ((imported_count++))
        else
            echo "❌ Failed to import: $keyname"
            ((failed_count++))
        fi
        echo ""
    else
        echo "⚠️  Skipping invalid line format: $line"
        ((skipped_count++))
    fi
done < "$KEYFILE"

echo "========================================="
echo "📊 Import Summary:"
echo "✅ Successfully imported: $imported_count"
echo "❌ Failed imports: $failed_count"
echo "⚠️  Skipped lines: $skipped_count"
echo "📈 Total processed: $((imported_count + failed_count + skipped_count))"
echo "========================================="

# Exit with appropriate code
if [[ $failed_count -gt 0 ]]; then
    exit 1
else
    exit 0
fi