package integration_test

import (
	"fmt"
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

// TestCuprChange_InFlightClaimStillSettles_UnderRedistribution covers the interaction
// between two independently-developed consensus changes that both modified the same
// settlement function, and which are combined for the first time in this release:
//
//   - The compute_units_per_relay (cupr) session-start pin, which made settlement
//     validate numRelays * cupr against the cupr effective at the claim's SESSION-START
//     height rather than the live value.
//   - Settlement budget redistribution, which demoted the per-supplier head-split cap to
//     a floor and routed settlement through the per-(application, session) budget
//     accumulation phases. That branch had removed the applicationInitialStake lookup
//     from this function while still validating against the LIVE service cupr.
//
// Neither source branch could test the combination, and reconciling them required
// choosing which side of the cupr check survived. This test pins that choice: a claim
// mined under the old cupr must still settle after a mid-session cupr change, at every
// overservicing_bonus_multiplier setting — with redistribution off (the shipped default)
// and with it on.
//
// If the merge had kept redistribution's live-cupr check instead of the pin, every case
// here inverts to numDiscardedFaultyClaims == 1 and numSettled == 0, which is exactly the
// mainnet failure the pin exists to prevent.
func TestCuprChange_InFlightClaimStillSettles_UnderRedistribution(t *testing.T) {
	// Multipliers span the three meaningful regimes of the redistribution param:
	//   0  - unset / clobbered; coerced to 1 at read time, so it must behave as legacy.
	//   1  - the value seeded by the v0.1.35 upgrade handler; exact legacy head-split cap.
	//   10 - redistribution actively enabled by governance.
	for _, overservicingBonusMultiplier := range []uint64{0, 1, 10} {
		t.Run(fmt.Sprintf("multiplier_%d", overservicingBonusMultiplier), func(t *testing.T) {
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

			sharedParams := keepers.SharedKeeper.GetParams(sdkCtx)
			sharedParams.NumBlocksPerSession = uint64(sessionN)
			sharedParams.SessionGridAnchorHeight = 1
			sharedParams.SessionNumberAtAnchor = 1
			sharedParams.ComputeUnitsToTokensMultiplier = sharedParams.ComputeUnitCostGranularity
			sharedParams.SupplierUnbondingPeriodSessions = 4
			sharedParams.ApplicationUnbondingPeriodSessions = 4
			sharedParams.GatewayUnbondingPeriodSessions = 4
			require.NoError(t, keepers.SharedKeeper.SetParams(sdkCtx, sharedParams))

			// Put the redistribution param under test into effect.
			tokenomicsParams := keepers.Keeper.GetParams(sdkCtx)
			tokenomicsParams.OverservicingBonusMultiplier = overservicingBonusMultiplier
			require.NoError(t, keepers.Keeper.SetParams(sdkCtx, tokenomicsParams))

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

			// Build the claim as the RelayMiner would have mined it: weighted by the cupr
			// in effect at session start (still the old value at this point).
			relayMiningDifficulty, ok := keepers.GetRelayMiningDifficulty(sdkCtx, service.Id)
			require.True(t, ok)
			claim := prepareRealClaim(t, numRelays, supplierAddress, inFlightSession, &service, &relayMiningDifficulty)

			// --- The service owner changes cupr old -> new AFTER the session started ---
			concreteService, ok := keepers.ServiceKeeper.(*servicekeeper.Keeper)
			require.True(t, ok, "expected a concrete service keeper")

			changeCtx := sdkCtx.WithBlockHeight(3) // still inside the in-flight session
			require.NoError(t, concreteService.SnapshotServiceComputeUnitsPerRelayChange(
				changeCtx, service.Id, oldComputeUnitsPerRelay, newComputeUnitsPerRelay,
			))
			liveService := service
			liveService.ComputeUnitsPerRelay = newComputeUnitsPerRelay
			concreteService.SetService(changeCtx, liveService)

			// Sanity: the live cupr moved, the session-start lookup did not.
			pinnedCupr, found := concreteService.GetServiceComputeUnitsPerRelayAtHeight(
				changeCtx, service.Id, inFlightSession.Header.SessionStartBlockHeight,
			)
			require.True(t, found)
			require.Equal(t, oldComputeUnitsPerRelay, pinnedCupr,
				"cupr at the session-start height must remain the old value after a mid-session change")

			// --- Settle the in-flight claim after the proof window closes ---
			settlementHeight := inFlightSession.Header.SessionEndBlockHeight + tail + 1
			sdkCtx = sdkCtx.WithBlockHeight(settlementHeight)

			keepers.UpsertClaim(sdkCtx, *claim)

			settledResult, expiredResult, numDiscardedFaultyClaims, err := keepers.SettlePendingClaims(sdkCtx)
			require.NoError(t, err)

			require.Equal(t, uint64(0), numDiscardedFaultyClaims,
				"claim mined under session-start cupr must not be discarded after a mid-session cupr change "+
					"(overservicing_bonus_multiplier=%d); a non-zero count here means settlement fell back to "+
					"validating against the LIVE cupr", overservicingBonusMultiplier)
			require.Equal(t, 1, int(settledResult.GetNumClaims()),
				"the in-flight claim must settle at overservicing_bonus_multiplier=%d", overservicingBonusMultiplier)
			require.Equal(t, 0, int(expiredResult.GetNumClaims()), "the in-flight claim must not expire")

			numComputeUnits, err := settledResult.GetNumComputeUnits()
			require.NoError(t, err)
			require.Greater(t, int(numComputeUnits), 0,
				"settled claim must carry compute units (supplier paid) at overservicing_bonus_multiplier=%d",
				overservicingBonusMultiplier)
		})
	}
}
