package integration_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtest "github.com/pokt-network/poktroll/testutil/shared"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	servicekeeper "github.com/pokt-network/poktroll/x/service/keeper"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TestCuprChange_InFlightClaimStillSettles is the settlement-side counterpart to the
// compute_units_per_relay (cupr) session-start pin. It reproduces the mid-session cupr
// change that historically forfeited in-flight claims:
//
//   - The RelayMiner bakes the session-start cupr into the append-only SMST at mine time.
//   - The chain re-derives numRelays * cupr at settlement.
//
// Before the settlement pin, settlement read the LIVE service cupr. A cupr change between
// session start and settlement therefore made numRelays * liveCupr != treeSum, so the
// claim was discarded (EventClaimDiscarded) and the supplier was paid nothing — the exact
// ErrProofComputeUnitsMismatch forfeit, relocated from claim creation to settlement.
//
// With the pin, settlement resolves cupr at the session-start height, so the in-flight
// claim mined under the old cupr still settles after a change. On main (live cupr at
// settlement) this test fails: numDiscardedFaultyClaims == 1 and numSettled == 0.
func TestCuprChange_InFlightClaimStillSettles(t *testing.T) {
	const (
		oldComputeUnitsPerRelay uint64 = 100
		newComputeUnitsPerRelay uint64 = 200
		numRelays               uint64 = 1000
	)

	serviceOwner := sample.AccAddressBech32()
	service := sharedtypes.Service{
		Id:                   "svc1",
		Name:                 "svcName1",
		ComputeUnitsPerRelay: oldComputeUnitsPerRelay,
		OwnerAddress:         serviceOwner,
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

	const sessionN int64 = 4

	// Anchor the session grid at genesis with N=sessionN, a well-defined CUTTM, and
	// unbonding periods large enough for ValidateBasic.
	sharedParams := keepers.SharedKeeper.GetParams(sdkCtx)
	sharedParams.NumBlocksPerSession = uint64(sessionN)
	sharedParams.SessionGridAnchorHeight = 1
	sharedParams.SessionNumberAtAnchor = 1
	sharedParams.ComputeUnitsToTokensMultiplier = sharedParams.ComputeUnitCostGranularity
	sharedParams.SupplierUnbondingPeriodSessions = 4
	sharedParams.ApplicationUnbondingPeriodSessions = 4
	sharedParams.GatewayUnbondingPeriodSessions = 4
	require.NoError(t, keepers.SharedKeeper.SetParams(sdkCtx, sharedParams))

	// Ensure a well-defined relay mining difficulty exists for the service.
	serviceParams := keepers.ServiceKeeper.GetParams(ctx)
	serviceParams.TargetNumRelays = numRelays
	require.NoError(t, keepers.ServiceKeeper.SetParams(ctx, serviceParams))
	_, err := keepers.ServiceKeeper.UpdateRelayMiningDifficulty(sdkCtx, map[string]uint64{service.Id: 1})
	require.NoError(t, err)

	tail := sharedtypes.GetSessionEndToProofWindowCloseBlocks(&sharedParams)

	// Capture the in-flight session [1, sessionN].
	sdkCtx = sdkCtx.WithBlockHeight(2) // mid-session
	inFlightRes, err := keepers.GetSession(sdkCtx, &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: appAddress,
		ServiceId:          service.Id,
		BlockHeight:        sdkCtx.BlockHeight(),
	})
	require.NoError(t, err)
	inFlightSession := inFlightRes.Session
	require.Equal(t, int64(1), inFlightSession.Header.SessionStartBlockHeight)
	require.Equal(t, sessionN, inFlightSession.Header.SessionEndBlockHeight)

	// Build the claim the supplier mined under the OLD cupr (weight = oldComputeUnitsPerRelay).
	// service still carries oldComputeUnitsPerRelay here, so the tree sum is numApplicableRelays * old.
	relayMiningDifficulty, ok := keepers.GetRelayMiningDifficulty(sdkCtx, service.Id)
	require.True(t, ok)
	claim := prepareRealClaim(t, numRelays, supplierAddress, inFlightSession, &service, &relayMiningDifficulty)

	// --- The service owner changes cupr old -> new AFTER the session started ---
	// This is the exact state a mid-session MsgAddService update produces: the live
	// service cupr becomes new, while the cupr history records old effective before the
	// change (seeded at height 1) and new effective at the next session boundary.
	concreteService, ok := keepers.ServiceKeeper.(*servicekeeper.Keeper)
	require.True(t, ok, "expected a concrete service keeper")

	changeCtx := sdkCtx.WithBlockHeight(3) // still inside the in-flight session
	require.NoError(t, concreteService.SnapshotServiceComputeUnitsPerRelayChange(
		changeCtx, service.Id, oldComputeUnitsPerRelay, newComputeUnitsPerRelay,
	))
	liveService := service
	liveService.ComputeUnitsPerRelay = newComputeUnitsPerRelay
	concreteService.SetService(changeCtx, liveService)

	// Sanity: live cupr is now new, but the session-start lookup still resolves to old.
	require.Equal(t, newComputeUnitsPerRelay, mustGetService(t, concreteService, changeCtx, service.Id).ComputeUnitsPerRelay)
	pinnedCupr, found := concreteService.GetServiceComputeUnitsPerRelayAtHeight(changeCtx, service.Id, inFlightSession.Header.SessionStartBlockHeight)
	require.True(t, found)
	require.Equal(t, oldComputeUnitsPerRelay, pinnedCupr,
		"cupr at the session-start height must remain the old value after a mid-session change")

	// --- Settle the in-flight claim after the proof window closes ---
	settlementHeight := inFlightSession.Header.SessionEndBlockHeight + tail + 1
	sdkCtx = sdkCtx.WithBlockHeight(settlementHeight)

	keepers.UpsertClaim(sdkCtx, *claim)

	settledResult, expiredResult, numDiscardedFaultyClaims, err := keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// The core regression assertions: the claim mined under the old cupr must SETTLE, not
	// be discarded, despite the live cupr having changed. On main these invert
	// (numDiscardedFaultyClaims == 1, settled == 0).
	require.Equal(t, uint64(0), numDiscardedFaultyClaims,
		"claim mined under session-start cupr must not be discarded after a mid-session cupr change")
	require.Equal(t, 1, int(settledResult.GetNumClaims()), "the in-flight claim must settle")
	require.Equal(t, 0, int(expiredResult.GetNumClaims()), "the in-flight claim must not expire")

	numComputeUnits, err := settledResult.GetNumComputeUnits()
	require.NoError(t, err)
	require.Greater(t, int(numComputeUnits), 0, "settled claim must carry compute units (supplier paid)")
}

// mustGetService is a small helper to read a service from the concrete keeper in-test.
func mustGetService(t *testing.T, k *servicekeeper.Keeper, ctx sdk.Context, serviceId string) sharedtypes.Service {
	t.Helper()
	service, found := k.GetService(ctx, serviceId)
	require.True(t, found)
	return service
}
