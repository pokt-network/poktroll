package cmd

import (
	"sync"
	"time"

	cmtcfg "github.com/cometbft/cometbft/config"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/app"
)

var once sync.Once

func InitSDKConfig() {
	once.Do(func() {
		initSDKConfig()
	})
}

func initSDKConfig() {
	// Set prefixes
	accountPubKeyPrefix := app.AccountAddressPrefix + "pub"
	validatorAddressPrefix := app.AccountAddressPrefix + "valoper"
	validatorPubKeyPrefix := app.AccountAddressPrefix + "valoperpub"
	consNodeAddressPrefix := app.AccountAddressPrefix + "valcons"
	consNodePubKeyPrefix := app.AccountAddressPrefix + "valconspub"

	// Set and seal config
	config := sdk.GetConfig()
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
	srvCfg.Telemetry.PrometheusRetentionTime = 60 // in seconds. This turns on Prometheus support.
	srvCfg.Telemetry.MetricsSink = "mem"
	srvCfg.Pruning = "nothing" // archiving node by default
	srvCfg.API.Enable = true
	srvCfg.GRPC.Enable = true
	srvCfg.GRPCWeb.Enable = true

	customAppConfig := CustomAppConfig{
		Config: *srvCfg,
	}

	customAppTemplate := serverconfig.DefaultConfigTemplate
	// Edit the default template file
	//
	// customAppTemplate := serverconfig.DefaultConfigTemplate + `
	// [wasm]
	// # This is the maximum sdk gas (wasm and storage) that we allow for any x/wasm "smart" queries
	// query_gas_limit = 300000
	// # This is the number of wasm vm instances we keep cached in memory for speed-up
	// # Warning: this is currently unstable and may lead to crashes, best to keep for 0 unless testing locally
	// lru_size = 0`

	return customAppTemplate, customAppConfig
}
