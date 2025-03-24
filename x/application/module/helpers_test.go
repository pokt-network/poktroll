// Package cli_test provides unit tests for the CLI functionality.
package application_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/pocket/testutil/network"
	"github.com/pokt-network/pocket/x/application/types"
)

// Dummy variable to avoid unused import error.
var _ = strconv.IntSize

// networkWithApplicationObjects creates a new network with a given number of application objects.
// It returns the network and a slice of the created application objects.
func networkWithApplicationObjects(t *testing.T, n int) (*network.Network, []types.Application) {
	t.Helper()

	cfg := network.DefaultConfig()
	appGenesisState := network.DefaultApplicationModuleGenesisState(t, n)
	buf, err := cfg.Codec.MarshalJSON(appGenesisState)
	require.NoError(t, err)
	cfg.GenesisState[types.ModuleName] = buf
	return network.New(t, cfg), appGenesisState.ApplicationList
}
