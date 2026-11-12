package keeper_test

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	testutilevents "github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// nextSessionStart returns the start height of the session after the current one.
func nextSessionStart(k keepertest.SupplierModuleKeepers, ctx cosmostypes.Context) int64 {
	sharedParams := k.SharedKeeper.GetParams(ctx)
	return sharedtypes.GetNextSessionStartHeight(&sharedParams, ctx.BlockHeight())
}

// deactivationHeights returns the deactivation height of each of the supplier's
// stored service configs, by service ID.
func deactivationHeights(t *testing.T, k keepertest.SupplierModuleKeepers, ctx cosmostypes.Context, operatorAddr string) map[string]int64 {
	t.Helper()
	supplier, found := k.GetSupplier(ctx, operatorAddr)
	require.True(t, found)
	heights := make(map[string]int64)
	for _, config := range supplier.ServiceConfigHistory {
		heights[config.Service.ServiceId] = config.DeactivationHeight
	}
	return heights
}

func TestDeactivateSupplierService(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	ownerAddr, operatorAddr := sample.AccAddressBech32(), sample.AccAddressBech32()

	stakeMsg, _ := newSupplierStakeMsg(ownerAddr, operatorAddr, 1000000, "svcId")
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)
	supplier, found := k.GetDehydratedSupplier(ctx, operatorAddr)
	require.True(t, found)

	// A service the supplier has no config for.
	require.False(t, k.DeactivateSupplierService(ctx, &supplier, "svcId2"))

	// An unbonding supplier is left to the unbonding path.
	unbonding := supplier
	unbonding.UnstakeSessionEndHeight = 10
	require.False(t, k.DeactivateSupplierService(ctx, &unbonding, "svcId"))
	require.Empty(t, testutilevents.FilterEvents[*suppliertypes.EventSupplierServiceNotAllowed](t, sdkCtx.EventManager().Events()))

	// A counted config deactivates at the next session start.
	next := nextSessionStart(k, sdkCtx)
	require.True(t, k.DeactivateSupplierService(ctx, &supplier, "svcId"))
	require.Equal(t, map[string]int64{"svcId": next}, deactivationHeights(t, k, sdkCtx, operatorAddr))

	// Already deactivating at the next session start: nothing left to deactivate.
	require.False(t, k.DeactivateSupplierService(ctx, &supplier, "svcId"))

	events := testutilevents.FilterEvents[*suppliertypes.EventSupplierServiceNotAllowed](t, sdkCtx.EventManager().Events())
	require.Len(t, events, 1)
	require.Equal(t, operatorAddr, events[0].GetOperatorAddress())
	require.Equal(t, ownerAddr, events[0].GetOwnerAddress())
	require.Equal(t, "svcId", events[0].GetServiceId())
	require.Equal(t, next, events[0].GetEffectiveHeight())

	// The supplier is never unbonded.
	stored, found := k.GetDehydratedSupplier(ctx, operatorAddr)
	require.True(t, found)
	require.False(t, stored.IsUnbonding())
}

func TestStakeSupplier_OwnerChangeAllowlist(t *testing.T) {
	tests := []struct {
		desc            string
		listOperator    bool
		wantDeactivated bool
	}{
		{desc: "new owner not listed: permissioned service deactivates", listOperator: false, wantDeactivated: true},
		{desc: "operator listed: nothing deactivates", listOperator: true, wantDeactivated: false},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			k, ctx := keepertest.SupplierKeeper(t)
			sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
			srv := keeper.NewMsgServerImpl(*k.Keeper)
			oldOwnerAddr, newOwnerAddr, operatorAddr := sample.AccAddressBech32(), sample.AccAddressBech32(), sample.AccAddressBech32()

			// svcId is permissioned and lists the old owner; svcId2 is open.
			k.ServiceKeeper.SetSupplierAllowlistEntry(ctx, "svcId", oldOwnerAddr)
			if test.listOperator {
				k.ServiceKeeper.SetSupplierAllowlistEntry(ctx, "svcId", operatorAddr)
			}

			stakeMsg, _ := newSupplierStakeMsg(oldOwnerAddr, operatorAddr, 1000000, "svcId", "svcId2")
			_, err := srv.StakeSupplier(ctx, stakeMsg)
			require.NoError(t, err)

			// Stake-only owner change, signed by the old owner.
			_, err = srv.StakeSupplier(ctx, &suppliertypes.MsgStakeSupplier{
				Signer:          oldOwnerAddr,
				OwnerAddress:    newOwnerAddr,
				OperatorAddress: operatorAddr,
				Stake:           stakeMsg.Stake,
			})
			require.NoError(t, err)

			stored, found := k.GetSupplier(ctx, operatorAddr)
			require.True(t, found)
			require.Equal(t, newOwnerAddr, stored.OwnerAddress)

			events := testutilevents.FilterEvents[*suppliertypes.EventSupplierServiceNotAllowed](t, sdkCtx.EventManager().Events())
			heights := deactivationHeights(t, k, sdkCtx, operatorAddr)
			require.Equal(t, int64(sharedtypes.NoDeactivationHeight), heights["svcId2"])
			if !test.wantDeactivated {
				require.Equal(t, int64(sharedtypes.NoDeactivationHeight), heights["svcId"])
				require.Empty(t, events)
				return
			}

			next := nextSessionStart(k, sdkCtx)
			require.Equal(t, next, heights["svcId"])
			require.Len(t, events, 1)
			require.Equal(t, "svcId", events[0].GetServiceId())
			require.Equal(t, newOwnerAddr, events[0].GetOwnerAddress())
			require.Equal(t, next, events[0].GetEffectiveHeight())
		})
	}
}
