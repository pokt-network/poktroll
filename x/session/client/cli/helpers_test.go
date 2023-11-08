// Package cli_test provides unit tests for the CLI functionality.
package cli_test

import (
	"strconv"
	"testing"

	"github.com/pokt-network/poktroll/cmd/pocketd/cmd"
	"github.com/pokt-network/poktroll/testutil/network"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	"github.com/stretchr/testify/require"
)

// Dummy variable to avoid unused import error.
var _ = strconv.IntSize

// init initializes the SDK configuration.
func init() {
	cmd.InitSDKConfig()
}

// networkWithApplicationsAndSupplier creates a new network with a given number of supplier & application objects.
// It returns the network and a slice of the created supplier & application objects.
func networkWithApplicationsAndSupplier(t *testing.T, n int) (*network.Network, []sharedtypes.Supplier, []apptypes.Application) {
	t.Helper()
	cfg := network.DefaultConfig()

	// Prepare the application genesis state
	applicationGenesisState := network.DefaultApplicationModuleGenesisState(t, n)
	buf, err := cfg.Codec.MarshalJSON(applicationGenesisState)
	require.NoError(t, err)
	cfg.GenesisState[apptypes.ModuleName] = buf

	// Prepare the supplier genesis state
	supplierGenesisState := network.DefaultSupplierModuleGenesisState(t, n)
	buf, err = cfg.Codec.MarshalJSON(supplierGenesisState)
	require.NoError(t, err)
	cfg.GenesisState[suppliertypes.ModuleName] = buf

	// Start the network
	return network.New(t, cfg), supplierGenesisState.SupplierList, applicationGenesisState.ApplicationList
}
