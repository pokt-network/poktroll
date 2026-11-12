package integration_test

import (
	"strings"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtest "github.com/pokt-network/poktroll/testutil/shared"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	supplierkeeper "github.com/pokt-network/poktroll/x/supplier/keeper"
)

// A claim that settlement discards (its supplier record is gone by settlement time)
// must not count as a supplier of its (application, session) group: the other
// suppliers' floor is the application's budget divided by the claims that settle.
// The honest supplier is paid the same whether or not such a claim sits next to it.
func TestSettlePendingClaims_DiscardedClaimDoesNotLowerFloor(t *testing.T) {
	paidWithout := settleOverservingSupplier(t, false)
	paidWith := settleOverservingSupplier(t, true)
	require.True(t, paidWithout.IsPositive())
	require.Equal(t, paidWithout.String(), paidWith.String(), "a discarded claim must not change the honest supplier's payout")
}

// settleOverservingSupplier settles one over-serving supplier's claim, optionally next
// to a claim whose supplier is removed before settlement (a non-canonical operator
// removed by the v0.1.37 migration), and returns what the supplier was paid.
func settleOverservingSupplier(t *testing.T, withDiscardedClaim bool) math.Int {
	t.Helper()
	service := sharedtypes.Service{Id: "svc1", Name: "svcName1", ComputeUnitsPerRelay: 1, OwnerAddress: sample.AccAddressBech32()}

	appAddress := sample.AccAddressBech32()
	appStake := apptypes.DefaultMinStake
	application := apptypes.Application{
		Address:        appAddress,
		Stake:          &appStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: service.Id}},
	}

	newSupplier := func(operator, owner string) sharedtypes.Supplier {
		configs := []*sharedtypes.SupplierServiceConfig{{
			ServiceId: service.Id,
			RevShare:  []*sharedtypes.ServiceRevenueShare{{Address: owner, RevSharePercentage: 100}},
		}}
		stake := sdk.NewInt64Coin(pocket.DenomuPOKT, 1000)
		return sharedtypes.Supplier{
			OperatorAddress:      operator,
			OwnerAddress:         owner,
			Stake:                &stake,
			Services:             configs,
			ServiceConfigHistory: sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(operator, configs, 1, 0),
		}
	}
	honestAddress := sample.AccAddressBech32()
	honest := newSupplier(honestAddress, honestAddress)
	otherAddress := strings.ToUpper(honestAddress)
	other := newSupplier(otherAddress, sample.AccAddressBech32())

	opts := []testkeeper.TokenomicsModuleKeepersOptFn{
		testkeeper.WithService(service),
		testkeeper.WithApplication(application),
		testkeeper.WithSupplier(honest),
		testkeeper.WithBlockProposer(sample.ConsAddress(), sample.ValOperatorAddress()),
		testkeeper.WithProofRequirement(false),
		testkeeper.WithDefaultModuleBalances(),
	}
	if withDiscardedClaim {
		opts = append(opts, testkeeper.WithSupplier(other))
	}
	keepers, ctx := testkeeper.NewTokenomicsModuleKeepers(t, nil, opts...)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(1)

	const sessionN int64 = 4
	sharedParams := keepers.SharedKeeper.GetParams(sdkCtx)
	sharedParams.NumBlocksPerSession = uint64(sessionN)
	sharedParams.SessionGridAnchorHeight = 1
	sharedParams.SessionNumberAtAnchor = 1
	// Every relay is worth far more than the application can pay, so the claim is
	// capped at the supplier's floor and the payout measures the floor itself.
	sharedParams.ComputeUnitsToTokensMultiplier = sharedParams.ComputeUnitCostGranularity * 1_000_000
	require.NoError(t, keepers.SharedKeeper.SetParams(sdkCtx, sharedParams))

	serviceParams := keepers.ServiceKeeper.GetParams(ctx)
	serviceParams.TargetNumRelays = 1000
	require.NoError(t, keepers.ServiceKeeper.SetParams(ctx, serviceParams))
	_, err := keepers.ServiceKeeper.UpdateRelayMiningDifficulty(sdkCtx, map[string]uint64{service.Id: 1})
	require.NoError(t, err)

	sdkCtx = sdkCtx.WithBlockHeight(2)
	sessionRes, err := keepers.GetSession(sdkCtx, &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: appAddress, ServiceId: service.Id, BlockHeight: 2,
	})
	require.NoError(t, err)
	session := sessionRes.Session
	difficulty, ok := keepers.GetRelayMiningDifficulty(sdkCtx, service.Id)
	require.True(t, ok)
	keepers.UpsertClaim(sdkCtx, *prepareRealClaim(t, 50, honestAddress, session, &service, &difficulty))
	if withDiscardedClaim {
		keepers.UpsertClaim(sdkCtx, *prepareRealClaim(t, 50, otherAddress, session, &service, &difficulty))
		// The other supplier's record is removed before settlement, as the v0.1.37
		// migration removes non-canonical operators.
		concreteSupplierKeeper, isKeeper := keepers.SupplierKeeper.(*supplierkeeper.Keeper)
		require.True(t, isKeeper)
		numRemoved, _ := concreteSupplierKeeper.MigrateNonCanonicalSupplierAddresses(sdkCtx.WithBlockHeight(sessionN + 1))
		require.Equal(t, 1, numRemoved)
	}

	balanceBefore := supplierBalance(t, keepers, sdkCtx, honestAddress)
	settlementHeight := session.Header.SessionEndBlockHeight + sharedtypes.GetSessionEndToProofWindowCloseBlocks(&sharedParams) + 1
	settleCtx := sdkCtx.WithBlockHeight(settlementHeight)
	settled, _, numDiscarded, err := keepers.SettlePendingClaims(settleCtx)
	require.NoError(t, err)
	require.Equal(t, 1, int(settled.GetNumClaims()))
	if withDiscardedClaim {
		require.Equal(t, uint64(1), numDiscarded)
	} else {
		require.Equal(t, uint64(0), numDiscarded)
	}
	return supplierBalance(t, keepers, settleCtx, honestAddress).Sub(balanceBefore)
}

func supplierBalance(t *testing.T, keepers testkeeper.TokenomicsModuleKeepers, ctx sdk.Context, address string) math.Int {
	t.Helper()
	res, err := keepers.Balance(ctx, &banktypes.QueryBalanceRequest{Address: address, Denom: pocket.DenomuPOKT})
	require.NoError(t, err)
	return res.GetBalance().Amount
}
