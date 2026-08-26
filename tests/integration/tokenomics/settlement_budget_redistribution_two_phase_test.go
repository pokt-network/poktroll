package integration_test

import (
	"fmt"
	"math/big"
	"testing"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/pkg/encoding"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtest "github.com/pokt-network/poktroll/testutil/shared"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// TestSettlementBudgetRedistribution_TwoSuppliers_ThroughSettlePendingClaims drives the
// overservicing bonus through the REAL two-phase settlement entry point (SettlePendingClaims:
// Phase 1 collect, Phase 1.5 AccumulateClaimBudget, Phase 2 ensureClaimAmountLimits) with two
// suppliers in one (application, session) group: one claiming well above its floor and one
// well below it.
//
// The keeper-level redistribution tests hand-drive AccumulateClaimBudget and
// ProcessTokenLogicModules, and the cupr-change integration test uses a single supplier, so
// neither exercises the bonus with real competition inside the session. This test pins the
// end-to-end payout:
//
//	m == 1  -> heavy supplier capped at exactly floor (legacy head-split).
//	m == 2  -> heavy supplier paid floor + unused (the light supplier's leftover), which here is
//	           below 2*floor, so the budget - not the multiplier - binds.
//	m == 10 -> identical to m == 2: once the multiplier stops binding, raising it pays nothing more.
//
// Amounts are asserted exactly, in settlement terms, by replicating the stake-terms budget
// arithmetic (floor = stake / numPendingSessions / N; global_inflation_per_claim gross-up) rather
// than by loose inequalities, so a regression in either phase's accounting fails loudly.
//
// Mainnet reference (h894793, first settlement under overservicing_bonus_multiplier=2): the capped
// supplier settled at exactly 2*floor, not floor - the settled==floor identity used by offchain
// analysis holds ONLY at m == 1.
func TestSettlementBudgetRedistribution_TwoSuppliers_ThroughSettlePendingClaims(t *testing.T) {
	const (
		computeUnitsPerRelay uint64 = 100
		heavyNumRelays       uint64 = 60
		lightNumRelays       uint64 = 1
		appStakeAmount       int64  = 100_000_000
	)

	for _, multiplier := range []uint64{1, 2, 10} {
		t.Run(fmt.Sprintf("multiplier_%d", multiplier), func(t *testing.T) {
			serviceOwner := sample.AccAddressBech32()
			service := sharedtypes.Service{
				Id:                   "svc1",
				Name:                 "svcName1",
				ComputeUnitsPerRelay: computeUnitsPerRelay,
				OwnerAddress:         serviceOwner,
			}

			appAddress := sample.AccAddressBech32()
			appStake := sdk.NewInt64Coin(pocket.DenomuPOKT, appStakeAmount)
			application := apptypes.Application{
				Address:        appAddress,
				Stake:          &appStake,
				ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: service.Id}},
			}

			newSupplier := func() sharedtypes.Supplier {
				addr := sample.AccAddressBech32()
				serviceConfigs := []*sharedtypes.SupplierServiceConfig{{
					ServiceId: service.Id,
					RevShare:  []*sharedtypes.ServiceRevenueShare{{Address: addr, RevSharePercentage: 100}},
				}}
				stake := sdk.NewInt64Coin(pocket.DenomuPOKT, 1000)
				return sharedtypes.Supplier{
					OperatorAddress:      addr,
					OwnerAddress:         addr,
					Stake:                &stake,
					Services:             serviceConfigs,
					ServiceConfigHistory: sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(addr, serviceConfigs, 1, 0),
				}
			}
			heavySupplier := newSupplier()
			lightSupplier := newSupplier()

			keepers, ctx := testkeeper.NewTokenomicsModuleKeepers(t, nil,
				testkeeper.WithService(service),
				testkeeper.WithApplication(application),
				testkeeper.WithSupplier(heavySupplier),
				testkeeper.WithSupplier(lightSupplier),
				testkeeper.WithBlockProposer(sample.ConsAddress(), sample.ValOperatorAddress()),
				testkeeper.WithProofRequirement(false),
				testkeeper.WithDefaultModuleBalances(),
			)
			sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(1)

			sharedParams := keepers.SharedKeeper.GetParams(sdkCtx)
			sharedParams.NumBlocksPerSession = 4
			sharedParams.SessionGridAnchorHeight = 1
			sharedParams.SessionNumberAtAnchor = 1
			// Price one relay at computeUnitsPerRelay * 10_000 upokt so a handful of relays
			// spans the app's budget without inserting thousands of leaves into the SMST.
			sharedParams.ComputeUnitsToTokensMultiplier = sharedParams.ComputeUnitCostGranularity * 10_000
			sharedParams.SupplierUnbondingPeriodSessions = 4
			sharedParams.ApplicationUnbondingPeriodSessions = 4
			sharedParams.GatewayUnbondingPeriodSessions = 4
			require.NoError(t, keepers.SharedKeeper.SetParams(sdkCtx, sharedParams))

			tokenomicsParams := keepers.Keeper.GetParams(sdkCtx)
			tokenomicsParams.OverservicingBonusMultiplier = multiplier
			require.NoError(t, keepers.Keeper.SetParams(sdkCtx, tokenomicsParams))

			// Keep relay mining difficulty at its base so estimated == claimed relays.
			serviceParams := keepers.ServiceKeeper.GetParams(ctx)
			serviceParams.TargetNumRelays = 1_000_000
			require.NoError(t, keepers.ServiceKeeper.SetParams(ctx, serviceParams))
			_, err := keepers.ServiceKeeper.UpdateRelayMiningDifficulty(sdkCtx, map[string]uint64{service.Id: 1})
			require.NoError(t, err)

			sdkCtx = sdkCtx.WithBlockHeight(2)
			sessionRes, err := keepers.GetSession(sdkCtx, &sessiontypes.QueryGetSessionRequest{
				ApplicationAddress: appAddress,
				ServiceId:          service.Id,
				BlockHeight:        sdkCtx.BlockHeight(),
			})
			require.NoError(t, err)
			session := sessionRes.Session
			require.Len(t, session.Suppliers, 2, "both suppliers must be selected into the session")

			relayMiningDifficulty, ok := keepers.GetRelayMiningDifficulty(sdkCtx, service.Id)
			require.True(t, ok)
			heavyClaim := prepareRealClaim(t, heavyNumRelays, heavySupplier.OperatorAddress, session, &service, &relayMiningDifficulty)
			lightClaim := prepareRealClaim(t, lightNumRelays, lightSupplier.OperatorAddress, session, &service, &relayMiningDifficulty)

			heavyClaimed, err := heavyClaim.GetClaimeduPOKT(sharedParams, relayMiningDifficulty)
			require.NoError(t, err)
			lightClaimed, err := lightClaim.GetClaimeduPOKT(sharedParams, relayMiningDifficulty)
			require.NoError(t, err)

			// --- Expected payout, replicating the settlement budget arithmetic ---
			globalInflationRat, err := encoding.Float64ToRat(tokenomicsParams.GlobalInflationPerClaim)
			require.NoError(t, err)
			toStakeTerms := func(settlement math.Int) math.Int {
				// settlement + ceil(settlement * global_inflation_per_claim)
				inflation := new(big.Rat).Mul(new(big.Rat).SetInt(settlement.BigInt()), globalInflationRat)
				quo, rem := new(big.Int).QuoRem(inflation.Num(), inflation.Denom(), new(big.Int))
				if rem.Sign() != 0 {
					quo.Add(quo, big.NewInt(1))
				}
				return settlement.Add(math.NewIntFromBigInt(quo))
			}
			toSettlementTerms := func(stake math.Int) math.Int {
				// floor(stake / (1 + global_inflation_per_claim))
				divisor := new(big.Rat).Add(new(big.Rat).SetInt64(1), globalInflationRat)
				quotient := new(big.Rat).Quo(new(big.Rat).SetInt(stake.BigInt()), divisor)
				return math.NewIntFromBigInt(new(big.Int).Quo(quotient.Num(), quotient.Denom()))
			}

			numPendingSessions := sharedtypes.GetNumPendingSessions(&sharedParams)
			floor := appStake.Amount.QuoRaw(numPendingSessions).QuoRaw(2)
			heavyStake := toStakeTerms(heavyClaimed.Amount)
			lightStake := toStakeTerms(lightClaimed.Amount)
			require.True(t, heavyStake.GT(floor), "heavy supplier must claim above its floor")
			require.True(t, lightStake.LT(floor), "light supplier must claim below its floor")

			unused := floor.Sub(lightStake)
			maxClaimable := floor
			if multiplier > 1 {
				// Single overservicer: bonus = unused * excess / totalExcess = unused.
				maxClaimable = floor.Add(unused)
			}
			multiplierCap := floor.MulRaw(int64(multiplier))
			if maxClaimable.GT(multiplierCap) {
				maxClaimable = multiplierCap
			}
			if maxClaimable.GT(heavyStake) {
				maxClaimable = heavyStake
			}
			expectedHeavySettled := toSettlementTerms(maxClaimable)
			if multiplier > 1 {
				require.True(t, floor.Add(unused).LT(multiplierCap),
					"fixture must be budget-bound (floor+unused < m*floor) so m=2 and m=10 pay the same")
			}

			// --- Settle through the real two-phase path ---
			tail := sharedtypes.GetSessionEndToProofWindowCloseBlocks(&sharedParams)
			sdkCtx = sdkCtx.WithBlockHeight(session.Header.SessionEndBlockHeight + tail + 1)
			keepers.UpsertClaim(sdkCtx, *heavyClaim)
			keepers.UpsertClaim(sdkCtx, *lightClaim)

			settledResults, expiredResults, numDiscardedFaultyClaims, err := keepers.SettlePendingClaims(sdkCtx)
			require.NoError(t, err)
			require.Equal(t, uint64(0), numDiscardedFaultyClaims)
			require.Equal(t, 2, int(settledResults.GetNumClaims()))
			require.Equal(t, 0, int(expiredResults.GetNumClaims()))

			settledBySupplier := map[string]math.Int{}
			for _, event := range sdkCtx.EventManager().Events() {
				if event.Type != "pocket.tokenomics.EventClaimSettled" {
					continue
				}
				msg, err := sdk.ParseTypedEvent(abci.Event(event))
				require.NoError(t, err)
				claimSettled, ok := msg.(*tokenomicstypes.EventClaimSettled)
				require.True(t, ok)
				settledCoin, err := sdk.ParseCoinNormalized(claimSettled.SettledUpokt)
				require.NoError(t, err)
				settledBySupplier[claimSettled.SupplierOperatorAddress] = settledCoin.Amount
			}
			require.Len(t, settledBySupplier, 2)

			require.Equal(t, lightClaimed.Amount.String(), settledBySupplier[lightSupplier.OperatorAddress].String(),
				"a supplier below its floor is always paid in full")
			require.Equal(t, expectedHeavySettled.String(), settledBySupplier[heavySupplier.OperatorAddress].String(),
				"heavy supplier payout at overservicing_bonus_multiplier=%d (floor=%s stake terms, unused=%s)",
				multiplier, floor, unused)
			if multiplier == 1 {
				require.Equal(t, toSettlementTerms(floor).String(), settledBySupplier[heavySupplier.OperatorAddress].String(),
					"m=1 must reproduce the legacy head-split cap exactly")
			} else {
				require.True(t, settledBySupplier[heavySupplier.OperatorAddress].GT(toSettlementTerms(floor)),
					"m>1 must pay the heavy supplier above its floor")
			}
		})
	}
}
