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

// TestSessionLengthChange_InFlightClaimStillSettles is the encoded localnet repro for #543.
// It changes num_blocks_per_session to a NON-DIVISOR value while a session is in flight and
// asserts that the in-flight session (a) keeps the exact same session id / boundaries after
// the change and (b) still settles and pays its supplier. Before the anchored session grid,
// changing N re-derived the in-flight session onto a different grid → session-id mismatch →
// the claim "disappeared" and the supplier was never paid.
func TestSessionLengthChange_InFlightClaimStillSettles(t *testing.T) {
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
		oldN int64 = 4
		newN int64 = 3 // non-divisor of 4 — the case that breaks the legacy modulo grid
	)

	// Start with N=oldN anchored at the genesis grid, CUTTM=1, and unbonding periods large
	// enough to satisfy ValidateBasic for BOTH N values (unbonding_sessions * N >= tail).
	sharedParams := keepers.SharedKeeper.GetParams(sdkCtx)
	sharedParams.NumBlocksPerSession = uint64(oldN)
	sharedParams.SessionGridAnchorHeight = 1
	sharedParams.SessionNumberAtAnchor = 1
	sharedParams.ComputeUnitsToTokensMultiplier = sharedParams.ComputeUnitCostGranularity
	sharedParams.SupplierUnbondingPeriodSessions = 4
	sharedParams.ApplicationUnbondingPeriodSessions = 4
	sharedParams.GatewayUnbondingPeriodSessions = 4
	require.NoError(t, keepers.SharedKeeper.SetParams(sdkCtx, sharedParams))

	concreteShared, ok := keepers.SharedKeeper.(*sharedkeeper.Keeper)
	require.True(t, ok, "expected a concrete shared keeper")

	tail := sharedtypes.GetSessionEndToProofWindowCloseBlocks(&sharedParams)

	// Pick a target number of relays that keeps the difficulty at a well-defined multiplier.
	serviceParams := keepers.ServiceKeeper.GetParams(ctx)
	serviceParams.TargetNumRelays = 1000
	require.NoError(t, keepers.ServiceKeeper.SetParams(ctx, serviceParams))

	// Update the relay mining difficulty so there's always a difficulty to retrieve.
	_, err := keepers.ServiceKeeper.UpdateRelayMiningDifficulty(sdkCtx, map[string]uint64{service.Id: 1})
	require.NoError(t, err)

	// --- Capture the in-flight session under N=oldN (session [1, oldN]) ---
	sdkCtx = sdkCtx.WithBlockHeight(2) // mid old-session
	inFlightRes, err := keepers.GetSession(sdkCtx, &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: appAddress,
		ServiceId:          service.Id,
		BlockHeight:        sdkCtx.BlockHeight(),
	})
	require.NoError(t, err)
	inFlightSession := inFlightRes.Session
	require.Equal(t, int64(1), inFlightSession.Header.SessionStartBlockHeight)
	require.Equal(t, oldN, inFlightSession.Header.SessionEndBlockHeight)
	inFlightSessionId := inFlightSession.Header.SessionId

	// --- Change num_blocks_per_session to a non-divisor while the session is in flight ---
	sharedMsgSrv := sharedkeeper.NewMsgServerImpl(*concreteShared)
	_, err = sharedMsgSrv.UpdateParam(sdkCtx, &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      sharedtypes.ParamNumBlocksPerSession,
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: uint64(newN)},
	})
	require.NoError(t, err)

	// The change is DEFERRED: live N is unchanged until the next session boundary (oldN+1).
	require.Equal(t, uint64(oldN), keepers.SharedKeeper.GetParams(sdkCtx).NumBlocksPerSession)

	// Advance to the boundary and run the shared EndBlocker, which promotes the new epoch.
	const boundaryHeight = oldN + 1
	sdkCtx = sdkCtx.WithBlockHeight(boundaryHeight)
	require.NoError(t, concreteShared.EndBlocker(sdkCtx))
	require.Equal(t, uint64(newN), keepers.SharedKeeper.GetParams(sdkCtx).NumBlocksPerSession)

	// --- Grid stability: the in-flight session is unchanged after the N change ---
	// Re-query the session at a height inside the OLD session; under the anchored grid it
	// resolves via the genesis epoch (N=oldN) and yields the SAME id and boundaries.
	postChangeRes, err := keepers.GetSession(sdkCtx.WithBlockHeight(3), &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: appAddress,
		ServiceId:          service.Id,
		BlockHeight:        3,
	})
	require.NoError(t, err)
	require.Equal(t, inFlightSessionId, postChangeRes.Session.Header.SessionId,
		"in-flight session id changed after num_blocks_per_session change (#543 regression)")
	require.Equal(t, int64(1), postChangeRes.Session.Header.SessionStartBlockHeight)
	require.Equal(t, oldN, postChangeRes.Session.Header.SessionEndBlockHeight)

	// --- The in-flight session still settles and pays its supplier ---
	settlementHeight := inFlightSession.Header.SessionEndBlockHeight + tail + 1
	sdkCtx = sdkCtx.WithBlockHeight(settlementHeight)

	relayMiningDifficulty, ok := keepers.GetRelayMiningDifficulty(sdkCtx, service.Id)
	require.True(t, ok)

	claim := prepareRealClaim(t, 1000, supplierAddress, inFlightSession, &service, &relayMiningDifficulty)
	keepers.UpsertClaim(sdkCtx, *claim)

	settledResult, expiredResult, numDiscardedFaultyClaims, err := keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)
	require.Equal(t, 1, int(settledResult.GetNumClaims()), "the in-flight claim must settle, not disappear")
	require.Equal(t, 0, int(expiredResult.GetNumClaims()), "the in-flight claim must not expire")
	require.Equal(t, uint64(0), numDiscardedFaultyClaims)

	numComputeUnits, err := settledResult.GetNumComputeUnits()
	require.NoError(t, err)
	require.Greater(t, int(numComputeUnits), 0, "settled claim must carry compute units (supplier paid)")
}
