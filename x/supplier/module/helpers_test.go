package supplier_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/network"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// Dummy variable to avoid unused import error.
var _ = strconv.IntSize

// networkWithSupplierObjects creates a new network with a given number of supplier objects.
// It returns the network and a slice of the created supplier objects.
func networkWithSupplierObjects(t *testing.T, n int) (*network.Network, []sharedtypes.Supplier) {
	t.Helper()

	// Configure the testing network
	cfg := network.DefaultConfig()
	supplierGenesisState := network.DefaultSupplierModuleGenesisState(t, n)
	buf, err := cfg.Codec.MarshalJSON(supplierGenesisState)
	require.NoError(t, err)
	cfg.GenesisState[types.ModuleName] = buf

	// Start the network
	net := network.New(t, cfg)

	// Wait for the network to be fully initialized to avoid race conditions
	// with consensus reactor goroutines
	require.NoError(t, net.WaitForNextBlock())

	// Additional wait to ensure all consensus components are fully initialized
	require.NoError(t, net.WaitForNextBlock())

	return net, supplierGenesisState.SupplierList
}
