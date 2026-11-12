package keeper_test

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	supplier "github.com/pokt-network/poktroll/x/supplier/module"
)

// The allowlist sweep runs before the cap sweep in the EndBlocker: when a supplier
// leaves svcA because it is not allowed there, the service slot its stake pays
// for keeps svcX instead of the cap dropping svcX as well.
func TestEndBlocker_AllowlistSweepRunsBeforeCapSweep(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	enableServiceCap(t, k, sdkCtx)

	ownerAddr, operatorAddr := sample.AccAddressBech32(), sample.AccAddressBech32()
	_, err := srv.StakeSupplier(sdkCtx, multiServiceStakeMsg(t, k, sdkCtx, ownerAddr, operatorAddr, capMinStake+capPrice, "svcA", "svcX"))
	require.NoError(t, err)
	permissionAndMark(k, sdkCtx, "svcA")

	// The price doubles, so the stake now covers one service only.
	params := k.Keeper.GetParams(sdkCtx)
	price := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 2*capPrice)
	params.StakePerAdditionalService = &price
	require.NoError(t, k.SetParams(sdkCtx, params))

	blockCtx := sdkCtx.WithBlockHeight(5)
	require.NoError(t, supplier.EndBlocker(blockCtx, *k.Keeper))

	next := nextSessionStart(k, blockCtx)
	heights := deactivationHeights(t, k, sdkCtx, operatorAddr)
	require.Equal(t, next, heights["svcA"], "not allowed on svcA")
	require.Equal(t, int64(sharedtypes.NoDeactivationHeight), heights["svcX"], "the slot svcA leaves keeps svcX")
}
