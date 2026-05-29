package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/pkg/relayer/config"
	"github.com/pokt-network/poktroll/testutil/yaml"
)

// baseMiningKnobsConfig is a minimal valid RelayMiner config used to exercise the
// mining-pipeline tuning knobs in isolation.
const baseMiningKnobsConfig = `
default_signing_key_names: [supplier1]
smt_store_path: /tmp/pocket/smt
pocket_node:
  query_node_rpc_url: http://127.0.0.1:26657
  query_node_grpc_url: http://127.0.0.1:9090
  tx_node_rpc_url: http://127.0.0.1:26657
suppliers:
  - service_id: svc1
    listen_url: http://127.0.0.1:8545
    service_config:
      backend_url: http://127.0.0.1:8546
`

func Test_ParseRelayMinerConfigs_MiningKnobsDefaults(t *testing.T) {
	normalized := yaml.NormalizeYAMLIndentation(baseMiningKnobsConfig)

	cfg, err := config.ParseRelayMinerConfigs(polyzero.NewLogger(), []byte(normalized))
	require.NoError(t, err)

	// Unset knobs fall back to the historical hardcoded values; workers stays 0 (auto).
	require.Equal(t, int(config.DefaultServedRelaysBufferSize), cfg.ServedRelaysBufferSize)
	require.Equal(t, int(config.DefaultMiningPipelineBufferSize), cfg.MiningPipelineBufferSize)
	require.Equal(t, 0, cfg.MiningWorkers)
}

func Test_ParseRelayMinerConfigs_MiningKnobsOverrides(t *testing.T) {
	withOverrides := baseMiningKnobsConfig + `
served_relays_buffer_size: 5000
mining_pipeline_buffer_size: 200
mining_workers: 12
`
	normalized := yaml.NormalizeYAMLIndentation(withOverrides)

	cfg, err := config.ParseRelayMinerConfigs(polyzero.NewLogger(), []byte(normalized))
	require.NoError(t, err)

	require.Equal(t, 5000, cfg.ServedRelaysBufferSize)
	require.Equal(t, 200, cfg.MiningPipelineBufferSize)
	require.Equal(t, 12, cfg.MiningWorkers)
}
