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
    beta | main)
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
  echo "Dry run: ${DRY_RUN:-false}"
  echo "Verbose: ${VERBOSE:-false}"
  echo "======================================"

  # Validate files exist
  validate_files

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
