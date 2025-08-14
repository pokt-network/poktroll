#!/bin/bash

# Monitor RelayMiner responsiveness

show_help() {
    cat << EOF
Usage: $(basename "$0") [OPTIONS]

Monitor RelayMiner responsiveness by periodically sending RelayRequests
(using 'pocketd relayminer relay') and extracting the block number.

OPTIONS:
    --application <address>          (required) Staked application address (pokt1...)
    --supplier <address>             (required) Staked supplier address (pokt1...)
    --node <rpc-url>                 (required) Cosmos node RPC URL (e.g. tcp://127.0.0.1:26657)
    --grpc-addr <host:port>          (required) gRPC endpoint for pocketd (e.g. localhost:9090)
    --supplier-endpoint-override <url> (required) Supplier public endpoint override (HTTP endpoint the relay is sent to)
    --help                           Show this help and exit

EXAMPLE:
    $(basename "$0") \\
        --application pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4 \\
        --supplier pokt19a3t4yunp0dlpfjrp7qwnzwlrzd5fzs2gjaaaj \\
        --node tcp://127.0.0.1:26657 --grpc-addr localhost:9090 \\
        --supplier-endpoint-override http://localhost:8584
EOF
    exit 0
}

# -----------------------------
# Argument parsing
# -----------------------------
APP=""
SUPPLIER=""
NODE=""
GRPC_ADDR=""
ENDPOINT=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --application) APP="$2"; shift 2 ;;
    --supplier) SUPPLIER="$2"; shift 2 ;;
    --node) NODE="$2"; shift 2 ;;
    --grpc-addr) GRPC_ADDR="$2"; shift 2 ;;
    --supplier-endpoint-override) ENDPOINT="$2"; shift 2 ;;
    --help|-h) show_help ;;
    *) echo "Unknown option: $1"; show_help ;;
  esac
done

# Validate required
missing=()
[[ -z "$APP" ]] && missing+=("--application")
[[ -z "$SUPPLIER" ]] && missing+=("--supplier")
[[ -z "$NODE" ]] && missing+=("--node")
[[ -z "$GRPC_ADDR" ]] && missing+=("--grpc-addr")
[[ -z "$ENDPOINT" ]] && missing+=("--supplier-endpoint-override")
if (( ${#missing[@]} )); then
  echo "Missing required options: ${missing[*]}" >&2
  echo "Use --help for usage." >&2
  exit 1
fi

# Hardcoded payload & interval
PAYLOAD='{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}'
INTERVAL=0.5

# Helper: extract block hex from mixed relay output (raw JSON or logged line)
extract_block_hex() {
  local input="$1" blk=""
  # 1. Standard JSON: "result":"0x12ab"
  blk=$(echo "$input" | grep -Eo '"result":"0x[0-9a-fA-F]+"' | tail -n1 | sed -E 's/.*"result":"(0x[0-9a-fA-F]+)".*/\1/')
  if [[ -n "$blk" ]]; then
    echo "$blk"; return 0
  fi
  # 2. Logged line: result' (string): 0x12ab or ‘result’ (string): 0x12ab
  blk=$(echo "$input" | grep -Eo "result['’] \(string\): 0x[0-9a-fA-F]+" | awk '{print $NF}' | tail -n1)
  if [[ -n "$blk" ]]; then
    echo "$blk"; return 0
  fi
  return 1
}

echo "Starting RelayMiner monitoring"
echo "Application Address=$APP"
echo "Supplier Address=$SUPPLIER"
echo "Node RPC=$NODE"
echo "Node gRPC=$GRPC_ADDR"
echo "Supplier Endpoint Override=$ENDPOINT"
echo "Press Ctrl+C to stop"
echo ""

# Initialize variables for delta calculation
last_request_time_ns=0

while true; do
    # Get current timestamp (ns precision) & human timestamp
    current_time_ns=$(date +%s%N)
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')

    # Calculate delta without bc
    if [[ "$last_request_time_ns" != "0" ]]; then
        diff_ns=$(( current_time_ns - last_request_time_ns ))
        diff_ms=$(( diff_ns / 1000000 ))
        sec_int=$(( diff_ms / 1000 ))
        sec_frac=$(( (diff_ms % 1000) / 10 ))  # two decimals
        delta_formatted="${sec_int}.$(printf "%02d" "$sec_frac")"

        if (( diff_ms <= 1000 )); then
            delta_color="\033[32m"; delta_status="⚡"
        elif (( diff_ms <= 4000 )); then
            delta_color="\033[33m"; delta_status="⏱️ "
        elif (( diff_ms <= 10000 )); then
            delta_color="\033[31m"; delta_status="🐌"
        else
            delta_color="\033[31m"; delta_status="🚫"
        fi
        reset_color="\033[0m"
        delta_display=" ${delta_color}[Δ${delta_formatted}s ${delta_status}]${reset_color}"
    else
        delta_display=""
    fi

    # Execute relay command
    response=$(pocketd relayminer relay \
      --app="$APP" \
      --supplier="$SUPPLIER" \
      --node="$NODE" \
      --grpc-addr="$GRPC_ADDR" \
      --grpc-insecure=false \
      --payload="$PAYLOAD" \
      --supplier-public-endpoint-override="$ENDPOINT" 2>&1
    )
    relay_exit_code=$?

    if [ $relay_exit_code -ne 0 ]; then
        echo -e "❌ [$timestamp] ERROR: relay command failed (exit $relay_exit_code): $response$delta_display"
    else
        if echo "$response" | grep -q '"error"'; then
            echo -e "❌ [$timestamp] API ERROR: $response$delta_display"
        else
            # NEW robust extraction
            block_hex=$(extract_block_hex "$response")
            if [ -z "$block_hex" ]; then
                echo -e "❌ [$timestamp] PARSE ERROR: Could not extract block number: $response$delta_display"
            else
                if block_decimal=$((16#${block_hex#0x})) 2>/dev/null; then
                    echo -e "✅ [$timestamp] Block: $block_decimal (hex: $block_hex)$delta_display"
                else
                    echo -e "❌ [$timestamp] CONVERSION ERROR: hex $block_hex$delta_display"
                fi
            fi
        fi
    fi

    last_request_time_ns=$current_time_ns
done