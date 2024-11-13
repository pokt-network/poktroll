// Package cli_test provides unit tests for the CLI functionality.
package tokenomics_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/network"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// Dummy variable to avoid unused import error.
var _ = strconv.IntSize

// networkWithDefaultConfig is a helper function to create a network for testing
// with a default tokenomics genesis state.
//
//lint:ignore U1000 Ignore unused function for testing purposes
func networkWithDefaultConfig(t *testing.T) *network.Network {
	t.Helper()
	cfg := network.DefaultConfig()
	tokenomicsGenesisState := network.DefaultTokenomicsModuleGenesisState(t)
	buf, err := cfg.Codec.MarshalJSON(tokenomicsGenesisState)
	require.NoError(t, err)
	cfg.GenesisState[types.ModuleName] = buf
	return network.New(t, cfg)
}
