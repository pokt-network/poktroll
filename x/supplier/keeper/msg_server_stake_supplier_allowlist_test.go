package keeper_test

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

func TestStakeSupplier_Allowlist(t *testing.T) {
	const svcId = "svcId"

	tests := []struct {
		desc string
		// listed returns the addresses to put on svcId's allowlist.
		listed      func(owner, operator, revShare string) []string
		wantAllowed bool
	}{
		{
			desc:        "empty list allows anyone",
			listed:      func(_, _, _ string) []string { return nil },
			wantAllowed: true,
		},
		{
			desc:        "neither owner nor operator listed",
			listed:      func(_, _, _ string) []string { return []string{sample.AccAddressBech32()} },
			wantAllowed: false,
		},
		{
			desc:        "only owner listed",
			listed:      func(owner, _, _ string) []string { return []string{owner} },
			wantAllowed: true,
		},
		{
			desc:        "only operator listed",
			listed:      func(_, operator, _ string) []string { return []string{operator} },
			wantAllowed: true,
		},
		{
			desc:        "only a revenue share recipient listed",
			listed:      func(_, _, revShare string) []string { return []string{revShare} },
			wantAllowed: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			k, ctx := keepertest.SupplierKeeper(t)
			srv := keeper.NewMsgServerImpl(*k.Keeper)
			ownerAddr, operatorAddr, revShareAddr := sample.AccAddressBech32(), sample.AccAddressBech32(), sample.AccAddressBech32()

			for _, address := range test.listed(ownerAddr, operatorAddr, revShareAddr) {
				k.ServiceKeeper.SetSupplierAllowlistEntry(ctx, svcId, address)
			}

			stakeMsg, _ := newSupplierStakeMsg(ownerAddr, operatorAddr, 1000000, svcId)
			stakeMsg.Services[0].RevShare[0].Address = revShareAddr

			_, err := srv.StakeSupplier(ctx, stakeMsg)
			if test.wantAllowed {
				require.NoError(t, err)
				_, isFound := k.GetSupplier(ctx, operatorAddr)
				require.True(t, isFound)
				return
			}

			require.Error(t, err)
			require.Equal(t, codes.InvalidArgument, status.Code(err))
			require.ErrorContains(t, err, suppliertypes.ErrSupplierServiceNotAllowed.Error())
			require.ErrorContains(t, err, svcId)

			// Nothing was charged or persisted.
			require.Zero(t, k.SupplierBalanceMap[ownerAddr])
			require.Zero(t, k.SupplierBalanceMap[operatorAddr])
			_, isFound := k.GetSupplier(ctx, operatorAddr)
			require.False(t, isFound)
		})
	}
}

// A supplier staked while listed must not slip back in through a re-stake after
// its entry was removed.
func TestStakeSupplier_AllowlistRestakeAfterRemoval(t *testing.T) {
	const svcId = "svcId"

	k, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	ownerAddr, operatorAddr := sample.AccAddressBech32(), sample.AccAddressBech32()

	// Another entry keeps the list non-empty once the supplier is removed.
	k.ServiceKeeper.SetSupplierAllowlistEntry(ctx, svcId, sample.AccAddressBech32())
	k.ServiceKeeper.SetSupplierAllowlistEntry(ctx, svcId, operatorAddr)

	stakeMsg, _ := newSupplierStakeMsg(ownerAddr, operatorAddr, 1000000, svcId)
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	k.ServiceKeeper.RemoveSupplierAllowlistEntry(ctx, svcId, operatorAddr)

	restakeMsg, _ := newSupplierStakeMsg(ownerAddr, operatorAddr, 1000000, svcId)
	_, err = srv.StakeSupplier(ctx, restakeMsg)
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.ErrorContains(t, err, suppliertypes.ErrSupplierServiceNotAllowed.Error())
}

// While a service's first allowlist entry waits for its deferred sweep, an
// already staked, unlisted supplier cannot send any stake message that still
// lists the service, not even an endpoint change; a stake-only upstake by the
// owner still works, and dropping the service deactivates it at the next session
// start, one session before the sweep would have.
func TestStakeSupplier_AllowlistDeferralWindow(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	ownerAddr, operatorAddr := sample.AccAddressBech32(), sample.AccAddressBech32()

	stakeMsg, _ := newSupplierStakeMsg(ownerAddr, operatorAddr, 1000000, "svcId", "svcId2")
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)
	before := deactivationHeights(t, k, sdkCtx, operatorAddr)

	k.ServiceKeeper.SetSupplierAllowlistEntry(ctx, "svcId", sample.AccAddressBech32())
	k.ServiceKeeper.ScheduleInitialAllowlistSweep(ctx, "svcId")

	// An endpoint change that keeps svcId is rejected and changes nothing.
	restake, _ := newSupplierStakeMsg(ownerAddr, operatorAddr, 1000000, "svcId", "svcId2")
	restake.Services[0].Endpoints[0].Url = "http://changed:8080"
	_, err = srv.StakeSupplier(ctx, restake)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.ErrorContains(t, err, suppliertypes.ErrSupplierServiceNotAllowed.Error())
	require.Equal(t, before, deactivationHeights(t, k, sdkCtx, operatorAddr))

	// Even a change to the other service's endpoint is rejected while svcId stays listed in the msg.
	restake2, _ := newSupplierStakeMsg(ownerAddr, operatorAddr, 1000000, "svcId", "svcId2")
	restake2.Services[1].Endpoints[0].Url = "http://changed:8080"
	_, err = srv.StakeSupplier(ctx, restake2)
	require.ErrorContains(t, err, suppliertypes.ErrSupplierServiceNotAllowed.Error())

	// A stake-only upstake by the owner succeeds and leaves svcId active.
	upstake := &suppliertypes.MsgStakeSupplier{
		Signer: ownerAddr, OwnerAddress: ownerAddr, OperatorAddress: operatorAddr,
		Stake: &cosmostypes.Coin{Denom: "upokt", Amount: stakeMsg.Stake.Amount.AddRaw(1)},
	}
	_, err = srv.StakeSupplier(ctx, upstake)
	require.NoError(t, err)
	require.Equal(t, int64(sharedtypes.NoDeactivationHeight), deactivationHeights(t, k, sdkCtx, operatorAddr)["svcId"])

	// Dropping svcId succeeds; it deactivates at the next session start.
	dropped, _ := newSupplierStakeMsg(ownerAddr, operatorAddr, upstake.Stake.Amount.Int64(), "svcId2")
	dropped.Services[0].Endpoints[0].Url = "http://changed:8080"
	_, err = srv.StakeSupplier(ctx, dropped)
	require.NoError(t, err)
	supplier, found := k.GetSupplier(ctx, operatorAddr)
	require.True(t, found)
	next := nextSessionStart(k, sdkCtx)
	for _, config := range supplier.ServiceConfigHistory {
		if config.Service.ServiceId == "svcId" {
			require.Equal(t, next, config.DeactivationHeight)
		}
	}
}
