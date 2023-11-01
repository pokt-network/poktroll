// Package cli_test provides unit tests for the CLI functionality.
package cli_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"pocket/cmd/pocketd/cmd"
	"pocket/testutil/network"
	apptypes "pocket/x/application/types"
	sharedtypes "pocket/x/shared/types"
	suppliertypes "pocket/x/supplier/types"
)

// Dummy variable to avoid unused import error.
var _ = strconv.IntSize

// init initializes the SDK configuration.
func init() {
	cmd.InitSDKConfig()
}

// networkWithSupplierObjects creates a new network with a given number of supplier objects.
// It returns the network and a slice of the created supplier objects.
func networkWithApplicationsAndSupplier(t *testing.T, n int) (*network.Network, []sharedtypes.Supplier, []apptypes.Application) {
	t.Helper()
	cfg := network.DefaultConfig()

	applicationGenesisState := network.DefaultApplicationModuleGenesisState(t, n)
	buf, err := cfg.Codec.MarshalJSON(applicationGenesisState)
	require.NoError(t, err)
	cfg.GenesisState[apptypes.ModuleName] = buf

	supplierGenesisState := network.DefaultSupplierModuleGenesisState(t, n)
	buf, err = cfg.Codec.MarshalJSON(supplierGenesisState)
	require.NoError(t, err)
	cfg.GenesisState[suppliertypes.ModuleName] = buf

	return network.New(t, cfg), supplierGenesisState.SupplierList, applicationGenesisState.ApplicationList
}
