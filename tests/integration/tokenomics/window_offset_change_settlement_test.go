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

// TestWindowOffsetChange_CrossSessionClaimStillSettles_Shrink encodes the
// cross-session window-offset orphan class (O2), SHRINK direction. The claim is
// created under a larger proof window; mid-session governance shrinks the proof
// window; the change is deferred to the next session boundary and promoted there;
// at the claim's original settlement height (under OLD offsets) live offsets give
// a SMALLER tail and therefore a DIFFERENT expiringSessionEndHeight than the
// claim's stored sessionEndHeight. Before the per-epoch candidate scan in
// settlement, the loop missed the claim. The fix walks recent params history
// epochs and computes a candidate sessionEndHeight under each epoch's offsets, so
// the claim is located under the OLD epoch's offsets.
func TestWindowOffsetChange_CrossSessionClaimStillSettles_Shrink(t *testing.T) {
	runWindowOffsetCrossSessionTest(t, 2 /* oldProofClose */, 1 /* newProofClose */)
}

// TestWindowOffsetChange_CrossSessionClaimStillSettles_Grow is the symmetric case:
// growing a window offset. The claim is created under a smaller proof window;
// mid-session governance grows the proof window; under live (new, larger) offsets
// at the claim's original settlement height, expiringSessionEndHeight is LOWER
// than the claim's stored sessionEndHeight. The same per-epoch candidate scan
// locates the claim under the OLD epoch's offsets and settles it at the
// originally-scheduled height instead of letting it drift to a later block under
// the new larger window.
func TestWindowOffsetChange_CrossSessionClaimStillSettles_Grow(t *testing.T) {
	runWindowOffsetCrossSessionTest(t, 1 /* oldProofClose */, 2 /* newProofClose */)
}

// runWindowOffsetCrossSessionTest is the shared body for the O2 cross-session
// orphan tests. Sets up an in-flight claim under oldProofClose, changes
// ProofWindowCloseOffsetBlocks to newProofClose mid-session, runs the EndBlocker
// at the boundary to promote, then settles at the claim's ORIGINAL settlement
// height (under oldProofClose). Asserts the claim settles regardless of the
// direction of the offset change.
func runWindowOffsetCrossSessionTest(t *testing.T, oldProofClose, newProofClose int64) {
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

	// --- Change ProofWindowCloseOffsetBlocks while the session is in flight ---
	// Direction (shrink/grow) is parameterized via oldProofClose/newProofClose.
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
