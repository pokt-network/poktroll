package keeper

import (
	"context"
	"fmt"

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
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
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
	pendingResult *tlm.PendingSettlementResult,
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
		relayMiningDifficulty = servicekeeper.NewDefaultRelayMiningDifficulty(ctx, logger, service.Id, servicekeeper.TargetNumRelays)
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
	actualSettlementCoin, err := k.ensureClaimAmountLimits(ctx, logger, &application, &supplier, claimSettlementCoin)
	if err != nil {
		return err
	}
	logger = logger.With("actual_settlement_upokt", actualSettlementCoin)
	logger.Info(fmt.Sprintf("About to start processing TLMs for (%d) compute units, equal to (%s) claimed", numClaimComputeUnits, actualSettlementCoin))

	// Execute all the token logic modules processors
	for _, tlmProcessor := range k.tokenLogicModuleProcessors {
		tlmName := tlmProcessor.GetTLM().String()
		logger.Info(fmt.Sprintf("Starting TLM processing: %q", tlmName))

		if err = tlmProcessor.Process(
			ctx, logger,
			pendingResult,
			&service,
			pendingResult.Claim.GetSessionHeader(),
			&application,
			&supplier,
			actualSettlementCoin,
			&relayMiningDifficulty,
		); err != nil {
			return tokenomicstypes.ErrTokenomicsTLMError.Wrapf("TLM %q: %s", tlmName, err)
		}

		logger.Info(fmt.Sprintf("Finished TLM processing: %q", tlmName))
	}

	// TODO_CONSIDERATION: If we support multiple native tokens, we will need to
	// start checking the denom here.
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, cosmostypes.UnwrapSDKContext(ctx).BlockHeight())
	if application.Stake.Amount.LT(apptypes.DefaultMinStake.Amount) {
		// Mark the application as unbonding if it has less than the minimum stake.
		application.UnstakeSessionEndHeight = uint64(sessionEndHeight)
		unbondingEndHeight := apptypes.GetApplicationUnbondingHeight(&sharedParams, &application)

		appUnbondingBeginEvent := &apptypes.EventApplicationUnbondingBegin{
			Application:        &application,
			Reason:             apptypes.ApplicationUnbondingReason_BELOW_MIN_STAKE,
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

	// TODO_MAINNET: If the application stake has dropped to (near?) zero, should
	// we unstake it? Should we use it's balance? Should there be a payee of last resort?
	// Make sure to document whatever decision we come to.

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
	application *apptypes.Application,
	supplier *sharedtypes.Supplier,
	claimSettlementCoin cosmostypes.Coin,
) (
	actualSettlementCoins cosmostypes.Coin,
	err error,
) {
	logger = logger.With("helper", "ensureClaimAmountLimits")

	// TODO_BETA_OR_MAINNET(@red-0ne): The application stake gets reduced with every claim
	// settlement. Relay miners use the appStake at the beginning of a session to determine
	// the maximum amount they can claim. We need to somehow access and propagate this
	// value (via context?) so it is the same for all TLM processors for each claim.
	// Note that this will also need to incorporate MintPerClaimGlobalInflation because
	// applications are being overcharged by that amount in the meantime. Whatever the
	// solution and implementation ends up being, make sure to KISS.
	appStake := application.GetStake()

	// Determine the max claimable amount for the supplier based on the application's stake in this session.
	maxClaimableCoin := sdk.NewCoin(volatile.DenomuPOKT, appStake.Amount.Quo(math.NewInt(sessionkeeper.NumSupplierPerSession)))

	if maxClaimableCoin.Amount.GTE(claimSettlementCoin.Amount) {
		logger.Info(fmt.Sprintf("Claim by supplier %s IS WITHIN LIMITS of servicing application %s. Max claimable amount >= Claim amount: %v >= %v",
			supplier.GetOperatorAddress(), application.GetAddress(), maxClaimableCoin, claimSettlementCoin.Amount))
		return claimSettlementCoin, nil
	}

	logger.Warn(fmt.Sprintf("Claim by supplier %s EXCEEDS LIMITS for application %s. Max claimable amount < Claim amount: %v < %v",
		supplier.GetOperatorAddress(), application.GetAddress(), maxClaimableCoin, claimSettlementCoin.Amount))

	// Reduce the settlement amount if the application was over-serviced
	actualSettlementCoins = maxClaimableCoin

	// Prepare and emit the event for the application being overserviced
	applicationOverservicedEvent := &tokenomicstypes.EventApplicationOverserviced{
		ApplicationAddr:      application.Address,
		SupplierOperatorAddr: supplier.GetOperatorAddress(),
		ExpectedBurn:         &claimSettlementCoin,
		EffectiveBurn:        &maxClaimableCoin,
	}
	eventManager := cosmostypes.UnwrapSDKContext(ctx).EventManager()
	if err := eventManager.EmitTypedEvent(applicationOverservicedEvent); err != nil {
		return cosmostypes.Coin{},
			tokenomicstypes.ErrTokenomicsEmittingEventFailed.Wrapf("error emitting event %v", applicationOverservicedEvent)
	}

	return actualSettlementCoins, nil
}
