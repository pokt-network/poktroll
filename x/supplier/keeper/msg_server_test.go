package keeper_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/pocket/testutil/keeper"
	"github.com/pokt-network/pocket/x/supplier/keeper"
	"github.com/pokt-network/pocket/x/supplier/types"
)

func setupMsgServer(t testing.TB) (keeper.Keeper, types.MsgServer, context.Context) {
	t.Helper()

	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	return *supplierModuleKeepers.Keeper, keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper), ctx
}

func TestMsgServer(t *testing.T) {
	t.Helper()

	k, ms, ctx := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotEmpty(t, k)
}
