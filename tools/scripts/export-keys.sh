#!/bin/bash

show_help() {
    cat << EOF
🔐 Key Export Script 🔐

Usage: $0 <keyfile> [OPTIONS]

Export keys from a file containing a list of keynames or addresses.

⚠️  🚨 CRITICAL SECURITY WARNING 🚨 ⚠️
This script exports UNENCRYPTED PRIVATE KEYS in plain text format!
The exported keys will be displayed in your terminal and may be:
- 👀 Visible to anyone looking at your screen
- 📝 Logged in your shell history
- 💾 Saved in terminal scrollback buffers
- 🔍 Accessible to other processes on your system

ONLY use this script if you:
- 🧠 Fully understand the security implications
- 🔒 Are in a secure, private environment
- 🎯 Have a specific need for raw key material
- 💪 Are confident in handling private keys safely

❌ DO NOT use this script on shared/public computers!
❌ DO NOT use this script over remote connections!
❌ DO NOT leave exported keys in terminal history!

Arguments:
    <keyfile> 📄   Path to file containing keynames or addresses (one per line)

Options:
    -h, --help 🆘                    Show this help message
    --keyring-backend <backend>      Select keyring's backend (os|file|kwallet|pass|test|memory)
    --output <format> 📊             Output format (raw|file)
                                     • raw: Private keys only, one per line
                                     • file: Raw keys to file (requires --file)
    -f, --file <path> 📝             Output file path (required for file)

Examples:
    $0 keys.txt 📁
    $0 keys.txt --output raw 🔢
    $0 keys.txt --output file --file exported_keys.txt 📄
    $0 keys.txt --keyring-backend file --output raw 🗂️

The keyfile should contain one keyname per line, e.g.:
    big-wallet2
    eth-app3
    company-gateway

🔒 Remember: With great keys comes great responsibility! 🔒

EOF
}

# Initialize variables
KEYFILE=""
KEYRING_BACKEND=""
OUTPUT_FORMAT="default"
OUTPUT_FILE=""

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
        --output)
            if [[ -n "$2" ]]; then
                case "$2" in
                    raw|file)
                        OUTPUT_FORMAT="$2"
                        shift 2
                        ;;
                    *)
                        echo "❌ Error: Invalid output format '$2'. Valid options: raw, file"
                        exit 1
                        ;;
                esac
            else
                echo "❌ Error: --output requires a value (raw|file)"
                exit 1
            fi
            ;;
        -f|--file)
            if [[ -n "$2" ]]; then
                OUTPUT_FILE="$2"
                shift 2
            else
                echo "❌ Error: --file requires a file path"
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

# Validate file output requirements
if [[ "$OUTPUT_FORMAT" == "file" ]]; then
    if [[ -z "$OUTPUT_FILE" ]]; then
        echo "❌ Error: --file is required when using --output file"
        exit 1
    fi
fi

# Clear output file if using file output mode
if [[ "$OUTPUT_FORMAT" == "file" ]]; then
    > "$OUTPUT_FILE"
    echo "📝 Writing keys to: $OUTPUT_FILE"
fi

# Count total keys to export
total_keys=0
while IFS= read -r line || [[ -n "$line" ]]; do
    # Skip empty lines and lines starting with #
    if [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]]; then
        continue
    fi
    ((total_keys++))
done < "$KEYFILE"

# Show summary for raw output
if [[ "$OUTPUT_FORMAT" == "raw" ]]; then
    echo "🔑 Exporting $total_keys keys..." >&2
fi

# Read file and export each key
while IFS= read -r keyname || [[ -n "$keyname" ]]; do
    # Skip empty lines and lines starting with #
    if [[ -z "$keyname" || "$keyname" =~ ^[[:space:]]*# ]]; then
        continue
    fi

    # Trim whitespace
    keyname=$(echo "$keyname" | xargs)

    # Show progress for non-raw modes
    if [[ "$OUTPUT_FORMAT" != "raw" ]]; then
        echo "🔑 Exporting key: $keyname"
    fi

    # Build the export command with optional keyring backend
    export_cmd="pocketd keys export \"$keyname\" --unsafe --unarmored-hex --yes"
    if [[ -n "$KEYRING_BACKEND" ]]; then
        export_cmd="$export_cmd --keyring-backend $KEYRING_BACKEND"
    fi

    # Execute the export command and capture output
    private_key=$(eval "$export_cmd" 2>/dev/null)

    # Check if export was successful
    if [[ $? -ne 0 || -z "$private_key" ]]; then
        echo "❌ Failed to export key: $keyname" >&2
        continue
    fi

    # Handle different output formats
    case "$OUTPUT_FORMAT" in
        "raw")
            echo "$private_key"
            ;;
        "file")
            echo "$private_key" >> "$OUTPUT_FILE"
            ;;
        "default")
            # Default behavior - show the full pocketd output
            eval "$export_cmd"
            ;;
    esac
done < "$KEYFILE"
