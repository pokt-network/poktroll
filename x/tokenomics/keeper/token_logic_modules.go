package keeper

import (
	"context"
	"fmt"
	"math/big"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/telemetry"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	tlm "github.com/pokt-network/poktroll/x/tokenomics/token_logic_module"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// ProcessTokenLogicModules is the entrypoint for all TLM processing.
//
// It is responsible for running all the independent TLMs necessary to limit, burn, mint or transfer tokens as a result of the
// amount of work (i.e. relays, compute units) done in proportion to the global governance parameters.
//
// Prior to running the TLMs, it handles the business logic of converting the claimed
// amount to the actual settlement amount and handling the case for overserviced applications.
//
// IMPORTANT: It is assumed that the proof for the claim has been validated BEFORE calling this function.
func (k Keeper) ProcessTokenLogicModules(
	ctx context.Context,
	settlementContext *settlementContext,
	pendingResult *tokenomicstypes.ClaimSettlementResult,
) (cosmostypes.Coin, error) {
	logger := k.Logger().With("method", "ProcessTokenLogicModules")

	// Telemetry variable declaration to be emitted at the end of the function
	claimSettlementCoin := cosmostypes.NewCoin("upokt", math.NewInt(0))
	isSuccessful := false

	// This is emitted only when the function returns (successful or not)
	defer telemetry.EventSuccessCounter(
		"process_token_logic_modules",
		func() float32 {
			if claimSettlementCoin.Amount.BigInt() == nil {
				return 0
			}

			// Avoid out of range errors by converting to float64 first
			claimSettlementFloat64, _ := claimSettlementCoin.Amount.BigInt().Float64()
			return float32(claimSettlementFloat64)
		},
		func() bool { return isSuccessful },
	)

	// Retrieve & validate the session header
	sessionHeader := pendingResult.Claim.GetSessionHeader()
	if sessionHeader == nil {
		logger.Error("received a nil session header")
		return cosmostypes.Coin{}, tokenomicstypes.ErrTokenomicsClaimSessionHeaderNil
	}
	if err := sessionHeader.ValidateBasic(); err != nil {
		logger.Error("received an invalid session header", "error", err)
		return cosmostypes.Coin{}, tokenomicstypes.ErrTokenomicsClaimSessionHeaderInvalid
	}

	// Retrieve and validate the root of the claim to determine the amount of work done
	root := (smt.MerkleSumRoot)(pendingResult.Claim.GetRootHash())
	if !root.HasDigestSize(protocol.TrieHasherSize) {
		return cosmostypes.Coin{}, tokenomicstypes.ErrTokenomicsClaimRootHashInvalid.Wrapf(
			"root hash has invalid digest size (%d), expected (%d)",
			root.DigestSize(), protocol.TrieHasherSize,
		)
	}

	// Retrieve the sum (i.e. number of compute units) to determine the amount of work done
	numClaimComputeUnits, err := pendingResult.Claim.GetNumClaimedComputeUnits()
	if err != nil {
		return cosmostypes.Coin{}, tokenomicstypes.ErrTokenomicsClaimRootHashInvalid.Wrapf("failed to retrieve numClaimComputeUnits: %s", err)
	}
	// TODO_MAINNET_MIGRATION(@bryanchriswhite, @red-0ne): Fix the low-volume exploit here.
	// https://www.notion.so/buildwithgrove/RelayMiningDifficulty-and-low-volume-7aab3edf6f324786933af369c2fa5f01?pvs=4
	if numClaimComputeUnits == 0 {
		return cosmostypes.Coin{}, tokenomicstypes.ErrTokenomicsClaimRootHashInvalid.Wrap("root hash has zero relays")
	}

	numRelays, err := pendingResult.Claim.GetNumRelays()
	if err != nil {
		return cosmostypes.Coin{}, tokenomicstypes.ErrTokenomicsClaimRootHashInvalid.Wrapf("failed to retrieve numRelays: %s", err)
	}

	/*
		TODO_TECHDEBT(@olshansk): Fix the root.Count and root.Sum confusion.

		Because of how things have evolved, we are now using root.Count (numRelays)
		instead of root.Sum (numComputeUnits) to determine the amount of work done.

		This is because the compute_units_per_relay is a service specific (not request specific)
		parameter that will be maintained by the service owner to capture the average amount of
		resources (i.e. compute, storage, bandwidth, electricity, etc...) per request.

		Modifying this on a per request basis has been deemed too complex and not a mainnet blocker.
	*/

	// DEV_NOTE: two distinct shared params epochs are in play here, and they are NOT
	// interchangeable:
	//
	//   - sharedParams (LIVE) is used further below to project FUTURE heights from the
	//     CURRENT block (session end height, application unbonding height). Those must use
	//     the session grid in effect now; resolving them at session start would schedule
	//     unbonding against a stale num_blocks_per_session.
	//   - pricingParams (at session start) converts claimed work to uPOKT. It must match
	//     the epoch x/proof priced and proof-gated the claim under.
	sharedParams := settlementContext.GetSharedParams()
	pricingParams := settlementContext.GetSharedParamsAtSessionStart(ctx, sessionHeader.GetSessionStartBlockHeight())
	tokenomicsParams := settlementContext.GetTokenomicsParams()

	service, err := settlementContext.GetService(sessionHeader.ServiceId)
	if err != nil {
		return cosmostypes.Coin{}, err
	}

	relayMiningDifficulty, err := settlementContext.GetRelayMiningDifficulty(sessionHeader.ServiceId, sessionHeader.SessionStartBlockHeight)
	if err != nil {
		return cosmostypes.Coin{}, err
	}

	application, err := settlementContext.GetApplication(sessionHeader.ApplicationAddress)
	if err != nil {
		return cosmostypes.Coin{}, err
	}

	supplier, err := settlementContext.GetSupplier(pendingResult.Claim.GetSupplierOperatorAddress())
	if err != nil {
		return cosmostypes.Coin{}, err
	}

	// Resolve the compute_units_per_relay that was effective at the session start height,
	// NOT the current (possibly changed) service cupr. The RelayMiner bakes the
	// session-start cupr into the append-only SMST at mine time, so validating against a
	// mid-session cupr change would discard otherwise-valid in-flight claims. This mirrors
	// the session-start pin applied at claim creation and the difficulty-at-height lookup
	// above.
	sessionStartComputeUnitsPerRelay, err := settlementContext.GetServiceComputeUnitsPerRelay(sessionHeader.ServiceId, sessionHeader.SessionStartBlockHeight)
	if err != nil {
		return cosmostypes.Coin{}, err
	}

	// Ensure the number of compute units claimed is equal to the number of relays * CUPR
	expectedClaimComputeUnits := numRelays * sessionStartComputeUnitsPerRelay
	if numClaimComputeUnits != expectedClaimComputeUnits {
		return cosmostypes.Coin{}, tokenomicstypes.ErrTokenomicsClaimRootHashInvalid.Wrapf(
			"mismatch: claim compute units (%d) != number of relays (%d) * service compute units per relay (%d)",
			numClaimComputeUnits,
			numRelays,
			sessionStartComputeUnitsPerRelay,
		)
	}

	// Determine the total number of tokens being claimed (i.e. for the work completed)
	// by the supplier for the amount of work they did to service the application
	// in the session.
	claimSettlementCoin, err = pendingResult.Claim.GetClaimeduPOKT(pricingParams, relayMiningDifficulty)
	if err != nil {
		return cosmostypes.Coin{}, err
	}

	// Helpers for logging the same metadata throughout this function calls
	logger = logger.With(
		"num_relays", numRelays,
		"num_claim_compute_units", numClaimComputeUnits,
		"claim_settlement_upokt", claimSettlementCoin.Amount,
		"session_id", sessionHeader.GetSessionId(),
		"service_id", sessionHeader.GetServiceId(),
		"supplier_operator", supplier.OperatorAddress,
		"application", application.Address,
	)

	// Look up the per-(app, session) budget allocation used to demote the head-split cap to
	// a redistributable floor. Phase 1.5 of SettlePendingClaims precomputes this for every
	// group before Phase 2 runs, so in the normal settlement flow this is always a cache hit
	// carrying the full unused/excess aggregates needed for redistribution.
	//
	// getOrInitSessionBudget falls back to computing a floor-only budget (unused = 0,
	// totalExcess = 0 ⇒ no bonus ⇒ the legacy head-split cap) if the group was not
	// precomputed. This can only happen when ProcessTokenLogicModules is invoked outside the
	// two-phase flow (e.g. isolated unit tests); it is provably conservative (a supplier is
	// paid at most its floor, so the application can never be overpaid) and deterministic.
	sessionBudget, err := settlementContext.getOrInitSessionBudget(
		ctx,
		sessionHeader.ApplicationAddress,
		sessionHeader.SessionId,
		sessionHeader.SessionEndBlockHeight,
	)
	if err != nil {
		return cosmostypes.Coin{}, err
	}

	// global_inflation_per_claim as a big.Rat, converted once per settlement block rather
	// than per claim (it is the same constant for every claim in the block).
	globalInflationPerClaimRat, err := settlementContext.getGlobalInflationPerClaimRat()
	if err != nil {
		return cosmostypes.Coin{}, err
	}

	// Ensure the claim amount is within the limits set by RelayMining.
	// If not, update the settlement amount and emit relevant events.
	// TODO_IMPROVE: Consider pulling this out of Keeper#ProcessTokenLogicModules
	// and ensure claim amount limits are enforced before TLM processing.
	actualSettlementCoin, err := k.ensureClaimAmountLimits(ctx, logger, &tokenomicsParams, application, supplier, claimSettlementCoin, sessionBudget, globalInflationPerClaimRat, sessionHeader.ServiceId, sessionHeader.SessionEndBlockHeight)
	if err != nil {
		return cosmostypes.Coin{}, err
	}
	logger = logger.With("actual_settlement_upokt", actualSettlementCoin)
	logger.Info(fmt.Sprintf("About to start processing TLMs for (%d) compute units, equal to (%s) claimed", numClaimComputeUnits, actualSettlementCoin))

	if actualSettlementCoin.Amount.IsZero() {
		logger.Warn(fmt.Sprintf(
			"actual settlement coin is zero, skipping TLM processing, application %q stake %s",
			application.Address, application.Stake,
		))
		return actualSettlementCoin, nil
	}

	tlmCtx := tlm.TLMContext{
		TokenomicsParams:           tokenomicsParams,
		SettlementCoin:             actualSettlementCoin,
		SessionHeader:              pendingResult.Claim.GetSessionHeader(),
		Result:                     pendingResult,
		Service:                    service,
		Application:                application,
		Supplier:                   supplier,
		RelayMiningDifficulty:      &relayMiningDifficulty,
		StakingKeeper:              k.stakingKeeper,
		ValidatorRewardAccumulator: settlementContext.GetValidatorRewardAccumulator(),
	}

	// Execute all the token logic modules processors.
	// TLMs accumulate validator rewards in tlmCtx.ValidatorRewardAccumulator (#1758)
	// instead of calling distributeValidatorRewards per-claim. The accumulated totals
	// are flushed once per settlement batch in SettlePendingClaims, achieving perfect
	// precision via the Largest Remainder Method on the batched sum.
	for _, tokenLogicModule := range k.tokenLogicModules {
		tlmName := tokenLogicModule.GetId().String()
		logger.Info(fmt.Sprintf("Starting processing TLM: %q", tlmName))

		if err = tokenLogicModule.Process(ctx, logger, tlmCtx); err != nil {
			return cosmostypes.Coin{}, tokenomicstypes.ErrTokenomicsProcessingTLM.Wrapf("TLM %q: %s", tlmName, err)
		}

		logger.Info(fmt.Sprintf("Finished processing TLM: %q", tlmName))
	}

	// Unbond the application if it has less than the minimum stake.
	// Use the application from the TLM context as it may have been modified by the TLMs.
	//
	// Use the on-chain min_stake governance param, NOT apptypes.DefaultMinStake.
	// DefaultMinStake is the hardcoded genesis default (1 POKT); mainnet governance
	// has raised min_stake well above it (e.g. 1,000 POKT). Comparing against the
	// default left apps below the real min_stake staked indefinitely (issue #1846).
	//
	// Stake only decreases via settlement burn, so this check runs in the same block
	// an application first drops below min_stake — every future crossing is caught here.
	//
	// No pending-payment race: the session hydrator stops assigning the app to new
	// sessions once queryHeight > UnstakeSessionEndHeight (Application.IsActive), and the
	// unbonding period (ApplicationUnbondingPeriodSessions × NumBlocksPerSession = 1 session
	// = 60 blocks on mainnet) strictly exceeds the settlement lag (~33 blocks). So the last
	// session the app is assigned to settles before the app is removed and its stake returned.
	// Defensive: GetParams returns a zero-value Params{} (nil MinStake) if the
	// application params have never been written. That cannot happen on a chain
	// initialized via InitGenesis, but a nil deref here would halt the chain, so
	// fall back to DefaultMinStake (matching the pre-fix behavior) if unset.
	minStake := apptypes.DefaultMinStake
	if appMinStake := k.applicationKeeper.GetParams(ctx).MinStake; appMinStake != nil {
		minStake = *appMinStake
	}
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, cosmostypes.UnwrapSDKContext(ctx).BlockHeight())
	if tlmCtx.Application.Stake.Amount.LT(minStake.Amount) {
		// Mark the application as unbonding if it has less than the minimum stake.
		tlmCtx.Application.UnstakeSessionEndHeight = uint64(sessionEndHeight)
		unbondingEndHeight := apptypes.GetApplicationUnbondingHeight(&sharedParams, tlmCtx.Application)

		appUnbondingBeginEvent := &apptypes.EventApplicationUnbondingBegin{
			Application:        tlmCtx.Application,
			Reason:             apptypes.ApplicationUnbondingReason_APPLICATION_UNBONDING_REASON_BELOW_MIN_STAKE,
			SessionEndHeight:   sessionEndHeight,
			UnbondingEndHeight: unbondingEndHeight,
		}

		sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
		if err = sdkCtx.EventManager().EmitTypedEvent(appUnbondingBeginEvent); err != nil {
			err = apptypes.ErrAppEmitEvent.Wrapf("(%+v): %s", appUnbondingBeginEvent, err)
			logger.Error(err.Error())
			return cosmostypes.Coin{}, err
		}

		// Update the application in the keeper to persist the unbonding state.
		k.applicationKeeper.SetApplication(ctx, *tlmCtx.Application)
	}

	// TODO_IMPROVE: If the application stake has dropped to (near?) zero:
	// - Unstake it
	// - Emit an event
	// - Ensure this doesn't happen
	// - Document the decision

	// Update isSuccessful to true for telemetry
	isSuccessful = true
	return actualSettlementCoin, nil
}

// ensureClaimAmountLimits caps a supplier's settlement to the budget the application
// committed for the session (settlement budget redistribution).
//
// The per-supplier head-split B/N (where B is the application's per-session budget and N
// the actual number of claiming suppliers) is a GUARANTEED FLOOR, not a hard ceiling:
//   - Serving at or below the floor is always paid in full.
//   - Serving above the floor may additionally be paid from the budget that idle/light
//     suppliers left unused, in proportion to this claim's excess over the floor, bounded
//     by the application's total committed budget B and by overservicing_bonus_multiplier.
//
// The floor and the unused/excess aggregates are precomputed once per (app, session) group
// in Phase 1.5 (see settlementContext.getOrInitSessionBudget); this function only reads
// them, computes THIS claim's bonus, and applies the multiplier cap.
//
// With overservicing_bonus_multiplier == 1 this reproduces the legacy head-split cap
// (AppStake / numPendingSessions / actualNumSuppliers) exactly.
// Ref: https://arxiv.org/pdf/2305.10672
func (k Keeper) ensureClaimAmountLimits(
	ctx context.Context,
	logger log.Logger,
	tokenomicsParams *tokenomicstypes.Params,
	application *apptypes.Application,
	supplier *sharedtypes.Supplier,
	claimSettlementCoin cosmostypes.Coin,
	sessionBudget *sessionBudget,
	globalInflationPerClaimRat *big.Rat,
	serviceId string,
	sessionEndBlockHeight int64,
) (
	actualSettlementCoins cosmostypes.Coin,
	err error,
) {
	logger = logger.With("helper", "ensureClaimAmountLimits")

	// minRequiredAppStakeAmt is the claim expressed in "stake terms": the settlement amount
	// plus its per-claim global-inflation reimbursement. This is the same unit the floor and
	// the redistribution aggregates are computed in.
	// Note that this also incorporates MintPerClaimGlobalInflation since applications are
	// being overcharged by that amount and the funds are sent to the DAO/PNF before being
	// reimbursed to the application in the future.
	//
	// globalInflationPerClaimRat is converted once per settlement block by the caller
	// (settlementContext.getGlobalInflationPerClaimRat) rather than per claim here.
	minRequiredAppStakeAmt := claimStakeTermsAmount(claimSettlementCoin, globalInflationPerClaimRat)
	totalClaimedCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, minRequiredAppStakeAmt)

	// The guaranteed floor (B / N) and the spend-limit flag were precomputed in Phase 1.5.
	floor := sessionBudget.floor
	spendLimitExceeded := sessionBudget.spendLimitExceeded

	// Demote the floor from a hard ceiling to a guaranteed minimum: redistribute the budget
	// left unused by suppliers that claimed below their floor to those that claimed above it,
	// proportional to each overservicer's excess over the floor.
	//   bonus_i = unused * excess_i / totalExcess     (floor division ⇒ Σ bonus ≤ unused)
	// Since Σ floor + Σ bonus ≤ N*floor = B, the total settled across the group never exceeds
	// the application's committed budget, so its stake cannot go negative.
	maxClaimableAmt := floor
	claimExcess := minRequiredAppStakeAmt.Sub(floor)
	if sessionBudget.totalExcess.IsPositive() && claimExcess.IsPositive() {
		bonus := sessionBudget.unused.Mul(claimExcess).Quo(sessionBudget.totalExcess)
		maxClaimableAmt = floor.Add(bonus)
	}

	// overservicing_bonus_multiplier bounds the settlement to m * floor.
	//   m == 0 or 1 => cap at the floor exactly (legacy head-split behaviour; no redistribution).
	//   m >  1      => allow up to m * floor from the unused budget.
	// A ZERO value is deliberately treated as 1 (the legacy cap), NOT as "unlimited": the
	// param's zero-value — whether from a fresh proto3 decode, an un-run upgrade handler, or a
	// clobbered write — must be BENIGN and never silently enable redistribution. To lift the
	// cap in practice, governance sets m high enough that the application's committed budget B
	// (= numSuppliers * floor) binds first (any m >= numSuppliers is equivalent to "only B").
	effectiveMultiplier := tokenomicsParams.OverservicingBonusMultiplier
	if effectiveMultiplier < 1 {
		effectiveMultiplier = 1
	}
	multiplierCap := floor.Mul(math.NewIntFromUint64(effectiveMultiplier))
	if maxClaimableAmt.GT(multiplierCap) {
		maxClaimableAmt = multiplierCap
	}

	maxClaimSettlementAmt := supplierAppStakeToMaxSettlementAmount(maxClaimableAmt, globalInflationPerClaimRat)

	// Check if the claimable amount is capped by the max claimable amount.
	// As per the Relay Mining paper, the Supplier claim MUST NOT exceed the application's
	// allocated stake. If it does, the claim is capped by the application's allocated stake
	// and the supplier is effectively "overserviced".
	if minRequiredAppStakeAmt.GT(maxClaimableAmt) {
		logger.Warn(fmt.Sprintf("claim by supplier %s EXCEEDS LIMITS for application %s. Max claimable amount < claim amount: %v < %v",
			supplier.GetOperatorAddress(), application.GetAddress(), maxClaimableAmt, claimSettlementCoin.Amount))

		minRequiredAppStakeAmt = maxClaimableAmt
		maxClaimSettlementAmt = supplierAppStakeToMaxSettlementAmount(minRequiredAppStakeAmt, globalInflationPerClaimRat)
	}

	// Nominal case: The claimable amount is within the limits set by Relay Mining.
	if claimSettlementCoin.Amount.LTE(maxClaimSettlementAmt) {
		logger.Info(fmt.Sprintf("claim by supplier %s IS WITHIN LIMITS of servicing application %s. Max claimable amount >= claim amount: %v >= %v",
			supplier.GetOperatorAddress(), application.GetAddress(), maxClaimSettlementAmt, claimSettlementCoin.Amount))
		return claimSettlementCoin, nil
	}

	// Claimable amount is capped by the max claimable amount or the application allocated stake.
	// Determine the max claimable amount for the supplier based on the application's stake in this session.
	maxClaimableCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, maxClaimSettlementAmt)

	// Prepare and emit the event for the application being overserviced.
	// Both ExpectedBurn and EffectiveBurn include the globalInflation component
	// so they are on the same basis (total tokens burnt from app stake).
	applicationOverservicedEvent := &tokenomicstypes.EventApplicationOverserviced{
		ApplicationAddr:       application.GetAddress(),
		SupplierOperatorAddr:  supplier.GetOperatorAddress(),
		ExpectedBurn:          totalClaimedCoin.String(),
		EffectiveBurn:         cosmostypes.NewCoin(pocket.DenomuPOKT, minRequiredAppStakeAmt).String(),
		ServiceId:             serviceId,
		SessionEndBlockHeight: sessionEndBlockHeight,
		SpendLimitExceeded:    spendLimitExceeded,
	}
	eventManager := cosmostypes.UnwrapSDKContext(ctx).EventManager()
	if err = eventManager.EmitTypedEvent(applicationOverservicedEvent); err != nil {
		return cosmostypes.Coin{},
			tokenomicstypes.ErrTokenomicsEmittingEventFailed.Wrapf("error emitting event %v", applicationOverservicedEvent)
	}

	return maxClaimableCoin, nil
}

// claimStakeTermsAmount converts a claim's settlement amount into "stake terms" — the
// amount that must be available in the application's stake to honor the claim, i.e. the
// settlement amount plus its per-claim global-inflation reimbursement.
//
// This is the unit in which the per-session budget, floor, and redistribution are all
// computed, so the Phase 1.5 budget accounting (settlement_context.go) and the Phase 2
// per-claim cap (ensureClaimAmountLimits) MUST derive it identically — hence the shared
// helper.
//
// Takes the already-converted big.Rat rather than the float64 param: the conversion is
// identical for every claim in a block, so it is done once per settlement block by
// settlementContext.getGlobalInflationPerClaimRat. The Rat is read-only here.
func claimStakeTermsAmount(claimSettlementCoin cosmostypes.Coin, globalInflationPerClaimRat *big.Rat) math.Int {
	globalInflationCoin := tlm.CalculateGlobalPerClaimMintInflationFromSettlementAmount(claimSettlementCoin, globalInflationPerClaimRat)
	return claimSettlementCoin.Amount.Add(globalInflationCoin.Amount)
}

// supplierAppStakeToMaxSettlementAmount calculates the max amount of uPOKT the supplier
// can claim based on the stake allocated to the supplier and the global inflation
// allocation percentage.
// This is the inverse of CalculateGlobalPerClaimMintInflationFromSettlementAmount:
// stake = maxSettlementAmt + globalInflationAmt
// stake = maxSettlementAmt + (maxSettlementAmt * GlobalInflationPerClaim)
// stake = maxSettlementAmt * (1 + GlobalInflationPerClaim)
// maxSettlementAmt = stake / (1 + GlobalInflationPerClaim)
//
// Takes the already-converted big.Rat (see claimStakeTermsAmount) so the per-block
// float64->Rat conversion is not repeated for every claim. The Rat is read-only here.
func supplierAppStakeToMaxSettlementAmount(stakeAmount math.Int, globalInflationPerClaimRat *big.Rat) math.Int {
	// divisor = 1 + globalInflationPerClaim
	divisor := new(big.Rat).Add(new(big.Rat).SetInt64(1), globalInflationPerClaimRat)
	stakeRat := new(big.Rat).SetInt(stakeAmount.BigInt())
	resultRat := new(big.Rat).Quo(stakeRat, divisor)
	// Truncate (floor) to get the integer settlement amount.
	resultInt := new(big.Int).Quo(resultRat.Num(), resultRat.Denom())
	return math.NewIntFromBigInt(resultInt)
}
