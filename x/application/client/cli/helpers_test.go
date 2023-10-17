// Package cli_test provides unit tests for the CLI functionality.
package cli_test

import (
	"strconv"
	"testing"

	"pocket/cmd/pocketd/cmd"
	"pocket/testutil/network"
	"pocket/x/application/types"
)

// Dummy variable to avoid unused import error.
var _ = strconv.IntSize

// init initializes the SDK configuration.
func init() {
	cmd.InitSDKConfig()
}

// networkWithApplicationObjects creates a new network with a given number of application objects.
// It returns the network and a slice of the created application objects.
func networkWithApplicationObjects(t *testing.T, n int) (*network.Network, []types.Application) {
	t.Helper()
	cfg := network.DefaultConfig()
	appGenesisState := network.DefaultApplicationModuleGenesisState(t, n)
	network.HydrateGenesisState(t, &cfg, appGenesisState, types.ModuleName)
	return network.New(t, cfg), appGenesisState.ApplicationList
}
