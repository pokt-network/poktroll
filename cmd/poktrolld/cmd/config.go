package cmd

import (
	"strings"
	"sync"
	"time"

	cmtcfg "github.com/cometbft/cometbft/config"
	clientconfig "github.com/cosmos/cosmos-sdk/client/config"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/app"
	"github.com/pokt-network/poktroll/telemetry"
)

var once sync.Once

// PoktrollAppConfig represents a poktroll-specific part of `app.toml` file.
// Checkout `customAppConfigTemplate()` for additional information about each setting.
type PoktrollAppConfig struct {
	Telemetry telemetry.PoktrollTelemetryConfig `mapstructure:"telemetry"`
}

// poktrollAppConfigDefaults sets default values to render in `app.toml`.
// Checkout `customAppConfigTemplate()` for additional information about each setting.
func poktrollAppConfigDefaults() PoktrollAppConfig {
	return PoktrollAppConfig{
		Telemetry: telemetry.PoktrollTelemetryConfig{
			CardinalityLevel: "medium",
		},
	}
}

func InitSDKConfig() {
	once.Do(func() {
		checkOrInitSDKConfig()
	})
}

// checkOrInitSDKConfig updates the prefixes for all account types and seals the config.
// DEV_NOTE: Due to the separation of this repo and the SDK, where the config is also sealed,
// we have an added check to return early in case the config has already been set to the expected
// value.
func checkOrInitSDKConfig() {
	config := sdk.GetConfig()

	// Check if the config is already set with the correct prefixes
	if config.GetBech32AccountAddrPrefix() == app.AccountAddressPrefix {
		// Config is already initialized, return early
		return
	}

	// Set prefixes
	accountPubKeyPrefix := app.AccountAddressPrefix + "pub"
	validatorAddressPrefix := app.AccountAddressPrefix + "valoper"
	validatorPubKeyPrefix := app.AccountAddressPrefix + "valoperpub"
	consNodeAddressPrefix := app.AccountAddressPrefix + "valcons"
	consNodePubKeyPrefix := app.AccountAddressPrefix + "valconspub"

	// Set and seal config
	config.SetBech32PrefixForAccount(app.AccountAddressPrefix, accountPubKeyPrefix)
	config.SetBech32PrefixForValidator(validatorAddressPrefix, validatorPubKeyPrefix)
	config.SetBech32PrefixForConsensusNode(consNodeAddressPrefix, consNodePubKeyPrefix)
	config.Seal()
}

// The values set here become the default configuration for newly initialized nodes.
// However, it's crucial to note that:
// 1. These defaults only apply when a node is first initialized using `poktrolld init`.
// 2. Changing these values in the code will not automatically update existing node configurations.
// 3. Node operators can still manually override these defaults in their local config files.
//
// Therefore, it's critical to choose sensible default values carefully, as they will form
// the baseline configuration for most network participants. Any future changes to these
// defaults will only affect newly initialized nodes, not existing ones.

// As we use `ignite` CLI to provision the first validator it is important to note that the configuration files
// provisioned by ignite have additional overrides adjusted in ignite's `config.yml`

// initCometBFTConfig helps to override default CometBFT Config (config.toml) values.
// These values are going to be rendered into the config file on `poktrolld init`.
// TODO_MAINNET: Reconsider values - check `config.toml` for possible options.
func initCometBFTConfig() *cmtcfg.Config {
	cfg := cmtcfg.DefaultConfig()

	// these values put a higher strain on node memory
	// cfg.P2P.MaxNumInboundPeers = 100
	// cfg.P2P.MaxNumOutboundPeers = 40

	cfg.Consensus.TimeoutPropose = 60 * time.Second
	cfg.Consensus.TimeoutProposeDelta = 5 * time.Second
	cfg.Consensus.TimeoutPrevote = 10 * time.Second
	cfg.Consensus.TimeoutPrevoteDelta = 5 * time.Second
	cfg.Consensus.TimeoutPrecommit = 10 * time.Second
	cfg.Consensus.TimeoutPrecommitDelta = 5 * time.Second
	cfg.Consensus.TimeoutCommit = 60 * time.Second
	cfg.Instrumentation.Prometheus = true
	cfg.LogLevel = "info"

	return cfg
}

// initAppConfig helps to override default appConfig (app.toml) template and configs.
// These values are going to be rendered into the config file on `poktrolld init`.
// return "", nil if no custom configuration is required for the application.
// TODO_MAINNET: Reconsider values - check `app.toml` for possible options.
func initAppConfig() (string, interface{}) {
	// The following code snippet is just for reference.
	type CustomAppConfig struct {
		serverconfig.Config `mapstructure:",squash"`
		Poktroll            PoktrollAppConfig `mapstructure:"poktroll"`
	}

	// Optionally allow the chain developer to overwrite the SDK's default
	// server config.
	srvCfg := serverconfig.DefaultConfig()
	// The SDK's default minimum gas price is set to "" (empty value) inside
	// app.toml. If left empty by validators, the node will halt on startup.
	// However, the chain developer can set a default app.toml value for their
	// validators here.
	//
	// In summary:
	// - if you leave srvCfg.MinGasPrices = "", all validators MUST tweak their
	//   own app.toml config,
	// - if you set srvCfg.MinGasPrices non-empty, validators CAN tweak their
	//   own app.toml to override, or use this default value.
	//
	// In tests, we set the min gas prices to 0.
	// srvCfg.MinGasPrices = "0stake"
	// srvCfg.BaseConfig.IAVLDisableFastNode = true // disable fastnode by default

	srvCfg.MinGasPrices = "0.000000001upokt" // Also adjust ignite's `config.yml`.
	srvCfg.Mempool.MaxTxs = 10000
	srvCfg.Telemetry.Enabled = true
	// Positive non-zero value turns on Prometheus support.
	// Prometheus metrics are removed from the exporter when retention time is reached.
	srvCfg.Telemetry.PrometheusRetentionTime = 60 * 60 * 24 // in seconds.
	srvCfg.Telemetry.MetricsSink = "mem"
	srvCfg.Pruning = "nothing" // archiving node by default
	srvCfg.API.Enable = true
	srvCfg.GRPC.Enable = true
	srvCfg.GRPCWeb.Enable = true

	customAppConfig := CustomAppConfig{
		Config:   *srvCfg,
		Poktroll: poktrollAppConfigDefaults(),
	}

	return customPoktrollAppConfigTemplate(), customAppConfig
}

// customPoktrollAppConfigTemplate extends the default configuration `app.toml` file with our own configs.
// They are going to be used by validators and full-nodes.
// These configs are rendered using default values from `poktrollAppConfigDefaults()`.
func customPoktrollAppConfigTemplate() string {
	return serverconfig.DefaultConfigTemplate + `
		###############################################################################
		###                               Poktroll                                  ###
		###############################################################################

		# Poktroll-specific app configuration for Full Nodes and Validators.
		[poktroll]

		# Telemetry configuration in addition to the [telemetry] settings.
		[poktroll.telemetry]

		# Cardinality level for telemetry metrics collection
		# This controls the level of detail (number of unique labels) in metrics.
		# Options:
		#   - "low":    Collects basic metrics with low cardinality.
		#              Suitable for production environments with tight performance constraints.
		#   - "medium": Collects a moderate number of labels, balancing detail and performance.
		#              Suitable for moderate workloads or staging environments.
		#   - "high":   WARNING: WILL CAUSE STRESS TO YOUR MONITORING ENVIRONMENT! Collects detailed metrics with high
		#              cardinality, including labels with many unique values (e.g., application_id, session_id).
		#              Recommended for debugging or testing environments.
		cardinality-level = "{{ .Poktroll.Telemetry.CardinalityLevel }}"
		`
}

// initClientConfig helps to override default client config (client.toml) template and configs.
// Allows to dynamically create client.toml file with custom values.
func initClientConfig() (string, interface{}) {
	type GasConfig struct {
		GasAdjustment float64 `mapstructure:"gas-adjustment"`
		Gas           string  `mapstructure:"gas"`
	}

	type CustomClientConfig struct {
		clientconfig.ClientConfig `mapstructure:",squash"`

		GasConfig GasConfig `mapstructure:"gas"`
	}

	// Default configuration from cosmos-sdk
	clientCfg := clientconfig.DefaultConfig()
	// clientCfg.ChainID = "" TODO_MAINNET: set mainnet chain-id

	// Now we set the custom config default values.
	customClientConfig := CustomClientConfig{
		ClientConfig: *clientCfg,
		GasConfig: GasConfig{
			GasAdjustment: 1.5,    // default is 1.
			Gas:           "auto", // default is 200000
		},
	}

	// According to cosmos-sdk documentation, the template is exported via clientconfig.DefaultClientConfigTemplate,
	// however as of 0.50.9 - it is not, so we're copying their template.
	// TODO_TECHDEBT: switch to `clientconfig.DefaultClientConfigTemplate` on newer cosmos-sdk versions.
	defaultConfigTemplate := `# This is a TOML config file.
	# For more information, see https://github.com/toml-lang/toml
	
	###############################################################################
	###                           Client Configuration                            ###
	###############################################################################
	
	# The network chain ID
	chain-id = "{{ .ChainID }}"
	# The keyring's backend, where the keys are stored (os|file|kwallet|pass|test|memory)
	keyring-backend = "{{ .KeyringBackend }}"
	# CLI output format (text|json)
	output = "{{ .Output }}"
	# <host>:<port> to CometBFT RPC interface for this chain
	node = "{{ .Node }}"
	# Transaction broadcasting mode (sync|async)
	broadcast-mode = "{{ .BroadcastMode }}"
	`

	// Adding our custom config template
	customClientConfigTemplate := defaultConfigTemplate + strings.TrimSpace(`
	# This is default the gas adjustment factor used in tx commands.
	# It can be overwritten by the --gas-adjustment flag in each tx command.
	gas-adjustment = {{ .GasConfig.GasAdjustment }}
	# Gas limit to set per-transaction; set to "auto" to calculate sufficient gas automatically
	gas = "{{ .GasConfig.Gas }}"
	`)

	return customClientConfigTemplate, customClientConfig
}
