// Package cli_test provides unit tests for the CLI functionality.
package application_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/network"
	"github.com/pokt-network/poktroll/x/application/types"
)

// Dummy variable to avoid unused import error.
var _ = strconv.IntSize

// networkWithApplicationObjects creates a new network with a given number of application objects.
// It returns the network and a slice of the created application objects.
func networkWithApplicationObjects(t *testing.T, n int) (*network.Network, []types.Application) {
	t.Helper()

	// Configure the testing network
	cfg := network.DefaultConfig()
	appGenesisState := network.DefaultApplicationModuleGenesisState(t, n)
	buf, err := cfg.Codec.MarshalJSON(appGenesisState)
	require.NoError(t, err)
	cfg.GenesisState[types.ModuleName] = buf

	// Start the network
	net := network.New(t, cfg)

	// Wait for the network to be fully initialized to avoid race conditions
	// with consensus reactor goroutines
	require.NoError(t, net.WaitForNextBlock())

	// Additional wait to ensure all consensus components are fully initialized
	require.NoError(t, net.WaitForNextBlock())

	return net, appGenesisState.ApplicationList
}
