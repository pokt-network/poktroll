# Using string allow to add comment lines to the generated file
default_localnet_config_template = """
# Enable or disable Tiltfile hot-reloading when files change
hot-reloading: true

faucet:
  # Whether to deploy a faucet service for funding accounts
  enabled: true

grove_helm_chart_local_repo:
  # Enable usage of a local checkout of the Grove Helm charts instead of fetching from remote
  enabled: false
  # Relative path to the Grove Helm charts repo
  path: ../grove-helm-charts

helm_chart_local_repo:
  # Enable usage of a local checkout of Helm charts for other components
  enabled: false
  # Relative path to the Helm charts directory
  path: ../helm-charts

indexer:
  # Whether to deploy the indexer (e.g., PocketDex indexer)
  enabled: false
  # Branch to use from the indexer Git repository
  repo_branch: main
  # Local path to the indexer Git repository (used if already cloned)
  repo_path: ../pocketdex
  # URL to clone the indexer Git repository (used if clone_if_not_present is true)
  repo_url: https://github.com/pokt-network/pocketdex
  # Clone the indexer repo if it's not already present at repo_path
  clone_if_not_present: false
  # Path to the Tiltfile inside the indexer repo to include and run
  entrypoint_path: tilt/Tiltfile
  # pocketdex params to modify its behavior
  params:
    network: localnet
    # relative to `repo_path` (only valid for localnet)
    genesis_file_path: "../poktroll/localnet/pocketd/config/genesis.json"

    pgadmin:
      # Whether to deploy pgAdmin UI
      enabled: true
      # Optional: pgAdmin login email (default: pocketdex@local.dev)
      # email: ""
      # Optional: pgAdmin login password (default: pocketdex)
      # password: ""

    overwrite: {}
      # Optional parameters that can be customized:
      # node_options: "--max-old-space-size=24576"
      # block_batch_size: ""           # Size of block batch processing
      # endpoint: ""                   # Node RPC endpoint
      # chain_id: ""                   # Target chain ID
      # start_block: ""                # Start block for indexing
      # page_limit: ""                 # Query page size
      # db_batch_size: ""              # DB insert batch size
      # db_bulk_concurrency: ""        # Concurrent bulk insert jobs
      # pg_pool:                       # Optional PgBouncer tuning
      #   min: ""                      # Min pool connections
      #   max: ""                      # Max pool connections
      #   acquire: ""                  # Acquire timeout
      #   idle: ""                     # Idle timeout
      #   evict: ""                    # Eviction timeout

observability:
  # Whether to deploy Prometheus + Grafana
  enabled: true
  grafana:
    # Whether to deploy default dashboards
    defaultDashboardsEnabled: false

ollama:
  # Whether to deploy Ollama (for AI agent support)
  enabled: false
  # Ollama model to use (e.g., llama2, mistral, qwen)
  model: qwen:0.5b

path_gateways:
  # Number of path gateway nodes to run
  count: 1

path_local_repo:
  # Whether to use a local checkout of the PATH repo
  enabled: false
  # Relative path to the PATH repo
  path: ../path

relayminers:
  # Number of relay miner nodes to run
  # IMPORTANT: Set to 0 when using ha_relayminers (they are mutually exclusive)
  count: 1
  delve:
    # Enable delve debugger for relay miners
    enabled: false
  logs:
    # Log level for relay miners (e.g., debug, info, warn)
    level: debug

ha_relayminers:
  # Enable HA (High Availability) relay miners
  # When enabled, relay miners use Redis for shared state and can scale horizontally
  # IMPORTANT: Mutually exclusive with standard relayminers - set relayminers.count to 0 when enabling
  enabled: false
  # Number of HA relay miner instances to run
  count: 1
  logs:
    # Log level for HA relay miners
    level: debug
  redis:
    # Deploy Redis for HA relay miner coordination
    enabled: true

rest:
  # Whether to enable the REST server
  enabled: true

ibc:
  enabled: false
  validator_configs:
    agoric:
      enabled: false
      # NOTE: this chain ID is baked into the image genesis.json and is difficult to change.
      chain_id: "agoriclocal"
      values_path: localnet/kubernetes/values-agoric.yaml
      tilt_ui_name: "Agoric Validator"
      chart_name: "agoric-validator"
      dockerfile_path: localnet/dockerfiles/agoric-validator.dockerfile
      image_name: "agoric"
      port_forwards: ["46657:26657", "11090:9090", "40009:40009" ]
    axelar:
        enabled: True
        chain_id: "axelar"
        values_path: os.path.join("localnet", "kubernetes", "values-axelar.yaml")
        tilt_ui_name: "Axelar Validator"
        chart_name: "axelar-validator"
        dockerfile_path: os.path.join("localnet", "dockerfiles", "axelar-validator.dockerfile")
        image_name: "axelar"
        port_forwards:
          - "56657:26657"
          - "12090:9090"
          - "40010:40010"

validator:
  # If true, delete validator state before each start
  cleanupBeforeEachStart: true
  delve:
    # Enable delve debugger for validator
    enabled: false
  logs:
    # Log format: plain or json
    format: json
    # Log level: debug, info, warn, error
    level: info

"""

def get_defaults():
    # Load defaults
    default_stream = decode_yaml_stream(default_localnet_config_template)
    # First item in stream must be a dict
    localnet_config_defaults = default_stream[0] if len(default_stream) > 0 else {}

    return localnet_config_defaults

