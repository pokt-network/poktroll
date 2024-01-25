// Package cli_test provides unit tests for the CLI functionality.
package cli_test

import (
	"strconv"
	"testing"

	"github.com/pokt-network/poktroll/cmd/pocketd/cmd"
	"github.com/pokt-network/poktroll/testutil/network"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
	"github.com/stretchr/testify/require"
)

// Dummy variable to avoid unused import error.
var _ = strconv.IntSize

// init initializes the SDK configuration.
func init() {
	cmd.InitSDKConfig()
}

// networkWithDefaultConfig is a helper function to create a network for testing
// with a default tokenomics genesis state.
func networkWithDefaultConfig(t *testing.T) *network.Network {
	t.Helper()
	cfg := network.DefaultConfig()
	tokenomicsGenesisState := network.DefaultTokenomicsModuleGenesisState(t)
	buf, err := cfg.Codec.MarshalJSON(tokenomicsGenesisState)
	require.NoError(t, err)
	cfg.GenesisState[types.ModuleName] = buf
	return network.New(t, cfg)
}
