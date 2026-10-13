package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	testutilevents "github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// stakeThenLower stakes 3 services in the given order with enough stake for 3,
// then lowers the stored stake to cover `covered` services (as a slash would).
func stakeThenLower(t *testing.T, covered int64, serviceIds ...string) (keepertest.SupplierModuleKeepers, cosmostypes.Context, sharedtypes.Supplier) {
	t.Helper()
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	enableServiceCap(t, k, sdkCtx)
	ownerAddr, operatorAddr := sample.AccAddressBech32(), sample.AccAddressBech32()

	msg := multiServiceStakeMsg(t, k, sdkCtx, ownerAddr, operatorAddr, capMinStake+int64(len(serviceIds)-1)*capPrice, serviceIds...)
	_, err := srv.StakeSupplier(ctx, msg)
	require.NoError(t, err)

	supplier, found := k.GetDehydratedSupplier(ctx, operatorAddr)
	require.True(t, found)
	supplier.Stake = &cosmostypes.Coin{Denom: pocket.DenomuPOKT, Amount: math.NewInt(capMinStake + (covered-1)*capPrice)}
	k.SetDehydratedSupplier(ctx, supplier)
	return k, sdkCtx, supplier
}

func countedServiceIds(t *testing.T, k keepertest.SupplierModuleKeepers, ctx cosmostypes.Context, operatorAddr string) []string {
	t.Helper()
	sharedParams := k.SharedKeeper.GetParams(ctx)
	next := sharedtypes.GetNextSessionStartHeight(&sharedParams, ctx.BlockHeight())
	supplier, found := k.GetSupplier(ctx, operatorAddr)
	require.True(t, found)
	ids := make([]string, 0)
	for _, config := range supplier.ServiceConfigHistory {
		if suppliertypes.IsCountedServiceConfig(config, next) {
			ids = append(ids, config.Service.ServiceId)
		}
	}
	return ids
}

func TestEnforceServiceCap_KeepsOperatorOrderThroughTheStore(t *testing.T) {
	k, ctx, supplier := stakeThenLower(t, 2, "sol", "eth", "base")

	deactivated := k.EnforceServiceCap(ctx, &supplier, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_SLASH)
	require.Equal(t, []string{"base"}, deactivated)
	require.ElementsMatch(t, []string{"sol", "eth"}, countedServiceIds(t, k, ctx, supplier.OperatorAddress))

	sharedParams := k.SharedKeeper.GetParams(ctx)
	next := sharedtypes.GetNextSessionStartHeight(&sharedParams, ctx.BlockHeight())
	events := testutilevents.FilterEvents[*suppliertypes.EventSupplierServicesCapped](t, ctx.EventManager().Events())
	require.Len(t, events, 1)
	require.Equal(t, []string{"base"}, events[0].GetDeactivatedServiceIds())
	require.Equal(t, uint64(2), events[0].GetAllowedServices())
	require.Equal(t, next, events[0].GetEffectiveHeight())
	require.Equal(t, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_SLASH, events[0].GetReason())
}

func TestEnforceServiceCap_FallbackOrderForEmptyPriority(t *testing.T) {
	k, ctx, supplier := stakeThenLower(t, 2, "sol", "eth", "base")
	supplier.ServicePriority = nil // as for a supplier staked before the field existed
	k.SetDehydratedSupplier(ctx, supplier)

	deactivated := k.EnforceServiceCap(ctx, &supplier, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_PARAMS)
	require.Equal(t, []string{"sol"}, deactivated) // service ID order: base, eth, sol
}

func TestEnforceServiceCap_IdempotentAndUnderCapIsNoop(t *testing.T) {
	k, ctx, supplier := stakeThenLower(t, 2, "sol", "eth", "base")
	require.Len(t, k.EnforceServiceCap(ctx, &supplier, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_SLASH), 1)
	require.Empty(t, k.EnforceServiceCap(ctx, &supplier, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_SLASH))

	k2, ctx2, underCap := stakeThenLower(t, 3, "a", "b", "c")
	require.Empty(t, k2.EnforceServiceCap(ctx2, &underCap, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_SLASH))
	require.Empty(t, testutilevents.FilterEvents[*suppliertypes.EventSupplierServicesCapped](t, ctx2.EventManager().Events()))
}

func TestEnforceServiceCap_NeverUnbondsAndSkipsUnbonding(t *testing.T) {
	// Stake below min stake: left to the unbonding path, nothing deactivated.
	k, ctx, supplier := stakeThenLower(t, 0, "sol", "eth")
	require.Empty(t, k.EnforceServiceCap(ctx, &supplier, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_SLASH))
	stored, _ := k.GetDehydratedSupplier(ctx, supplier.OperatorAddress)
	require.False(t, stored.IsUnbonding())

	// Unbonding supplier: skipped.
	k2, ctx2, unbonding := stakeThenLower(t, 1, "sol", "eth")
	unbonding.UnstakeSessionEndHeight = 10
	require.Empty(t, k2.EnforceServiceCap(ctx2, &unbonding, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_SLASH))
}

func TestEnforceServiceCap_ZeroServicesUntouched(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	enableServiceCap(t, k, sdkCtx)
	ownerAddr, operatorAddr := sample.AccAddressBech32(), sample.AccAddressBech32()
	_, err := srv.StakeSupplier(ctx, &suppliertypes.MsgStakeSupplier{
		Signer: ownerAddr, OwnerAddress: ownerAddr, OperatorAddress: operatorAddr,
		Stake: &cosmostypes.Coin{Denom: pocket.DenomuPOKT, Amount: math.NewInt(capMinStake)},
	})
	require.NoError(t, err)

	supplier, _ := k.GetDehydratedSupplier(ctx, operatorAddr)
	require.Empty(t, k.EnforceServiceCap(sdkCtx, &supplier, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_PARAMS))
	stored, found := k.GetDehydratedSupplier(ctx, operatorAddr)
	require.True(t, found)
	require.False(t, stored.IsUnbonding())
	require.Empty(t, testutilevents.FilterEvents[*suppliertypes.EventSupplierServicesCapped](t, sdkCtx.EventManager().Events()))
}

func TestEnforceServiceCap_DisabledIsNoop(t *testing.T) {
	k, ctx, supplier := stakeThenLower(t, 1, "sol", "eth", "base")
	params := k.Keeper.GetParams(ctx)
	params.StakePerAdditionalService = nil
	require.NoError(t, k.SetParams(ctx, params))
	require.Empty(t, k.EnforceServiceCap(ctx, &supplier, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_PARAMS))
}
