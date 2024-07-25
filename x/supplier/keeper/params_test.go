package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

func TestGetParams(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	params := types.DefaultParams()

	require.NoError(t, supplierModuleKeepers.SetParams(ctx, params))
	require.EqualValues(t, params, supplierModuleKeepers.Keeper.GetParams(ctx))
}
