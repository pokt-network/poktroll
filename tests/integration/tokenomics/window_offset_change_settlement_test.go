package integration_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtest "github.com/pokt-network/poktroll/testutil/shared"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedkeeper "github.com/pokt-network/poktroll/x/shared/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TestWindowOffsetChange_CrossSessionClaimStillSettles is the encoded repro for the
// cross-session window-offset orphan class (O2). It creates a claim under one set of
// window offsets, changes a window offset BEFORE the claim's proof window closes
// (the claim's lifecycle straddles a session boundary at which the new offsets are
// promoted to live), then advances to the original settlement height and asserts the
// claim still settles. Before the per-epoch candidate sessionEndHeight scan in
// settlement, the loop computed expiringSessionEndHeight against LIVE offsets only —
// after the promotion live's E no longer matched the claim's actual stored
// sessionEndHeight, orphaning the claim forever.
func TestWindowOffsetChange_CrossSessionClaimStillSettles(t *testing.T) {
	service := sharedtypes.Service{
		Id:                   "svc1",
		Name:                 "svcName1",
		ComputeUnitsPerRelay: 1,
		OwnerAddress:         sample.AccAddressBech32(),
	}

	appAddress := sample.AccAddressBech32()
	appStake := apptypes.DefaultMinStake.Add(apptypes.DefaultMinStake)
	application := apptypes.Application{
		Address:        appAddress,
		Stake:          &appStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: service.Id}},
	}

	supplierAddress := sample.AccAddressBech32()
	supplierServiceConfigs := []*sharedtypes.SupplierServiceConfig{
		{
			ServiceId: service.Id,
			RevShare: []*sharedtypes.ServiceRevenueShare{
				{Address: supplierAddress, RevSharePercentage: 100},
			},
		},
	}
	supplierStake := sdk.NewInt64Coin(pocket.DenomuPOKT, 1000)
	supplier := sharedtypes.Supplier{
		OperatorAddress:      supplierAddress,
		OwnerAddress:         supplierAddress,
		Stake:                &supplierStake,
		Services:             supplierServiceConfigs,
		ServiceConfigHistory: sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(supplierAddress, supplierServiceConfigs, 1, 0),
	}

	keepers, ctx := testkeeper.NewTokenomicsModuleKeepers(t, nil,
		testkeeper.WithService(service),
		testkeeper.WithApplication(application),
		testkeeper.WithSupplier(supplier),
		testkeeper.WithBlockProposer(sample.ConsAddress(), sample.ValOperatorAddress()),
		testkeeper.WithProofRequirement(false),
		testkeeper.WithDefaultModuleBalances(),
	)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(1)

	const (
		n                int64 = 4
		oldProofClose    int64 = 2
		newProofClose    int64 = 1 // shrink — the orphan-inducing direction
		gracePeriod      int64 = 1
		claimWindowOpen  int64 = 1
		claimWindowClose int64 = 2
		proofWindowOpen  int64 = 0
	)

	// Anchored at the genesis grid (N=n anchored at block 1), CUTTM=1, unbonding periods
	// large enough to satisfy ValidateBasic for both offset configurations.
	sharedParams := keepers.SharedKeeper.GetParams(sdkCtx)
	sharedParams.NumBlocksPerSession = uint64(n)
	sharedParams.SessionGridAnchorHeight = 1
	sharedParams.SessionNumberAtAnchor = 1
	sharedParams.ComputeUnitsToTokensMultiplier = sharedParams.ComputeUnitCostGranularity
	sharedParams.GracePeriodEndOffsetBlocks = uint64(gracePeriod)
	sharedParams.ClaimWindowOpenOffsetBlocks = uint64(claimWindowOpen)
	sharedParams.ClaimWindowCloseOffsetBlocks = uint64(claimWindowClose)
	sharedParams.ProofWindowOpenOffsetBlocks = uint64(proofWindowOpen)
	sharedParams.ProofWindowCloseOffsetBlocks = uint64(oldProofClose)
	sharedParams.SupplierUnbondingPeriodSessions = 4
	sharedParams.ApplicationUnbondingPeriodSessions = 4
	sharedParams.GatewayUnbondingPeriodSessions = 4
	require.NoError(t, keepers.SharedKeeper.SetParams(sdkCtx, sharedParams))

	concreteShared, ok := keepers.SharedKeeper.(*sharedkeeper.Keeper)
	require.True(t, ok, "expected a concrete shared keeper")

	oldTail := sharedtypes.GetSessionEndToProofWindowCloseBlocks(&sharedParams)

	// Service difficulty so claim creation has a defined multiplier.
	serviceParams := keepers.ServiceKeeper.GetParams(ctx)
	serviceParams.TargetNumRelays = 1000
	require.NoError(t, keepers.ServiceKeeper.SetParams(ctx, serviceParams))
	_, err := keepers.ServiceKeeper.UpdateRelayMiningDifficulty(sdkCtx, map[string]uint64{service.Id: 1})
	require.NoError(t, err)

	// --- Resolve the in-flight session under the OLD offsets (session [1, n]) ---
	sdkCtx = sdkCtx.WithBlockHeight(2) // mid old-session
	inFlightRes, err := keepers.GetSession(sdkCtx, &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: appAddress,
		ServiceId:          service.Id,
		BlockHeight:        sdkCtx.BlockHeight(),
	})
	require.NoError(t, err)
	inFlightSession := inFlightRes.Session
	require.Equal(t, int64(1), inFlightSession.Header.SessionStartBlockHeight)
	require.Equal(t, n, inFlightSession.Header.SessionEndBlockHeight)

	// --- Shrink ProofWindowCloseOffsetBlocks while the session is in flight ---
	// The change is deferred (#543 Option B + session-timing-deferral extension): live
	// offsets are unchanged until the next session boundary (n+1).
	sharedMsgSrv := sharedkeeper.NewMsgServerImpl(*concreteShared)
	_, err = sharedMsgSrv.UpdateParam(sdkCtx, &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamProofWindowCloseOffsetBlocks,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: uint64(newProofClose)},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(oldProofClose),
		keepers.SharedKeeper.GetParams(sdkCtx).ProofWindowCloseOffsetBlocks,
		"window-offset change must not change live before the session boundary")

	// Advance to the boundary and run the shared EndBlocker to promote the new offsets
	// to live. From here on, live offsets give a DIFFERENT expiringSessionEndHeight
	// than the OLD offsets the claim was created under.
	const boundaryHeight = n + 1
	sdkCtx = sdkCtx.WithBlockHeight(boundaryHeight)
	require.NoError(t, concreteShared.EndBlocker(sdkCtx))
	require.Equal(t, uint64(newProofClose),
		keepers.SharedKeeper.GetParams(sdkCtx).ProofWindowCloseOffsetBlocks,
		"EndBlocker must promote the new offsets at the boundary")

	// --- Settle the claim at the height computed under the OLD offsets ---
	// settlementHeight = sessionEndHeight + oldTail + 1. Under live (new) offsets the
	// settlement loop would compute expiringSessionEndHeight = settlementHeight - newTail - 1
	// which DIFFERS from the claim's actual stored sessionEndHeight. Only the per-epoch
	// candidate scan added to the settlement loop can locate the claim by its stored
	// sessionEndHeight.
	settlementHeight := inFlightSession.Header.SessionEndBlockHeight + oldTail + 1
	sdkCtx = sdkCtx.WithBlockHeight(settlementHeight)

	// Sanity check: live offsets at the settlement height yield a DIFFERENT
	// expiringSessionEndHeight than the claim's stored sessionEndHeight. If this
	// assertion fails, the offsets / N values in the test setup no longer exercise
	// the cross-session orphan path the test is meant to cover.
	liveAtSettlement := keepers.SharedKeeper.GetParams(sdkCtx)
	liveTail := sharedtypes.GetSessionEndToProofWindowCloseBlocks(&liveAtSettlement)
	liveDerivedE := settlementHeight - liveTail - 1
	require.NotEqual(t, inFlightSession.Header.SessionEndBlockHeight, liveDerivedE,
		"test setup: live-derived expiringSessionEndHeight must differ from the claim's stored sessionEndHeight")

	relayMiningDifficulty, ok := keepers.GetRelayMiningDifficulty(sdkCtx, service.Id)
	require.True(t, ok)

	claim := prepareRealClaim(t, 1000, supplierAddress, inFlightSession, &service, &relayMiningDifficulty)
	keepers.UpsertClaim(sdkCtx, *claim)

	settledResult, expiredResult, numDiscardedFaultyClaims, err := keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)
	require.Equal(t, 1, int(settledResult.GetNumClaims()),
		"in-flight claim must settle despite the cross-session window-offset change")
	require.Equal(t, 0, int(expiredResult.GetNumClaims()), "in-flight claim must not expire")
	require.Equal(t, uint64(0), numDiscardedFaultyClaims)

	numComputeUnits, err := settledResult.GetNumComputeUnits()
	require.NoError(t, err)
	require.Greater(t, int(numComputeUnits), 0, "settled claim must carry compute units (supplier paid)")
}
