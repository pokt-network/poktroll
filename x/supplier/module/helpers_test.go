package supplier_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/network"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// Dummy variable to avoid unused import error.
var _ = strconv.IntSize

// networkWithSupplierObjects creates a new network with a given number of supplier objects.
// It returns the network and a slice of the created supplier objects.
func networkWithSupplierObjects(t *testing.T, n int) (*network.Network, []sharedtypes.Supplier) {
	t.Helper()

	// TODO_TECHDEBT: Remove once dao reward address is promoted to a tokenomics param.
	tokenomicstypes.DaoRewardAddress = sample.AccAddress()

	cfg := network.DefaultConfig()
	supplierGenesisState := network.DefaultSupplierModuleGenesisState(t, n)
	buf, err := cfg.Codec.MarshalJSON(supplierGenesisState)
	require.NoError(t, err)
	cfg.GenesisState[types.ModuleName] = buf
	return network.New(t, cfg), supplierGenesisState.SupplierList
}
