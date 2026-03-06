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
	"github.com/pokt-network/poktroll/pkg/encoding"
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

	sharedParams := settlementContext.GetSharedParams()
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

	applicationInitialStake, err := settlementContext.GetApplicationInitialStake(sessionHeader.ApplicationAddress)
	if err != nil {
		return cosmostypes.Coin{}, err
	}

	// Ensure the number of compute units claimed is equal to the number of relays * CUPR
	expectedClaimComputeUnits := numRelays * service.ComputeUnitsPerRelay
	if numClaimComputeUnits != expectedClaimComputeUnits {
		return cosmostypes.Coin{}, tokenomicstypes.ErrTokenomicsClaimRootHashInvalid.Wrapf(
			"mismatch: claim compute units (%d) != number of relays (%d) * service compute units per relay (%d)",
			numClaimComputeUnits,
			numRelays,
			service.ComputeUnitsPerRelay,
		)
	}

	// Determine the total number of tokens being claimed (i.e. for the work completed)
	// by the supplier for the amount of work they did to service the application
	// in the session.
	claimSettlementCoin, err = pendingResult.Claim.GetClaimeduPOKT(sharedParams, relayMiningDifficulty)
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

	// Get the actual number of suppliers that submitted claims for this (app, session) pair.
	// This replaces the NumSuppliersPerSession governance param to ensure fair budget
	// distribution based on actual participation rather than the theoretical maximum.
	actualNumSuppliers := settlementContext.GetActualSupplierCount(
		sessionHeader.ApplicationAddress,
		sessionHeader.SessionId,
	)

	// Ensure the claim amount is within the limits set by RelayMining.
	// If not, update the settlement amount and emit relevant events.
	// TODO_IMPROVE: Consider pulling this out of Keeper#ProcessTokenLogicModules
	// and ensure claim amount limits are enforced before TLM processing.
	actualSettlementCoin, err := k.ensureClaimAmountLimits(ctx, logger, &sharedParams, &tokenomicsParams, application, supplier, claimSettlementCoin, applicationInitialStake, actualNumSuppliers, sessionHeader.ServiceId, sessionHeader.SessionEndBlockHeight)
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
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, cosmostypes.UnwrapSDKContext(ctx).BlockHeight())
	if tlmCtx.Application.Stake.Amount.LT(apptypes.DefaultMinStake.Amount) {
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

// ensureClaimAmountLimits checks and handles overserviced applications.
//
// Per Algorithm #1 in the Relay Mining paper, the maximum amount that a single
// supplier can claim in a session is AppStake/ActualNumSuppliers, where
// ActualNumSuppliers is the number of suppliers that submitted claims for
// this (app, session) pair. This replaces the previous NumSuppliersPerSession
// governance param to ensure fair budget distribution based on actual participation.
// Ref: https://arxiv.org/pdf/2305.10672
//
// If this is not the case, then the supplier essentially did "free work" and the
// actual claim amount is less than what was claimed.
func (k Keeper) ensureClaimAmountLimits(
	ctx context.Context,
	logger log.Logger,
	sharedParams *sharedtypes.Params,
	tokenomicsParams *tokenomicstypes.Params,
	application *apptypes.Application,
	supplier *sharedtypes.Supplier,
	claimSettlementCoin cosmostypes.Coin,
	initialApplicationStake cosmostypes.Coin,
	actualNumSuppliers int64,
	serviceId string,
	sessionEndBlockHeight int64,
) (
	actualSettlementCoins cosmostypes.Coin,
	err error,
) {
	logger = logger.With("helper", "ensureClaimAmountLimits")

	// Note that this also incorporates MintP	erClaimGlobalInflation since applications
	// are being overcharged by that amount and the funds are sent to the DAO/PNF
	// before being reimbursed to the application in the future.
	appStake := initialApplicationStake

	// The application should have enough stake to cover for the global mint reimbursement.
	// This amount is deducted from the maximum claimable amount.
	globalInflationPerClaim := tokenomicsParams.GlobalInflationPerClaim
	globalInflationPerClaimRat, err := encoding.Float64ToRat(globalInflationPerClaim)
	if err != nil {
		logger.Error(fmt.Sprintf("error calculating claim amount limits due to: %v", err))
		return actualSettlementCoins, err
	}

	globalInflationCoin := tlm.CalculateGlobalPerClaimMintInflationFromSettlementAmount(claimSettlementCoin, globalInflationPerClaimRat)
	globalInflationAmt := globalInflationCoin.Amount
	minRequiredAppStakeAmt := claimSettlementCoin.Amount.Add(globalInflationAmt)
	totalClaimedCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, minRequiredAppStakeAmt)

	// get the number of pending sessions that share the application stake at claim time
	// This is used to calculate the maximum claimable amount for the supplier within a session.
	numPendingSessions := sharedtypes.GetNumPendingSessions(sharedParams)

	// The maximum any single supplier can claim is a fraction of the app's total stake
	// divided by the actual number of suppliers that submitted claims for this session.
	// Using actual count instead of the NumSuppliersPerSession governance param ensures
	// fair budget distribution: if only 20 of 50 assigned suppliers claim, each gets
	// appStake/20 instead of appStake/50, eliminating wasted budget from no-shows.
	// Divide sessions first, then suppliers — matches the spend-limit path's
	// conceptual order and avoids different integer truncation results.
	maxClaimableAmt := appStake.Amount.
		Quo(math.NewInt(numPendingSessions)).
		Quo(math.NewInt(actualNumSuppliers))

	// Apply per-session spend limit if set on the application.
	// The spend limit caps the per-session budget, which is then divided among suppliers.
	spendLimitExceeded := false
	if application.PerSessionSpendLimit != nil &&
		application.PerSessionSpendLimit.Amount.GT(math.ZeroInt()) {
		// Validate the spend limit denom matches the native token.
		if application.PerSessionSpendLimit.Denom != pocket.DenomuPOKT {
			return cosmostypes.Coin{}, tokenomicstypes.ErrTokenomicsApplicationNewStakeInvalid.Wrapf(
				"application %s has per_session_spend_limit with invalid denom %q (expected %q)",
				application.GetAddress(), application.PerSessionSpendLimit.Denom, pocket.DenomuPOKT,
			)
		}
		// Per-session budget is min(spend_limit, appStake/numPendingSessions)
		perSessionBudget := application.PerSessionSpendLimit.Amount
		stakePerSession := appStake.Amount.Quo(math.NewInt(numPendingSessions))
		if perSessionBudget.GT(stakePerSession) {
			logger.Warn(fmt.Sprintf(
				"application %s per_session_spend_limit %s exceeds stake-based budget %s per session; clamping to stake-based budget",
				application.GetAddress(), perSessionBudget, stakePerSession,
			))
			perSessionBudget = stakePerSession
		}
		// Per-supplier cap from the spend limit
		spendLimitMaxClaimable := perSessionBudget.Quo(math.NewInt(actualNumSuppliers))
		if spendLimitMaxClaimable.LT(maxClaimableAmt) {
			maxClaimableAmt = spendLimitMaxClaimable
			spendLimitExceeded = true
		}
	}

	maxClaimSettlementAmt := supplierAppStakeToMaxSettlementAmount(maxClaimableAmt, globalInflationPerClaim)

	// Check if the claimable amount is capped by the max claimable amount.
	// As per the Relay Mining paper, the Supplier claim MUST NOT exceed the application's
	// allocated stake. If it does, the claim is capped by the application's allocated stake
	// and the supplier is effectively "overserviced".
	if minRequiredAppStakeAmt.GT(maxClaimableAmt) {
		logger.Warn(fmt.Sprintf("claim by supplier %s EXCEEDS LIMITS for application %s. Max claimable amount < claim amount: %v < %v",
			supplier.GetOperatorAddress(), application.GetAddress(), maxClaimableAmt, claimSettlementCoin.Amount))

		minRequiredAppStakeAmt = maxClaimableAmt
		maxClaimSettlementAmt = supplierAppStakeToMaxSettlementAmount(minRequiredAppStakeAmt, globalInflationPerClaim)
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

// supplierAppStakeToMaxSettlementAmount calculates the max amount of uPOKT the supplier
// can claim based on the stake allocated to the supplier and the global inflation
// allocation percentage.
// This is the inverse of CalculateGlobalPerClaimMintInflationFromSettlementAmount:
// stake = maxSettlementAmt + globalInflationAmt
// stake = maxSettlementAmt + (maxSettlementAmt * GlobalInflationPerClaim)
// stake = maxSettlementAmt * (1 + GlobalInflationPerClaim)
// maxSettlementAmt = stake / (1 + GlobalInflationPerClaim)
func supplierAppStakeToMaxSettlementAmount(stakeAmount math.Int, globalInflationPerClaim float64) math.Int {
	stakeAmountFloat := big.NewFloat(0).SetInt(stakeAmount.BigInt())
	maxSettlementAmountFloat := big.NewFloat(0).Quo(stakeAmountFloat, big.NewFloat(1+globalInflationPerClaim))

	settlementAmount, _ := maxSettlementAmountFloat.Int(nil)
	return math.NewIntFromBigInt(settlementAmount)
}
