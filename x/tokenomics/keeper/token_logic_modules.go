package keeper

import (
	"context"
	"fmt"
	"math/big"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/telemetry"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	servicekeeper "github.com/pokt-network/poktroll/x/service/keeper"
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
	pendingResult *tokenomicstypes.ClaimSettlementResult,
	applicationInitialStake cosmostypes.Coin,
) error {
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
		return tokenomicstypes.ErrTokenomicsSessionHeaderNil
	}
	if err := sessionHeader.ValidateBasic(); err != nil {
		logger.Error("received an invalid session header", "error", err)
		return tokenomicstypes.ErrTokenomicsSessionHeaderInvalid
	}

	// Retrieve and validate the root of the claim to determine the amount of work done
	root := (smt.MerkleSumRoot)(pendingResult.Claim.GetRootHash())
	if !root.HasDigestSize(protocol.TrieHasherSize) {
		return tokenomicstypes.ErrTokenomicsRootHashInvalid.Wrapf(
			"root hash has invalid digest size (%d), expected (%d)",
			root.DigestSize(), protocol.TrieHasherSize,
		)
	}

	// Retrieve the sum (i.e. number of compute units) to determine the amount of work done
	numClaimComputeUnits, err := pendingResult.Claim.GetNumClaimedComputeUnits()
	if err != nil {
		return tokenomicstypes.ErrTokenomicsRootHashInvalid.Wrapf("failed to retrieve numClaimComputeUnits: %s", err)
	}
	// TODO_MAINNET(@bryanchriswhite, @red-0ne): Fix the low-volume exploit here.
	// https://www.notion.so/buildwithgrove/RelayMiningDifficulty-and-low-volume-7aab3edf6f324786933af369c2fa5f01?pvs=4
	if numClaimComputeUnits == 0 {
		return tokenomicstypes.ErrTokenomicsRootHashInvalid.Wrap("root hash has zero relays")
	}

	numRelays, err := pendingResult.Claim.GetNumRelays()
	if err != nil {
		return tokenomicstypes.ErrTokenomicsRootHashInvalid.Wrapf("failed to retrieve numRelays: %s", err)
	}

	/*
		TODO_POST_MAINNET: Because of how things have evolved, we are now using
		root.Count (numRelays) instead of root.Sum (numComputeUnits) to determine
		the amount of work done. This is because the compute_units_per_relay is
		a service specific (not request specific) parameter that will be maintained
		by the service owner to capture the average amount of resources (i.e.
		compute, storage, bandwidth, electricity, etc...) per request. Modifying
		this on a per request basis has been deemed too complex and not a mainnet
		blocker.
	*/

	// Retrieve the application address that is being charged; getting services and paying tokens.
	applicationAddress, err := cosmostypes.AccAddressFromBech32(sessionHeader.GetApplicationAddress())
	if err != nil || applicationAddress == nil {
		return tokenomicstypes.ErrTokenomicsApplicationAddressInvalid.Wrapf("address (%q)", sessionHeader.GetApplicationAddress())
	}

	// Retrieve the on-chain staked application record
	application, isAppFound := k.applicationKeeper.GetApplication(ctx, applicationAddress.String())
	if !isAppFound {
		logger.Warn(fmt.Sprintf("application for claim with address %q not found", applicationAddress))
		return tokenomicstypes.ErrTokenomicsApplicationNotFound
	}

	// Retrieve the supplier operator address that will be getting rewarded; providing services and earning tokens
	supplierOperatorAddr, err := cosmostypes.AccAddressFromBech32(pendingResult.Claim.GetSupplierOperatorAddress())
	if err != nil || supplierOperatorAddr == nil {
		return tokenomicstypes.ErrTokenomicsSupplierOperatorAddressInvalid.Wrapf(
			"address (%q)", pendingResult.Claim.GetSupplierOperatorAddress(),
		)
	}

	// Retrieve the on-chain staked supplier record
	supplier, isSupplierFound := k.supplierKeeper.GetSupplier(ctx, supplierOperatorAddr.String())
	if !isSupplierFound {
		logger.Warn(fmt.Sprintf("supplier for claim with address %q not found", supplierOperatorAddr))
		return tokenomicstypes.ErrTokenomicsSupplierNotFound
	}

	service, isServiceFound := k.serviceKeeper.GetService(ctx, sessionHeader.ServiceId)
	if !isServiceFound {
		return tokenomicstypes.ErrTokenomicsServiceNotFound.Wrapf("service with ID %q not found", sessionHeader.ServiceId)
	}

	// Ensure the number of compute units claimed is equal to the number of relays * CUPR
	expectedClaimComputeUnits := numRelays * service.ComputeUnitsPerRelay
	if numClaimComputeUnits != expectedClaimComputeUnits {
		return tokenomicstypes.ErrTokenomicsRootHashInvalid.Wrapf(
			"mismatch: claim compute units (%d) != number of relays (%d) * service compute units per relay (%d)",
			numClaimComputeUnits,
			numRelays,
			service.ComputeUnitsPerRelay,
		)
	}

	// Retrieving the relay mining difficulty for service.
	relayMiningDifficulty, found := k.serviceKeeper.GetRelayMiningDifficulty(ctx, service.Id)
	if !found {
		targetNumRelays := k.serviceKeeper.GetParams(ctx).TargetNumRelays
		relayMiningDifficulty = servicekeeper.NewDefaultRelayMiningDifficulty(
			ctx,
			logger,
			service.Id,
			targetNumRelays,
			targetNumRelays,
		)
	}
	sharedParams := k.sharedKeeper.GetParams(ctx)

	// Determine the total number of tokens being claimed (i.e. for the work completed)
	// by the supplier for the amount of work they did to service the application
	// in the session.
	claimSettlementCoin, err = pendingResult.Claim.GetClaimeduPOKT(sharedParams, relayMiningDifficulty)
	if err != nil {
		return err
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

	// Ensure the claim amount is within the limits set by Relay Mining.
	// If not, update the settlement amount and emit relevant events.
	// TODO_MAINNET(@red-0ne): Consider pulling this out of Keeper#ProcessTokenLogicModules
	// and ensure claim amount limits are enforced before TLM processing.
	actualSettlementCoin, err := k.ensureClaimAmountLimits(ctx, logger, &sharedParams, &application, &supplier, claimSettlementCoin, applicationInitialStake)
	if err != nil {
		return err
	}
	logger = logger.With("actual_settlement_upokt", actualSettlementCoin)
	logger.Info(fmt.Sprintf("About to start processing TLMs for (%d) compute units, equal to (%s) claimed", numClaimComputeUnits, actualSettlementCoin))

	// TODO_MAINNET(@red-0ne): Add tests to ensure that a zero settlement coin
	// due to integer division rounding is handled correctly.
	if actualSettlementCoin.Amount.IsZero() {
		logger.Warn(fmt.Sprintf(
			"actual settlement coin is zero, skipping TLM processing, application %q stake %s",
			application.Address, application.Stake,
		))
		return nil
	}

	tlmCtx := tlm.TLMContext{
		TokenomicsParams:      k.GetParams(ctx),
		SettlementCoin:        actualSettlementCoin,
		SessionHeader:         pendingResult.Claim.GetSessionHeader(),
		Result:                pendingResult,
		Service:               &service,
		Application:           &application,
		Supplier:              &supplier,
		RelayMiningDifficulty: &relayMiningDifficulty,
	}

	// Execute all the token logic modules processors
	for _, tokenLogicModule := range k.tokenLogicModules {
		tlmName := tokenLogicModule.GetId().String()
		logger.Info(fmt.Sprintf("Starting processing TLM: %q", tlmName))

		if err = tokenLogicModule.Process(ctx, logger, tlmCtx); err != nil {
			return tokenomicstypes.ErrTokenomicsProcessingTLM.Wrapf("TLM %q: %s", tlmName, err)
		}

		logger.Info(fmt.Sprintf("Finished processing TLM: %q", tlmName))
	}

	// TODO_POST_MAINNET: If we support multiple native tokens, we will need to start checking the denom here.
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, cosmostypes.UnwrapSDKContext(ctx).BlockHeight())
	if application.Stake.Amount.LT(apptypes.DefaultMinStake.Amount) {
		// Mark the application as unbonding if it has less than the minimum stake.
		application.UnstakeSessionEndHeight = uint64(sessionEndHeight)
		unbondingEndHeight := apptypes.GetApplicationUnbondingHeight(&sharedParams, &application)

		appUnbondingBeginEvent := &apptypes.EventApplicationUnbondingBegin{
			Application:        &application,
			Reason:             apptypes.ApplicationUnbondingReason_APPLICATION_UNBONDING_REASON_BELOW_MIN_STAKE,
			SessionEndHeight:   sessionEndHeight,
			UnbondingEndHeight: unbondingEndHeight,
		}

		sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
		if err = sdkCtx.EventManager().EmitTypedEvent(appUnbondingBeginEvent); err != nil {
			err = apptypes.ErrAppEmitEvent.Wrapf("(%+v): %s", appUnbondingBeginEvent, err)
			logger.Error(err.Error())
			return err
		}
	}

	// State mutation: update the application's on-chain record.
	k.applicationKeeper.SetApplication(ctx, application)
	logger.Info(fmt.Sprintf("updated on-chain application record with address %q", application.Address))

	// TODO_MAINNET(@bryanchriswhite): If the application stake has dropped to (near?) zero:
	// - Unstake it
	// - Emit an event
	// - Ensure this doesn't happen
	// - Document the decision

	// State mutation: Update the suppliers's on-chain record
	k.supplierKeeper.SetSupplier(ctx, supplier)
	logger.Info(fmt.Sprintf("updated on-chain supplier record with address %q", supplier.OperatorAddress))

	// Update isSuccessful to true for telemetry
	isSuccessful = true
	return nil
}

// ensureClaimAmountLimits checks if the application was overserviced and handles
// the case if it was.
// Per Algorithm #1 in the Relay Mining paper, the maximum amount that a single supplier
// can claim in a session is AppStake/NumSuppliersPerSession.
// If this is not the case, then the supplier essentially did "free work" and the
// actual claim amount is less than what was claimed.
// Ref: https://arxiv.org/pdf/2305.10672
func (k Keeper) ensureClaimAmountLimits(
	ctx context.Context,
	logger log.Logger,
	sharedParams *sharedtypes.Params,
	application *apptypes.Application,
	supplier *sharedtypes.Supplier,
	claimSettlementCoin cosmostypes.Coin,
	initialApplicationStake cosmostypes.Coin,
) (
	actualSettlementCoins cosmostypes.Coin,
	err error,
) {
	logger = logger.With("helper", "ensureClaimAmountLimits")

	// Note that this also incorporates MintPerClaimGlobalInflation since applications
	// are being overcharged by that amount and the funds are sent to the DAO/PNF
	// before being reimbursed to the application in the future.
	appStake := initialApplicationStake

	// The application should have enough stake to cover for the global mint reimbursement.
	// This amount is deducted from the maximum claimable amount.
	globalInflationCoin, _ := tlm.CalculateGlobalPerClaimMintInflationFromSettlementAmount(claimSettlementCoin)
	globalInflationAmt := globalInflationCoin.Amount
	minRequiredAppStakeAmt := claimSettlementCoin.Amount.Add(globalInflationAmt)
	totalClaimedCoin := sdk.NewCoin(volatile.DenomuPOKT, minRequiredAppStakeAmt)

	// get the number of pending sessions that share the application stake at claim time
	// This is used to calculate the maximum claimable amount for the supplier within a session.
	numPendingSessions := sharedtypes.GetNumPendingSessions(sharedParams)

	// The maximum any single supplier can claim is a fraction of the app's total stake
	// divided by the number of suppliers per session.
	// Re decentralization - This ensures the app biases towards using all suppliers in a session.
	// Re costs - This is an easy way to split the stake evenly.
	// TODO_FUTURE: See if there's a way to let the application prefer (the best)
	// supplier(s) in a session while maintaining a simple solution to implement this.
	numSuppliersPerSession := int64(k.sessionKeeper.GetParams(ctx).NumSuppliersPerSession)
	maxClaimableAmt := appStake.Amount.
		Quo(math.NewInt(numSuppliersPerSession)).
		Quo(math.NewInt(numPendingSessions))
	maxClaimSettlementAmt := supplierAppStakeToMaxSettlementAmount(maxClaimableAmt)

	// Check if the claimable amount is capped by the max claimable amount.
	// As per the Relay Mining paper, the Supplier claim MUST NOT exceed the application's
	// allocated stake. If it does, the claim is capped by the application's allocated stake
	// and the supplier is effectively "overserviced".
	if minRequiredAppStakeAmt.GT(maxClaimableAmt) {
		logger.Warn(fmt.Sprintf("claim by supplier %s EXCEEDS LIMITS for application %s. Max claimable amount < claim amount: %v < %v",
			supplier.GetOperatorAddress(), application.GetAddress(), maxClaimableAmt, claimSettlementCoin.Amount))

		minRequiredAppStakeAmt = maxClaimableAmt
		maxClaimSettlementAmt = supplierAppStakeToMaxSettlementAmount(minRequiredAppStakeAmt)
	}

	// Nominal case: The claimable amount is within the limits set by Relay Mining.
	if claimSettlementCoin.Amount.LTE(maxClaimSettlementAmt) {
		logger.Info(fmt.Sprintf("claim by supplier %s IS WITHIN LIMITS of servicing application %s. Max claimable amount >= claim amount: %v >= %v",
			supplier.GetOperatorAddress(), application.GetAddress(), maxClaimSettlementAmt, claimSettlementCoin.Amount))
		return claimSettlementCoin, nil
	}

	// Claimable amount is capped by the max claimable amount or the application allocated stake.
	// Determine the max claimable amount for the supplier based on the application's stake in this session.
	maxClaimableCoin := sdk.NewCoin(volatile.DenomuPOKT, maxClaimSettlementAmt)

	// Prepare and emit the event for the application being overserviced
	applicationOverservicedEvent := &tokenomicstypes.EventApplicationOverserviced{
		ApplicationAddr:      application.GetAddress(),
		SupplierOperatorAddr: supplier.GetOperatorAddress(),
		ExpectedBurn:         &totalClaimedCoin,
		EffectiveBurn:        &maxClaimableCoin,
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
func supplierAppStakeToMaxSettlementAmount(stakeAmount math.Int) math.Int {
	stakeAmountFloat := big.NewFloat(0).SetInt(stakeAmount.BigInt())
	maxSettlementAmountFloat := big.NewFloat(0).Quo(stakeAmountFloat, big.NewFloat(1+tlm.GlobalInflationPerClaim))

	settlementAmount, _ := maxSettlementAmountFloat.Int(nil)
	return math.NewIntFromBigInt(settlementAmount)
}
