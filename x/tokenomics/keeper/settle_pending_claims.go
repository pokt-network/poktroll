package keeper

import (
	"fmt"

	cosmoslog "cosmossdk.io/log"
	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/telemetry"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tlm "github.com/pokt-network/poktroll/x/tokenomics/token_logic_module"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// SettlePendingClaims settles all pending (i.e. expiring) claims.
// - Settling a claim: If a claim is expired AND requires a proof AND a proof IS available.
// - Settling a claim: If a claim is expired AND does NOT require a proof.
// - Deleting a claim: If a claim is expired AND requires a proof AND a proof IS NOT available.
// Events are emitted for each claim that is settled or removed.
// Onchain Claims & Proofs are deleted after they're settled or expired to free up space.
func (k Keeper) SettlePendingClaims(ctx cosmostypes.Context) (
	settledResults tlm.ClaimSettlementResults,
	expiredResults tlm.ClaimSettlementResults,
	numDiscardedFaultyClaims uint64,
	err error,
) {
	logger := k.Logger().With("method", "SettlePendingClaims")
	logger.Debug("settling expiring claims")

	// Initialize results structs.
	blockHeight := ctx.BlockHeight()
	settledResults = make(tlm.ClaimSettlementResults, 0)
	expiredResults = make(tlm.ClaimSettlementResults, 0)
	settlementContext := NewSettlementContext(ctx, &k, logger)
	numExpiringClaims := 0
	numDiscardedFaultyClaims = 0

	// Retrieve an iterator of all expiring claims.
	// DEV_NOTE: This previously retrieve a list but has been change to account for large claim counts.
	expiringClaimsIterator := k.GetExpiringClaimsIterator(ctx, settlementContext, blockHeight)
	defer expiringClaimsIterator.Close()

	// Iterating over all potentially expiring claims.
	// This loop does the following:
	// 1. Retrieve the claim
	// 2. Settle the claim
	// 3. Remove the claim from the state
	// 4. Emit an event
	// 5. Update the relevant actors in the state
	for ; expiringClaimsIterator.Valid(); expiringClaimsIterator.Next() {
		claim, iterErr := expiringClaimsIterator.Value()
		if iterErr != nil {
			numDiscardedFaultyClaims++
			claimKey := string(expiringClaimsIterator.Key())
			claimErr := tokenomicstypes.ErrTokenomicsSettlementInternal.Wrapf(
				"[UNEXPECTED ERROR] Critical error during claim settlement during session with key %s. Claim will be discarded to prevent chain halt: %s",
				claimKey, iterErr)
			logger.Error(claimErr.Error())
			continue
		}

		// Settle the claim.
		claimProcessingContext, settlementErr := k.settleClaim(ctx, settlementContext, claim, logger)
		if settlementErr != nil {
			// Discard a faulty claim and continue iterating over the next one.
			numDiscardedFaultyClaims++
			err = tokenomicstypes.ErrTokenomicsSettlementInternal.Wrapf(
				"[UNEXPECTED ERROR] Critical error during claim settlement during session %q for operator %s and service %s. Claim will be discarded to prevent chain halt: %s",
				claim.SessionHeader.SessionId, claim.SupplierOperatorAddress, claim.SessionHeader.ServiceId, err,
			)
			logger.Error(err.Error())
			k.discardFaultyClaim(ctx, logger, claim, err.Error())
			continue
		}

		// If the code path reached this section, we will either:
		// - Successfully settle the claim
		// - Successfully expire the claim
		numExpiringClaims++

		// Identify the stage of the claim (e.g. settled, expired, proven, claimed, etc.)
		var settlementStage prooftypes.ClaimProofStage
		if claimProcessingContext.isSettled {
			// Append the token logic module processing ClaimSettlementResult to the settled results.
			settledResults = append(settledResults, claimProcessingContext.settlementResult)
			settlementStage = prooftypes.ClaimProofStage_SETTLED
		} else {
			// Append the settlement result to the expired results.
			expiredResults = append(expiredResults, claimProcessingContext.settlementResult)
			settlementStage = prooftypes.ClaimProofStage_EXPIRED
		}

		// Telemetry - defer telemetry calls so that they reference the final values the relevant variables.
		// Telemetry will be emitted for all non-faulty (i.e. discarded) claims.
		defer k.finalizeClaimTelemetry(
			settlementStage,
			claim.SessionHeader.ServiceId,
			claim.SessionHeader.ApplicationAddress,
			claim.SupplierOperatorAddress,
			claimProcessingContext.numClaimRelays,
			claimProcessingContext.numClaimComputeUnits,
			settlementErr,
		)

		// The claim is being settled or expired, so it no longer needs onchain storage.
		// Remove it from the latest (i.e. pruned) state so it will be maintained by archived nodes.
		k.proofKeeper.RemoveClaim(ctx, claim.SessionHeader.SessionId, claim.SupplierOperatorAddress)

		logger.Info(
			"Finished processing claim",
			"session_id", claim.SessionHeader.SessionId,
			"supplier", claim.SupplierOperatorAddress,
			"settled", claimProcessingContext.isSettled,
			"block_height", blockHeight,
		)
	}

	// When this code path is reached:
	// - All claims HAVE BEEN processed (successfully or not)
	// - Onchain data HAS NOT BEEN updated
	// - Settlement results HAVE NOT BEEN executed

	logger = logger.With(
		"num_expiring_claims", numExpiringClaims,
		"num_settled", settledResults.GetNumClaims(),
		"num_expired", expiredResults.GetNumClaims(),
		"num_discarded_faulty", numDiscardedFaultyClaims,
		"block_height", blockHeight,
	)

	logger.Info("About to start updating onchain data with claim settlement results")

	// Claims settlement results have been computed when this code path is reached.
	// The onchain data now needs to be updated.

	// Persist the state of all the applications and suppliers involved in the claims.
	// This is done in a single batch to reduce the number of writes to state storage.
	settlementContext.FlushAllActorsToStore(ctx)

	logger.Info("Updated onchain data with claim settlement results")

	// Execute all the pending mint, burn, and transfer operations.
	if err = k.ExecutePendingSettledResults(ctx, settledResults); err != nil {
		return settledResults, expiredResults, numDiscardedFaultyClaims, err
	}

	// Slash all suppliers who failed to submit a required proof.
	if err = k.ExecutePendingExpiredResults(ctx, settlementContext, expiredResults); err != nil {
		return settledResults, expiredResults, numDiscardedFaultyClaims, err
	}

	logger.Info(
		"claims settlement summary",
		"num_settled", settledResults.GetNumClaims(),
		"num_expired", expiredResults.GetNumClaims(),
		"num_discarded_faulty", numDiscardedFaultyClaims,
		"block_height", blockHeight,
	)

	return settledResults, expiredResults, numDiscardedFaultyClaims, nil
}

// ExecutePendingExpiredResults executes all pending supplier slashing operations.
func (k Keeper) ExecutePendingExpiredResults(
	ctx cosmostypes.Context,
	settlementContext *settlementContext,
	expiredResults tlm.ClaimSettlementResults,
) error {
	logger := k.logger.With("method", "ExecutePendingExpiredResults")

	// Slash any supplier(s) which failed to submit a required proof, per-claim.
	for _, expiredResult := range expiredResults {
		slashingErr := k.slashSupplierStake(ctx, settlementContext, expiredResult)

		// No error slashing, move on to the next expire claim
		if slashingErr == nil {
			continue
		}

		logger.Error(fmt.Sprintf("error slashing supplier %s: %v", expiredResult.GetSupplierOperatorAddr(), slashingErr))

		proofRequirement, requirementErr := k.proofKeeper.ProofRequirementForClaim(ctx, &expiredResult.Claim)
		if requirementErr != nil {
			return tokenomicstypes.ErrTokenomicsSettlementInternal.Wrapf(
				"unable to get proof requirement for session %q after slashing the supplier: %v",
				expiredResult.GetSessionId(), requirementErr,
			)
		}

		return tokenomicstypes.ErrTokenomicsSettlementInternal.Wrapf(
			"error slashing supplier %q for session %q with proof requirement %s: %v",
			expiredResult.GetSupplierOperatorAddr(),
			expiredResult.GetSessionId(),
			proofRequirement,
			slashingErr,
		)
	}

	return nil
}

// ExecutePendingSettledResults executes all pending mint, burn, and transfer operations.
func (k Keeper) ExecutePendingSettledResults(ctx cosmostypes.Context, settledResults tlm.ClaimSettlementResults) error {
	logger := k.logger.With("method", "ExecutePendingSettledResults")
	logger.Info(fmt.Sprintf("begin executing %d pending settlement results", len(settledResults)))

	for _, settledResult := range settledResults {
		sessionLogger := logger.With("session_id", settledResult.GetSessionId())
		sessionLogger.Info("begin executing pending settlement result")

		// Execute all pending mints.
		sessionLogger.Info(fmt.Sprintf("begin executing %d pending mints", len(settledResult.GetMints())))
		if err := k.executePendingModuleMints(ctx, sessionLogger, settledResult.GetMints()); err != nil {
			return err
		}
		sessionLogger.Info("done executing pending mints")

		// Execute all pending module to module transfers.
		sessionLogger.Info(fmt.Sprintf("begin executing %d pending module to module transfers", len(settledResult.GetModToModTransfers())))
		if err := k.executePendingModToModTransfers(ctx, sessionLogger, settledResult.GetModToModTransfers()); err != nil {
			return err
		}
		sessionLogger.Info("done executing pending module to module transfers")

		// Execute all pending module to account transfers.
		sessionLogger.Info(fmt.Sprintf("begin executing %d pending module to account transfers", len(settledResult.GetModToAcctTransfers())))
		if err := k.executePendingModToAcctTransfers(ctx, sessionLogger, settledResult.GetModToAcctTransfers()); err != nil {
			return err
		}
		sessionLogger.Info("done executing pending module to account transfers")

		// Execute all pending burns.
		sessionLogger.Info(fmt.Sprintf("begin executing %d pending burns", len(settledResult.GetBurns())))
		if err := k.executePendingModuleBurns(ctx, sessionLogger, settledResult.GetBurns()); err != nil {
			return err
		}
		sessionLogger.Info("done executing pending burns")

		// Done applying settled results for this session.
		sessionLogger.Info(fmt.Sprintf(
			"done applying settled results for session %q",
			settledResult.Claim.GetSessionHeader().GetSessionId(),
		))
	}

	logger.Info(fmt.Sprintf("done executing %d pending settlement results", len(settledResults)))

	return nil
}

// executePendingModuleMints executes all pending mint operations.
// DEV_NOTE: Mint and burn operations are ONLY applicable to module accounts.
func (k Keeper) executePendingModuleMints(
	ctx cosmostypes.Context,
	logger cosmoslog.Logger,
	mints []tokenomicstypes.MintBurnOp,
) error {
	for _, mint := range mints {
		if err := mint.Validate(); err != nil {
			return err
		}

		if err := k.bankKeeper.MintCoins(ctx, mint.DestinationModule, cosmostypes.NewCoins(mint.Coin)); err != nil {
			return tokenomicstypes.ErrTokenomicsSettlementMint.Wrapf(
				"destination module %q minting %s: %v", mint.DestinationModule, mint.Coin, err,
			)
		}
		telemetry.MintedTokensFromModule(mint.DestinationModule, float32(mint.Coin.Amount.Int64()))

		logger.Info(fmt.Sprintf(
			"minting %s coins to the %q module account, reason: %q",
			mint.Coin, mint.DestinationModule, mint.OpReason.String(),
		))
	}
	return nil
}

// executePendingModuleBurns executes all pending burn operations.
// DEV_NOTE: Mint and burn operations are ONLY applicable to module accounts.
func (k Keeper) executePendingModuleBurns(
	ctx cosmostypes.Context,
	logger cosmoslog.Logger,
	burns []tokenomicstypes.MintBurnOp,
) error {
	for _, burn := range burns {
		if err := burn.Validate(); err != nil {
			return err
		}

		if err := k.bankKeeper.BurnCoins(ctx, burn.DestinationModule, cosmostypes.NewCoins(burn.Coin)); err != nil {
			return tokenomicstypes.ErrTokenomicsSettlementBurn.Wrapf(
				"destination module %q burning %s: %v", burn.DestinationModule, burn.Coin, err,
			)
		}
		telemetry.BurnedTokensFromModule(burn.DestinationModule, float32(burn.Coin.Amount.Int64()))

		logger.Info(fmt.Sprintf(
			"burning %s coins from the %q module account, reason: %q",
			burn.Coin, burn.DestinationModule, burn.OpReason.String(),
		))
	}
	return nil
}

// executePendingModToModTransfers executes all pending module to module transfer operations.
func (k Keeper) executePendingModToModTransfers(
	ctx cosmostypes.Context,
	logger cosmoslog.Logger,
	transfers []tokenomicstypes.ModToModTransfer,
) error {
	for _, transfer := range transfers {
		if err := transfer.Validate(); err != nil {
			return err
		}

		if err := k.bankKeeper.SendCoinsFromModuleToModule(
			ctx,
			transfer.SenderModule,
			transfer.RecipientModule,
			cosmostypes.NewCoins(transfer.Coin),
		); err != nil {
			return tokenomicstypes.ErrTokenomicsSettlementTransfer.Wrapf(
				"sender module %q to recipient module %q transferring %s: %v",
				transfer.SenderModule, transfer.RecipientModule, transfer.Coin, err,
			)
		}

		logger.Info(fmt.Sprintf(
			"executing operation: transferring %s coins from the %q module account to the %q module account, reason: %q",
			transfer.Coin, transfer.SenderModule, transfer.RecipientModule, transfer.OpReason.String(),
		))
	}
	return nil
}

// executePendingModToAcctTransfers executes all pending module to account transfer operations.
func (k Keeper) executePendingModToAcctTransfers(
	ctx cosmostypes.Context,
	logger cosmoslog.Logger,
	transfers []tokenomicstypes.ModToAcctTransfer,
) error {
	for _, transfer := range transfers {
		if err := transfer.Validate(); err != nil {
			return err
		}

		recipientAddr, err := cosmostypes.AccAddressFromBech32(transfer.RecipientAddress)
		if err != nil {
			return tokenomicstypes.ErrTokenomicsSettlementTransfer.Wrapf(
				"sender module %q to recipient address %q transferring %s (reason %q): %v",
				transfer.SenderModule,
				transfer.RecipientAddress,
				transfer.Coin,
				transfer.GetOpReason(),
				err,
			)
		}

		if err = k.bankKeeper.SendCoinsFromModuleToAccount(
			ctx,
			transfer.SenderModule,
			recipientAddr,
			cosmostypes.NewCoins(transfer.Coin),
		); err != nil {
			return tokenomicstypes.ErrTokenomicsSettlementTransfer.Wrapf(
				"sender module %q to recipient address %q transferring %s (reason %q): %v",
				transfer.SenderModule,
				transfer.RecipientAddress,
				transfer.Coin,
				transfer.GetOpReason(),
				err,
			)
		}

		logger.Info(fmt.Sprintf(
			"executing operation: transferring %s coins from the %q module account to account address %q, reason: %q",
			transfer.Coin, transfer.SenderModule, transfer.RecipientAddress, transfer.OpReason.String(),
		))
	}
	return nil
}

// GetExpiringClaimsIterator returns an iterator of all claims expiring at the current block height.
// DEV_NOTE: It is exported for testing purposes.
func (k Keeper) GetExpiringClaimsIterator(
	ctx cosmostypes.Context,
	settlementContext *settlementContext,
	blockHeight int64,
) (expiringClaimsIterator sharedtypes.RecordIterator[prooftypes.Claim]) {
	sessionEndToProofWindowCloseNumBlocks := sharedtypes.GetSessionEndToProofWindowCloseBlocks(&settlementContext.sharedParams)
	expiringSessionEndHeight := blockHeight - (sessionEndToProofWindowCloseNumBlocks + 1)
	return k.proofKeeper.GetSessionEndHeightClaimsIterator(ctx, expiringSessionEndHeight)
}

// slashSupplierStake slashes the stake of a supplier and transfers the total
// slashing amount from the supplier bank module to the tokenomics module account.
// TODO_FUTURE: Slash the supplier in proportion to their stake.
func (k Keeper) slashSupplierStake(
	ctx cosmostypes.Context,
	settlementContext *settlementContext,
	claimSettlementResult *tokenomicstypes.ClaimSettlementResult,
) error {
	logger := k.logger.With("method", "slashSupplierStake")

	// Retrieve the supplier to slash.
	supplierOperatorAddress := claimSettlementResult.GetClaim().SupplierOperatorAddress
	proofParams := k.proofKeeper.GetParams(ctx)
	slashingCoin := *proofParams.GetProofMissingPenalty()
	supplierToSlash, err := settlementContext.GetSupplier(supplierOperatorAddress)
	if err != nil {
		logger.Error("failed to retrieve supplier to slash with operator address %s: %v", supplierOperatorAddress, err)
		return err
	}

	// Retrieve the supplier's initial stake.
	slashedSupplierInitialStakeCoin := supplierToSlash.GetStake()

	// Determine the supplier's remaining stake after the slashing.
	var remainingStakeCoin cosmostypes.Coin
	if slashedSupplierInitialStakeCoin.IsGTE(slashingCoin) {
		remainingStakeCoin = slashedSupplierInitialStakeCoin.Sub(slashingCoin)
	} else {
		// TODO: Emit a custom event for this case and consider custom logic where the
		// the protocol takes the remaining amount from the supplier's owner or operator balances.
		logger.Warn(fmt.Sprintf(
			"total slashing amount (%s) is greater than supplier %q stake (%s)",
			slashingCoin,
			supplierOperatorAddress,
			supplierToSlash.GetStake(),
		))
		// Set the remaining stake to 0 if the slashing amount is greater than the stake.
		remainingStakeCoin = cosmostypes.NewCoin(pocket.DenomuPOKT, math.NewInt(0))
		// Total slashing amount is the whole supplier's stake.
		slashingCoin = cosmostypes.NewCoin(pocket.DenomuPOKT, slashedSupplierInitialStakeCoin.Amount)
	}

	// Slashing a supplier's stake involves:
	// 1. Sending the slashing amount from the supplier module account to the tokenomics module account.
	// 2. Burning the slashing amount from the tokenomics module account.
	if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx,
		suppliertypes.ModuleName,
		tokenomicstypes.ModuleName,
		cosmostypes.NewCoins(slashingCoin),
	); err != nil {
		return tokenomicstypes.ErrTokenomicsSettlementInternal.Wrapf(
			"failed to send coins from module %q to module %q (reason %q): %v",
			suppliertypes.ModuleName,
			tokenomicstypes.ModuleName,
			tokenomicstypes.SettlementOpReason_UNSPECIFIED_TLM_SUPPLIER_SLASH_MODULE_TRANSFER,
			err,
		)
	}

	// Burn the slashing amount from the tokenomics module account.
	if err := k.bankKeeper.BurnCoins(ctx,
		tokenomicstypes.ModuleName,
		cosmostypes.NewCoins(slashingCoin),
	); err != nil {
		return tokenomicstypes.ErrTokenomicsSettlementInternal.Wrapf(
			"failed to burn coins from module %q (reason %q): %v",
			tokenomicstypes.ModuleName,
			tokenomicstypes.SettlementOpReason_UNSPECIFIED_TLM_SUPPLIER_SLASH_STAKE_BURN,
			err,
		)
	}

	// Update telemetry information
	if slashingCoin.Amount.IsInt64() {
		defer telemetry.SlashedTokensFromModule(suppliertypes.ModuleName, float32(slashingCoin.Amount.Int64()))
	}

	// Update the supplier's stake.
	supplierToSlash.Stake = &remainingStakeCoin
	logger.Info(fmt.Sprintf(
		"queueing operation: slash supplier owner with address %q operated by %q by %s, remaining stake: %v",
		supplierToSlash.GetOwnerAddress(),
		supplierToSlash.GetOperatorAddress(),
		slashingCoin,
		supplierToSlash.GetStake(),
	))

	// Prepare a list of events to emit.
	events := make([]cosmostypes.Msg, 0)

	// Check if the supplier's stake is below the minimum and unstake it if necessary.
	// Ensure that a slashed supplier going below min stake is unbonded only once.
	minSupplierStakeCoin := k.supplierKeeper.GetParams(ctx).MinStake
	if supplierToSlash.GetStake().IsLT(*minSupplierStakeCoin) && !supplierToSlash.IsUnbonding() {
		sharedParams := settlementContext.GetSharedParams()
		sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
		currentHeight := sdkCtx.BlockHeight()
		unstakeSessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight)

		logger.Warn(fmt.Sprintf(
			"unstaking supplier %q owned by %q due to stake (%s) below the minimum (%s)",
			supplierToSlash.GetOperatorAddress(),
			supplierToSlash.GetOwnerAddress(),
			supplierToSlash.GetStake(),
			minSupplierStakeCoin,
		))

		// Start force unstaking the supplier.
		supplierToSlash.UnstakeSessionEndHeight = uint64(unstakeSessionEndHeight)

		// Deactivate the supplier's services so they can no longer be selected to
		// service relays in the next session.
		for _, serviceConfig := range supplierToSlash.ServiceConfigHistory {
			serviceConfig.DeactivationHeight = unstakeSessionEndHeight
		}

		// Handling unbonding for slashed suppliers:
		// - Initiate unbonding at the current session end height (earliest possible time)
		// - Supplier remains staked during current session to preserve the active suppliers set
		// - Supplier will still appear in current sessions but won't receive rewards in next settlement
		// - If this settlement coincides with session end, supplier won't service further relays
		events = append(events, &suppliertypes.EventSupplierUnbondingBegin{
			Supplier:           supplierToSlash,
			Reason:             suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_BELOW_MIN_STAKE,
			SessionEndHeight:   unstakeSessionEndHeight,
			UnbondingEndHeight: unstakeSessionEndHeight,
		})
	}

	// Only update the dehydrated supplier, since the service config will remain unchanged.
	k.supplierKeeper.SetDehydratedSupplier(ctx, *supplierToSlash)

	// Emit an event that a supplier has been slashed.
	claim := claimSettlementResult.GetClaim()
	events = append(events, &tokenomicstypes.EventSupplierSlashed{
		ProofMissingPenalty:     slashingCoin.String(),
		ServiceId:               claim.SessionHeader.ServiceId,
		ApplicationAddress:      claim.SessionHeader.ApplicationAddress,
		SessionEndBlockHeight:   claim.SessionHeader.SessionEndBlockHeight,
		ClaimProofStatusInt:     int32(claim.ProofValidationStatus),
		SupplierOperatorAddress: claim.SupplierOperatorAddress,
	})

	// Emit all events.
	if err := ctx.EventManager().EmitTypedEvents(events...); err != nil {
		return err
	}

	return nil
}

// finalizeClaimTelemetry logs telemetry metrics for a claim based on its stage (e.g., EXPIRED, SETTLED).
// Meant to run deferred.
func (k Keeper) finalizeClaimTelemetry(
	claimProofStage prooftypes.ClaimProofStage,
	serviceId string,
	applicationAddress string,
	supplierOperatorAddress string,
	numRelays uint64,
	numClaimComputeUnits uint64,
	err error,
) {
	telemetry.ClaimCounter(claimProofStage.String(), 1, serviceId, applicationAddress, supplierOperatorAddress, err)
	telemetry.ClaimRelaysCounter(claimProofStage.String(), numRelays, serviceId, applicationAddress, supplierOperatorAddress, err)
	telemetry.ClaimComputeUnitsCounter(claimProofStage.String(), numClaimComputeUnits, serviceId, applicationAddress, supplierOperatorAddress, err)
}

// settleClaim processes a single claim and determines its settlement outcome.
// Responsibilities:
// - Evaluate if the claim should be settled or expired (per business logic rules)
// - Handle proof requirements, Token Logic Module (TLM) processing, and event emission
// - Return errors only for unexpected conditions that prevent normal processing (e.g., invalid claim data, cache failures, TLM or event emission errors)
//   - Normal business outcomes (expired claims, missing proofs, etc.) do NOT return errors—these are reflected in the returned ClaimProcessingContext
//   - When an error is returned, the claim should be discarded to avoid chain halt
//
// Notes:
// - Does NOT perform state changes directly
// - Prepares settlement operations for batch execution by ExecutePendingSettledResults and ExecutePendingExpiredResults
func (k Keeper) settleClaim(
	ctx cosmostypes.Context,
	settlementContext *settlementContext,
	claim prooftypes.Claim,
	logger cosmoslog.Logger,
) (*claimSettlementContext, error) {
	if err := settlementContext.ClaimCacheWarmUp(ctx, &claim); err != nil {
		return nil, err
	}

	var (
		// These values are used for cores business logic; minting, slashing, etc.
		proofRequirement prooftypes.ProofRequirementReason
		claimeduPOKT     cosmostypes.Coin

		// These values are used for basic validation, telemetry and observability.
		numClaimRelays,
		numClaimComputeUnits,
		numEstimatedComputeUnits uint64
	)

	sessionId := claim.GetSessionHeader().GetSessionId()

	// DEV_NOTE: Not every Relay (Request, Response) pair in the session is inserted into the tree.
	// See the godoc for GetNumRelays for more delays.
	numClaimRelays, err := claim.GetNumRelays()
	if err != nil {
		return nil, err
	}

	// DEV_NOTE: We are assuming that (numClaimComputeUnits := numClaimRelays * service.ComputeUnitsPerRelay)
	// because this code path is only reached if that has already been validated.
	numClaimComputeUnits, err = claim.GetNumClaimedComputeUnits()
	if err != nil {
		return nil, err
	}

	// Get the relay mining difficulty for the service that this claim is for.
	serviceId := claim.GetSessionHeader().GetServiceId()
	var relayMiningDifficulty servicetypes.RelayMiningDifficulty
	relayMiningDifficulty, err = settlementContext.GetRelayMiningDifficulty(serviceId)
	if err != nil {
		return nil, err
	}

	// Retrieve the shared module params.
	// It contains network wide governance params required to convert claims to POKT (e.g. CUTTM).
	sharedParams := settlementContext.GetSharedParams()

	// numEstimatedComputeUnits is the probabilistic estimation of the offchain
	// work done by the relay miner in this session.
	// It is derived from the claimed work and the relay mining difficulty.
	numEstimatedComputeUnits, err = claim.GetNumEstimatedComputeUnits(relayMiningDifficulty)
	if err != nil {
		return nil, err
	}

	// claimeduPOKT is the amount the supplier will receive if the claim is settled.
	// It is derived from:
	// - The claim's number of relays
	// - The service's configured CUPR
	// - The service's onchain current relay mining difficulty
	// - Global network parameters (e.g. CUTTM)
	claimeduPOKT, err = claim.GetClaimeduPOKT(sharedParams, relayMiningDifficulty)
	if err != nil {
		return nil, err
	}

	// Using the probabilistic proofs approach, determine if this expiring
	// claim required an onchain proof
	proofRequirement, err = k.proofKeeper.ProofRequirementForClaim(ctx, &claim)
	if err != nil {
		return nil, err
	}

	logger = logger.With(
		"session_id", claim.SessionHeader.SessionId,
		"supplier_operator_address", claim.SupplierOperatorAddress,
		"num_claim_compute_units", numClaimComputeUnits,
		"num_relays_in_session_tree", numClaimRelays,
		"num_estimated_compute_units", numEstimatedComputeUnits,
		"claimed_upokt", claimeduPOKT,
		"proof_requirement", proofRequirement,
	)

	// Initialize a claimSettlementResult to accumulate the results prior to executing state transitions.
	claimSettlementContext := &claimSettlementContext{
		settlementResult:     tlm.NewClaimSettlementResult(claim),
		numClaimRelays:       numClaimRelays,
		numClaimComputeUnits: numClaimComputeUnits,
	}

	// DEV_NOTE: Proof validation and claims settlement timing:
	// 	- Proof validation (proof end blocker): Executes WITHIN proof submission window
	// 	- Claims settlement (tokenomics end blocker): Executes AFTER window closes
	// This ensures proofs are validated before claims are settled
	proofIsRequired := proofRequirement != prooftypes.ProofRequirementReason_NOT_REQUIRED
	if proofIsRequired {

		var expirationReason tokenomicstypes.ClaimExpirationReason
		switch claim.ProofValidationStatus {

		// If the proof is required and not found, the claim is expired.
		case prooftypes.ClaimProofStatus_PENDING_VALIDATION:
			expirationReason = tokenomicstypes.ClaimExpirationReason_PROOF_MISSING

		// If the proof is required and invalid, the claim is expired.
		case prooftypes.ClaimProofStatus_INVALID:
			expirationReason = tokenomicstypes.ClaimExpirationReason_PROOF_INVALID

		// If the proof is required and valid, the claim is settled.
		case prooftypes.ClaimProofStatus_VALIDATED:
			expirationReason = tokenomicstypes.ClaimExpirationReason_EXPIRATION_REASON_UNSPECIFIED
		}

		// Proof was required but is invalid or not found.
		// Emit an event that a claim has expired and being removed without being settled.
		if claim.ProofValidationStatus != prooftypes.ClaimProofStatus_VALIDATED {
			claimExpiredEvent := tokenomicstypes.EventClaimExpired{
				ExpirationReason:         expirationReason,
				NumRelays:                numClaimRelays,
				NumClaimedComputeUnits:   numClaimComputeUnits,
				NumEstimatedComputeUnits: numEstimatedComputeUnits,
				ClaimedUpokt:             claimeduPOKT.String(),
				ServiceId:                claim.SessionHeader.ServiceId,
				ApplicationAddress:       claim.SessionHeader.ApplicationAddress,
				SessionEndBlockHeight:    claim.SessionHeader.SessionEndBlockHeight,
				ClaimProofStatusInt:      int32(claim.ProofValidationStatus),
				SupplierOperatorAddress:  claim.SupplierOperatorAddress,
			}
			if err = ctx.EventManager().EmitTypedEvent(&claimExpiredEvent); err != nil {
				return nil, err
			}
			logger.Info(fmt.Sprintf(
				"claim expired due to %s",
				tokenomicstypes.ClaimExpirationReason_name[int32(expirationReason)]),
			)
			return claimSettlementContext, nil
		}
	}

	// If this code path is reached, then either:
	// 1. The claim does not require a proof.
	// 2. The claim requires a proof and a valid proof was found.
	// Manage the mint & burn accounting for the claim.
	if err = k.ProcessTokenLogicModules(
		ctx,
		settlementContext,
		claimSettlementContext.settlementResult,
	); err != nil {
		logger.Error(fmt.Sprintf("error	 processing token logic modules for claim %q: %v", sessionId, err))
		return nil, err
	}

	claimSettledEvent := tokenomicstypes.EventClaimSettled{
		NumRelays:                numClaimRelays,
		NumClaimedComputeUnits:   numClaimComputeUnits,
		NumEstimatedComputeUnits: numEstimatedComputeUnits,
		ClaimedUpokt:             claimeduPOKT.String(),
		ProofRequirement:         proofRequirement,
		ServiceId:                claim.SessionHeader.ServiceId,
		ApplicationAddress:       claim.SessionHeader.ApplicationAddress,
		SessionEndBlockHeight:    claim.SessionHeader.SessionEndBlockHeight,
		ClaimProofStatusInt:      int32(claim.ProofValidationStatus),
		SupplierOperatorAddress:  claim.SupplierOperatorAddress,
	}

	if err = ctx.EventManager().EmitTypedEvent(&claimSettledEvent); err != nil {
		return nil, err
	}

	claimSettlementContext.isSettled = true

	return claimSettlementContext, nil
}

// discardFaultyClaim is used to handle unexpected faulty claims.
// It:
// - Logs the error
// - Emits an event
// - Removes the claim
func (k Keeper) discardFaultyClaim(
	sdkCtx cosmostypes.Context,
	logger cosmoslog.Logger,
	claim prooftypes.Claim,
	discardReason string,
) {
	// Emit an event that a claim settlement failed and the claim is being discarded.
	claimDiscardedEvent := tokenomicstypes.EventClaimDiscarded{
		Error:                   discardReason,
		ServiceId:               claim.SessionHeader.ServiceId,
		ApplicationAddress:      claim.SessionHeader.ApplicationAddress,
		SessionEndBlockHeight:   claim.SessionHeader.SessionEndBlockHeight,
		ClaimProofStatusInt:     int32(claim.ProofValidationStatus),
		SupplierOperatorAddress: claim.SupplierOperatorAddress,
	}
	if evtErr := sdkCtx.EventManager().EmitTypedEvent(&claimDiscardedEvent); evtErr != nil {
		logger.Error(fmt.Sprintf(
			"failed to emit claim discarded event for claim %q: %v",
			claim.SessionHeader.SessionId, evtErr,
		))
	}

	// Remove the faulty claim from the state.
	k.proofKeeper.RemoveClaim(sdkCtx, claim.SessionHeader.SessionId, claim.SupplierOperatorAddress)
}
