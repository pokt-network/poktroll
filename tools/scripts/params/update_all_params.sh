#!/bin/bash

# Parameter Update Script - Local, Alpha, Beta, and Main Network Support
# Usage: ./update_params.sh [local|alpha|beta|main] [--help]
#
# This script updates all parameter files for either local, alpha, beta, or main network
# See ./tools/scripts/params/gov_params.sh for additional governance parameters

set -e

# Default configuration
DEFAULT_ENV="beta"
PARAM_DIR="tools/scripts/params/bulk_params"
GOV_PARAM_SCRIPT="./tools/scripts/params/gov_params.sh"

# Parameter files to update
POCKET_PARAM_FILES=(
  application_params.json
  gateway_params.json
  proof_params.json
  service_params.json
  session_params.json
  shared_params.json
  supplier_params.json
  tokenomics_params.json
  slashing_params.json
)

COSMOS_PARAM_FILES=(
  mint_params.json
  staking_params.json
)

get_local_config() {
  case "$1" in
  from) echo "pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw" ;;
  keyring_backend) echo "test" ;;
  chain_id) echo "pocket" ;;
  node) echo "http://localhost:26657" ;;
  home) echo "./localnet/poktrolld" ;;
  fees) echo "1000upokt" ;;
  esac
}

get_alpha_config() {
  case "$1" in
  from) echo "pokt1r6ja6rz6rpae58njfrsgs5n5sp3r36r2q9j04h" ;;
  keyring_backend) echo "test" ;;
  chain_id) echo "pocket-alpha" ;;
  node) echo "https://shannon-testnet-grove-rpc.alpha.poktroll.com" ;;
  home) echo "~/.pocket_prod" ;;
  fees) echo "200upokt" ;;
  esac
}

get_beta_config() {
  case "$1" in
  from) echo "pokt1f0c9y7mahf2ya8tymy8g4rr75ezh3pkklu4c3e" ;;
  keyring_backend) echo "test" ;;
  chain_id) echo "pocket-beta" ;;
  node) echo "https://shannon-testnet-grove-rpc.beta.poktroll.com" ;;
  home) echo "~/.pocket_prod" ;;
  fees) echo "200upokt" ;;
  esac
}

get_main_config() {
  case "$1" in
  from) echo "pokt18808wvw0h4t450t06uvauny8lvscsxjfyua7vh" ;;
  keyring_backend) echo "test" ;;
  chain_id) echo "pocket" ;;
  node) echo "https://shannon-grove-rpc.mainnet.poktroll.com" ;;
  home) echo "~/.pocket_prod" ;;
  fees) echo "1000upokt" ;;
  esac
}

show_help() {
  cat <<EOF
Parameter Update Script

USAGE:
  $0 [ENVIRONMENT] [OPTIONS]

ENVIRONMENTS:
  local   Update parameters on localnet
  alpha   Update parameters on alpha network
  beta    Update parameters on beta network (default)
  main    Update parameters on main network

OPTIONS:
  -h, --help              Show this help message
  -n, --dry-run           Show commands that would be executed without running them
  -v, --verbose           Enable verbose output
  --gov-only              Only run governance parameters (from gov_params.sh)
  --state-only            Only run state shift parameters (default behavior)
  --param-dir=DIR         Override parameter directory (default: $PARAM_DIR)

EXAMPLES:
  $0 beta                                                   # Update beta network with default params
  $0 main --param-dir=tools/scripts/params/bulk_params_main # Update main network with main-specific params
  $0 beta --param-dir=tools/scripts/params/bulk_params_beta # Update beta network with beta-specific params
  $0 beta --dry-run                                         # Show what would be executed on beta
  $0 main --gov-only                                        # Run only governance params on main
  $0 local --param-dir=./custom_params                      # Use custom parameter directory

PARAMETER FILES:
  State shift parameters from: [specified directory]
  Pocket parameters:
$(printf "  - %s\n" "${POCKET_PARAM_FILES[@]}")
  Cosmos SDK parameters:
$(printf "  - %s\n" "${COSMOS_PARAM_FILES[@]}")

  Governance parameters: $GOV_PARAM_SCRIPT
  (See that script for additional governance parameter details)

CONFIGURATION:
  Local Network
    - Chain ID: $(get_local_config chain_id)
    - Node: $(get_local_config node)
    - From: $(get_local_config from)
    - Fees: $(get_local_config fees)

  Alpha Network:
    - Chain ID: $(get_alpha_config chain_id)
    - Node: $(get_alpha_config node)
    - From: $(get_alpha_config from)
    - Fees: $(get_alpha_config fees)

  Beta Network:
    - Chain ID: $(get_beta_config chain_id)
    - Node: $(get_beta_config node)
    - From: $(get_beta_config from)
    - Fees: $(get_beta_config fees)

  Main Network:
    - Chain ID: $(get_main_config chain_id)
    - Node: $(get_main_config node)
    - From: $(get_main_config from)
    - Fees: $(get_main_config fees)

NOTES:
  - Ensure you have the correct keyring access for the target environment
  - All parameter files must exist in the specified directory
  - Use --dry-run to verify commands before execution
  - For governance parameters, see $GOV_PARAM_SCRIPT for details
  - You will be prompted for confirmation before executing transactions

EOF
}

log() {
  if [[ "${VERBOSE:-false}" == "true" ]]; then
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $*" >&2
  fi
}

error() {
  echo "ERROR: $*" >&2
  exit 1
}

validate_environment() {
  local env="$1"
  case "$env" in
  local | alpha | beta | main)
    return 0
    ;;
  *)
    error "Invalid environment: $env. Use 'local', 'alpha', 'beta', or 'main'"
    ;;
  esac
}

validate_files() {
  log "Validating parameter files..."

  if [[ ! -d "$PARAM_DIR" ]]; then
    error "Parameter directory not found: $PARAM_DIR"
  fi

  local missing_files=()
  for file in "${POCKET_PARAM_FILES[@]}" "${COSMOS_PARAM_FILES[@]}"; do
    if [[ ! -f "$PARAM_DIR/$file" ]]; then
      missing_files+=("$file")
    fi
  done

  if [[ ${#missing_files[@]} -gt 0 ]]; then
    error "Missing parameter files: ${missing_files[*]}"
  fi

  if [[ "${GOV_ONLY:-false}" == "true" && ! -f "$GOV_PARAM_SCRIPT" ]]; then
    error "Governance parameter script not found: $GOV_PARAM_SCRIPT"
  fi

  log "All parameter files validated successfully"
}

get_config_value() {
  local env="$1"
  local key="$2"

  case "$env" in
  local)
    get_local_config "$key"
    ;;
  alpha)
    get_alpha_config "$key"
    ;;
  beta)
    get_beta_config "$key"
    ;;
  main)
    get_main_config "$key"
    ;;
  esac
}

run_command() {
  local cmd="$1"

  if [[ "${DRY_RUN:-false}" == "true" ]]; then
    echo "[DRY RUN] $cmd"
  else
    log "Executing: $cmd"
    eval "$cmd"
  fi
}

prompt_confirmation() {
  local env="$1"
  local mode="$2"

  echo ""
  echo "========================================="
  echo "⚠️  TRANSACTION CONFIRMATION REQUIRED ⚠️"
  echo "========================================="
  echo "Environment: $env"
  echo "Mode: $mode"
  echo "Parameter Directory: $PARAM_DIR"
  echo "Chain ID: $(get_config_value "$env" "chain_id")"
  echo "Node: $(get_config_value "$env" "node")"
  echo "From Address: $(get_config_value "$env" "from")"
  echo ""

  if [[ "$mode" == "state" || "$mode" == "both" ]]; then
    echo "Files to process:"
    for file in "${POCKET_PARAM_FILES[@]}" "${COSMOS_PARAM_FILES[@]}"; do
      echo "  - $PARAM_DIR/$file"
    done
  fi

  if [[ "$mode" == "gov" || "$mode" == "both" ]]; then
    echo "Governance script: $GOV_PARAM_SCRIPT"
  fi

  echo ""
  echo "This will execute blockchain transactions that cannot be undone."
  echo "========================================="

  while true; do
    read -p "Do you want to proceed? (yes/no): " yn
    case $yn in
    [Yy]es | [Yy])
      echo "Proceeding with parameter updates..."
      echo ""
      break
      ;;
    [Nn]o | [Nn])
      echo "Operation cancelled by user."
      exit 0
      ;;
    *)
      echo "Please answer 'yes' or 'no'."
      ;;
    esac
  done
}

update_state_params() {
  local env="$1"

  echo "Updating state shift parameters for $env network..."

  local from=$(get_config_value "$env" "from")
  local keyring_backend=$(get_config_value "$env" "keyring_backend")
  local chain_id=$(get_config_value "$env" "chain_id")
  local node=$(get_config_value "$env" "node")
  local home=$(get_config_value "$env" "home")
  local fees=$(get_config_value "$env" "fees")

  echo "Processing Pocket parameters..."
  for file in "${POCKET_PARAM_FILES[@]}"; do
    echo "Processing $file..."

    local cmd="pocketd tx authz exec \"$PARAM_DIR/$file\" \
      --from=$from \
      --keyring-backend=$keyring_backend \
      --chain-id=$chain_id \
      --node=\"$node\" \
      --yes \
      --home=$home \
      --fees=$fees \
      --unordered \
      --timeout-duration=1m"

    run_command "$cmd"
  done

  echo "Processing Cosmos SDK parameters..."
  for file in "${COSMOS_PARAM_FILES[@]}"; do
    echo "Processing $file..."

    local cmd="pocketd tx authz exec \"$PARAM_DIR/$file\" \
      --from=$from \
      --keyring-backend=$keyring_backend \
      --chain-id=$chain_id \
      --node=\"$node\" \
      --yes \
      --home=$home \
      --fees=$fees \
      --unordered \
      --timeout-duration=1m"

    run_command "$cmd"
  done
}

update_gov_params() {
  local env="$1"

  echo "Running governance parameters for $env network..."

  if [[ ! -x "$GOV_PARAM_SCRIPT" ]]; then
    error "Governance parameter script is not executable: $GOV_PARAM_SCRIPT"
  fi

  local cmd="$GOV_PARAM_SCRIPT $env"
  run_command "$cmd"
}

main() {
  local env=""
  local mode="state" # default to state parameters
  local env_provided=false

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case $1 in
    -h | --help)
      show_help
      exit 0
      ;;
    -n | --dry-run)
      DRY_RUN=true
      shift
      ;;
    -v | --verbose)
      VERBOSE=true
      shift
      ;;
    --gov-only)
      GOV_ONLY=true
      mode="gov"
      shift
      ;;
    --state-only)
      mode="state"
      shift
      ;;
    --param-dir=*)
      PARAM_DIR="${1#*=}"
      shift
      ;;
    --param-dir)
      if [[ -n "$2" && "$2" != -* ]]; then
        PARAM_DIR="$2"
        shift 2
      else
        error "--param-dir requires a directory path"
      fi
      ;;
    local | alpha | beta | main)
      env="$1"
      env_provided=true
      shift
      ;;
    *)
      error "Unknown option: $1. Use --help for usage information."
      ;;
    esac
  done

  # Show help if no environment was provided
  if [[ "$env_provided" == false ]]; then
    show_help
    exit 0
  fi

  # Validate environment
  validate_environment "$env"

  # Show configuration
  echo "=== Parameter Update Configuration ==="
  echo "Environment: $env"
  echo "Mode: $mode"
  echo "Parameter Directory: $PARAM_DIR"
  echo "Dry run: ${DRY_RUN:-false}"
  echo "Verbose: ${VERBOSE:-false}"
  echo "======================================"

  # Validate files exist
  validate_files

  # Prompt for confirmation (skip if dry run)
  if [[ "${DRY_RUN:-false}" != "true" ]]; then
    prompt_confirmation "$env" "$mode"
  fi

  # Execute based on mode
  case "$mode" in
  state)
    update_state_params "$env"
    ;;
  gov)
    update_gov_params "$env"
    ;;
  both)
    update_state_params "$env"
    update_gov_params "$env"
    ;;
  esac

  echo "✅ Parameter update completed successfully for $env network"
}

# Run main function with all arguments
main "$@"
