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
// If a claim is expired and requires a proof and a proof IS available -> it's settled.
// If a claim is expired and requires a proof and a proof IS NOT available -> it's deleted.
// If a claim is expired and does NOT require a proof -> it's settled.
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

	// Retrieve all expiring claims as an iterator.
	// DEV_NOTE: This previously retrieve a list but has been change to account for large claim counts.
	expiringClaimsIterator := k.GetExpiringClaimsIterator(ctx)
	defer expiringClaimsIterator.Close()

	blockHeight := ctx.BlockHeight()

	// Initialize results structs.
	settledResults = make(tlm.ClaimSettlementResults, 0)
	expiredResults = make(tlm.ClaimSettlementResults, 0)
	settlementContext := NewSettlementContext(ctx, &k, logger)
	numExpiringClaims := 0
	numDiscardedFaultyClaims = 0

	// Iterating over all potentially expiring claims.
	// This loop does the following:
	// - Retrieve the claim
	// - Settle the claim
	// - Remove the claim from the state
	// - Emit an event
	// - Update the relevant actors in the state
	for ; expiringClaimsIterator.Valid(); expiringClaimsIterator.Next() {
		claim, iterErr := expiringClaimsIterator.Value()
		if iterErr != nil {
			claimKey := string(expiringClaimsIterator.Key())
			logger.Error(tokenomicstypes.ErrTokenomicsSettlementInternal.Wrapf(
				"[UNEXPECTED ERROR] Critical error during claim settlement during session with key %s. Claim will be discarded to prevent chain halt: %s",
				claimKey, err,
			).Error())
			numDiscardedFaultyClaims++
			continue
		}

		// Settle the claim.
		claimProcessingContext, settlementErr := k.settleClaim(ctx, settlementContext, claim, logger)
		if settlementErr != nil {
			// Discard a faulty claim and continue iterating over the next one.
			k.discardFaultyClaim(ctx, claim, settlementErr, logger)
			numDiscardedFaultyClaims++
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
		// TODO_CONSIDERATION: Consider adding discarded faulty claims to the telemetry.
		defer k.finalizeTelemetry(
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
			"claim processed",
			"session_id", claim.SessionHeader.SessionId,
			"supplier", claim.SupplierOperatorAddress,
			"settled", claimProcessingContext.isSettled,
			"block_height", blockHeight,
		)
	}

	// Claims settlement results have been computed when this code path is reached.
	// The onchain data now needs to be updated.

	// Persist the state of all the applications and suppliers involved in the claims.
	// This is done in a single batch to reduce the number of writes to state storage.
	settlementContext.FlushAllActorsToStore(ctx)

	logger.Info(
		"expiring claims processed",
		"num_expiring_claims", numExpiringClaims,
		"num_settled", settledResults.GetNumClaims(),
		"num_expired", expiredResults.GetNumClaims(),
		"num_discarded_faulty", numDiscardedFaultyClaims,
		"block_height", blockHeight,
	)

	// TODO_IMPROVEMENT: Pre-validate all settlement operations before execution
	//
	// Goals:
	// - Validate all settlement and expiration results will succeed before executing any
	// - Allow discarding the whole settlement batch if any operation would unexpectedly fail
	// - Prevent chain halts due to failed settlement execution
	//
	// Rationale:
	// - Ignoring errors would lead to partial execution and leave the chain in an inconsistent state
	// - Not ignoring execution errors would halt the chain
	// - All-or-nothing approach ensures atomic settlement process and continues chain operation

	// Execute all the pending mint, burn, and transfer operations.
	if err = k.ExecutePendingSettledResults(ctx, settledResults); err != nil {
		return settledResults, expiredResults, numDiscardedFaultyClaims, err
	}

	// Slash all suppliers who failed to submit a required proof.
	if err = k.ExecutePendingExpiredResults(ctx, settlementContext, expiredResults); err != nil {
		return settledResults, expiredResults, numDiscardedFaultyClaims, err
	}

	// Persist the state of all the applications and suppliers involved in the claims.
	// This is done in a single batch to reduce the number of writes to state storage.
	settlementContext.FlushAllActorsToStore(ctx)

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
// IMPORTANT: If the execution of any such operation fails, the chain will halt.
// In this case, the state is left how it was immediately prior to the execution of
// the operation which failed.
func (k Keeper) ExecutePendingExpiredResults(
	ctx cosmostypes.Context,
	settlementContext *settlementContext,
	expiredResults tlm.ClaimSettlementResults,
) error {
	logger := k.logger.With("method", "ExecutePendingExpiredResults")

	// Slash any supplier(s) which failed to submit a required proof, per-claim.
	for _, expiredResult := range expiredResults {
		if err := k.slashSupplierStake(ctx, settlementContext, expiredResult); err != nil {
			logger.Error(fmt.Sprintf("error slashing supplier %s: %s", expiredResult.GetSupplierOperatorAddr(), err))

			proofRequirement, requirementErr := k.proofKeeper.ProofRequirementForClaim(ctx, &expiredResult.Claim)
			if requirementErr != nil {
				return tokenomicstypes.ErrTokenomicsSettlementInternal.Wrapf(
					"unable to get proof requirement for session %q: %s",
					expiredResult.GetSessionId(), err,
				)
			}

			return tokenomicstypes.ErrTokenomicsSettlementInternal.Wrapf(
				"error slashing supplier %q for session %q with proof requirement %s: %s",
				expiredResult.GetSupplierOperatorAddr(),
				expiredResult.GetSessionId(),
				proofRequirement,
				err,
			)
		}
	}

	return nil
}

// ExecutePendingSettledResults executes all pending mint, burn, and transfer operations.
// IMPORTANT: If the execution of any pending operation fails, the chain will halt.
// In this case, the state is left how it was immediately prior to the execution of
// the operation which failed.
// TODO_MAINNET_MIGRATION(@bryanchriswhite): Make this more "atomic", such that it reverts the state back to just prior
// to settling the offending claim.
func (k Keeper) ExecutePendingSettledResults(ctx cosmostypes.Context, settledResults tlm.ClaimSettlementResults) error {
	logger := k.logger.With("method", "ExecutePendingSettledResults")
	logger.Info(fmt.Sprintf("begin executing %d pending settlement results", len(settledResults)))

	for _, settledResult := range settledResults {
		sessionLogger := logger.With("session_id", settledResult.GetSessionId())
		sessionLogger.Info("begin executing pending settlement result")

		sessionLogger.Info(fmt.Sprintf("begin executing %d pending mints", len(settledResult.GetMints())))
		if err := k.executePendingModuleMints(ctx, sessionLogger, settledResult.GetMints()); err != nil {
			return err
		}
		sessionLogger.Info("done executing pending mints")

		sessionLogger.Info(fmt.Sprintf("begin executing %d pending module to module transfers", len(settledResult.GetModToModTransfers())))
		if err := k.executePendingModToModTransfers(ctx, sessionLogger, settledResult.GetModToModTransfers()); err != nil {
			return err
		}
		sessionLogger.Info("done executing pending module account to module account transfers")

		sessionLogger.Info(fmt.Sprintf("begin executing %d pending module to account transfers", len(settledResult.GetModToAcctTransfers())))
		if err := k.executePendingModToAcctTransfers(ctx, sessionLogger, settledResult.GetModToAcctTransfers()); err != nil {
			return err
		}
		sessionLogger.Info("done executing pending module to account transfers")

		sessionLogger.Info(fmt.Sprintf("begin executing %d pending burns", len(settledResult.GetBurns())))
		if err := k.executePendingModuleBurns(ctx, sessionLogger, settledResult.GetBurns()); err != nil {
			return err
		}
		sessionLogger.Info("done executing pending burns")

		sessionLogger.Info("done executing pending settlement result")

		sessionLogger.Info(fmt.Sprintf(
			"done applying settled results for session %q",
			settledResult.Claim.GetSessionHeader().GetSessionId(),
		))
	}

	logger.Info("done executing pending settlement results")

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
			return tokenomicstypes.ErrTokenomicsSettlementModuleMint.Wrapf(
				"destination module %q minting %s: %s", mint.DestinationModule, mint.Coin, err,
			)
		}

		logger.Info(fmt.Sprintf(
			"executing operation: minting %s coins to the %q module account, reason: %q",
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
			return tokenomicstypes.ErrTokenomicsSettlementModuleBurn.Wrapf(
				"destination module %q burning %s: %s", burn.DestinationModule, burn.Coin, err,
			)
		}

		logger.Info(fmt.Sprintf(
			"executing operation: burning %s coins from the %q module account, reason: %q",
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
				"sender module %q to recipient module %q transferring %s: %s",
				transfer.SenderModule, transfer.RecipientModule, transfer.Coin, err,
			)
		}

		logger.Info(fmt.Sprintf(
			"executing operation: transfering %s coins from the %q module account to the %q module account, reason: %q",
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
				"sender module %q to recipient address %q transferring %s (reason %q): %s",
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
				"sender module %q to recipient address %q transferring %s (reason %q): %s",
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

// GetExpiringClaimsIterator returns an iterator of all claims expiring at the current (i.e the context's) block height.
// This is the height at which the proof window closes.
// If the proof window closes and a proof IS NOT required -> settle the claim.
// If the proof window closes and a proof IS required -> only settle it if a proof is available.
// DEV_NOTE: It is exported for testing purposes.
func (k Keeper) GetExpiringClaimsIterator(ctx cosmostypes.Context) (expiringClaimsIterator sharedtypes.RecordIterator[prooftypes.Claim]) {
	// TODO_IMPROVE(@bryanchriswhite):
	//   1. Move height logic up to SettlePendingClaims.
	//   2. Ensure that claims are only settled or expired on a session end height.
	//     2a. This likely also requires adding validation to the shared module params.
	blockHeight := ctx.BlockHeight()

	// NB: This error can be safely ignored as onchain SharedQueryClient implementation cannot return an error.
	sharedParams, _ := k.sharedQuerier.GetParams(ctx)

	// expiringSessionEndHeight is the session end height of the session whose proof
	// window has most recently closed.
	sessionEndToProofWindowCloseNumBlocks := sharedtypes.GetSessionEndToProofWindowCloseBlocks(sharedParams)
	expiringSessionEndHeight := blockHeight - (sessionEndToProofWindowCloseNumBlocks + 1)

	return k.proofKeeper.GetSessionEndHeightClaimsIterator(ctx, expiringSessionEndHeight)
}

// slashSupplierStake slashes the stake of a supplier and transfers the total
// slashing amount from the supplier bank module to the tokenomics module account.
func (k Keeper) slashSupplierStake(
	ctx cosmostypes.Context,
	settlementContext *settlementContext,
	ClaimSettlementResult *tokenomicstypes.ClaimSettlementResult,
) error {
	logger := k.logger.With("method", "slashSupplierStake")

	supplierOperatorAddress := ClaimSettlementResult.GetClaim().SupplierOperatorAddress
	proofParams := k.proofKeeper.GetParams(ctx)
	slashingCoin := *proofParams.GetProofMissingPenalty()

	supplierToSlash, err := settlementContext.GetSupplier(supplierOperatorAddress)
	if err != nil {
		return err
	}

	slashedSupplierInitialStakeCoin := supplierToSlash.GetStake()

	var remainingStakeCoin cosmostypes.Coin
	if slashedSupplierInitialStakeCoin.IsGTE(slashingCoin) {
		remainingStakeCoin = slashedSupplierInitialStakeCoin.Sub(slashingCoin)
	} else {
		// TODO_MAINNET_MIGRATION: Consider emitting an event for this case.
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

	// Since staking mints tokens to the supplier module account, to have a correct
	// accounting, the slashing amount needs to be sent from the supplier module
	// account to the tokenomics module account.
	if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx,
		suppliertypes.ModuleName,
		tokenomicstypes.ModuleName,
		cosmostypes.NewCoins(slashingCoin),
	); err != nil {
		return tokenomicstypes.ErrTokenomicsSettlementInternal.Wrapf(
			"failed to send coins from module %q to module %q (reason %q): %s",
			suppliertypes.ModuleName,
			tokenomicstypes.ModuleName,
			tokenomicstypes.SettlementOpReason_UNSPECIFIED_TLM_SUPPLIER_SLASH_MODULE_TRANSFER,
			err,
		)
	}

	if err := k.bankKeeper.BurnCoins(ctx,
		tokenomicstypes.ModuleName,
		cosmostypes.NewCoins(slashingCoin),
	); err != nil {
		return tokenomicstypes.ErrTokenomicsSettlementInternal.Wrapf(
			"failed to burn coins from module %q (reason %q): %s",
			tokenomicstypes.ModuleName,
			tokenomicstypes.SettlementOpReason_UNSPECIFIED_TLM_SUPPLIER_SLASH_STAKE_BURN,
			err,
		)
	}

	// Update telemetry information
	if slashingCoin.Amount.IsInt64() {
		defer telemetry.SlashedTokensFromModule(suppliertypes.ModuleName, float32(slashingCoin.Amount.Int64()))
	}

	supplierToSlash.Stake = &remainingStakeCoin

	logger.Info(fmt.Sprintf(
		"queueing operation: slash supplier owner with address %q operated by %q by %s, remaining stake: %s",
		supplierToSlash.GetOwnerAddress(),
		supplierToSlash.GetOperatorAddress(),
		slashingCoin,
		supplierToSlash.GetStake(),
	))

	events := make([]cosmostypes.Msg, 0)

	// Check if the supplier's stake is below the minimum and unstake it if necessary.
	minSupplierStakeCoin := k.supplierKeeper.GetParams(ctx).MinStake
	// TODO_MAINNET_MIGRATION(@red-0ne): SettlePendingClaims is called at the end of every block,
	// but not every block corresponds to the end of a session. This may lead to a situation
	// where a force unstaked supplier may still be able to interact with a Gateway or Application.
	// However, claims are only processed when sessions end.
	// INVESTIGATION: This requires an investigation if the race condition exists
	// at all and fixed only if it does.
	// Ensure that a slashed supplier going below min stake is unbonded only once.
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

		// TODO_MAINNET_MIGRATION: Should we just remove the supplier if the stake is
		// below the minimum, at the risk of making the offchain actors have an
		// inconsistent session supplier list? See the comment above for more details.
		supplierToSlash.UnstakeSessionEndHeight = uint64(unstakeSessionEndHeight)

		// Deactivate the supplier's services so they can no longer be selected to
		// service relays in the next session.
		for _, serviceConfig := range supplierToSlash.ServiceConfigHistory {
			serviceConfig.DeactivationHeight = unstakeSessionEndHeight
		}

		dehydratedSupplier := *supplierToSlash
		dehydratedSupplier.ServiceUsageMetrics = make(map[string]*sharedtypes.ServiceUsageMetrics)

		events = append(events, &suppliertypes.EventSupplierUnbondingBegin{
			Supplier:         &dehydratedSupplier,
			Reason:           suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_BELOW_MIN_STAKE,
			SessionEndHeight: unstakeSessionEndHeight,
			// Handling unbonding for slashed suppliers:
			// - Initiate unbonding at the current session end height (earliest possible time)
			// - Supplier remains staked during current session to preserve the active suppliers set
			// - Supplier will still appear in current sessions but won't receive rewards in next settlement
			// - If this settlement coincides with session end, supplier won't service further relays
			UnbondingEndHeight: unstakeSessionEndHeight,
		})
	}

	// Only update the dehydrated supplier, since the service config will remain unchanged.
	k.supplierKeeper.SetDehydratedSupplier(ctx, *supplierToSlash)

	claim := ClaimSettlementResult.GetClaim()

	// Emit an event that a supplier has been slashed.
	events = append(events, &tokenomicstypes.EventSupplierSlashed{
		Claim:               &claim,
		ProofMissingPenalty: &slashingCoin,
	})

	if err := ctx.EventManager().EmitTypedEvents(events...); err != nil {
		return err
	}

	// TODO_POST_MAINNET: Handle the case where the total slashing amount is
	// greater than the supplier's stake. The protocol could take the remaining
	// amount from the supplier's owner or operator balances.

	return nil
}

// finalizeTelemetry logs telemetry metrics for a claim based on its stage (e.g., EXPIRED, SETTLED).
// Meant to run deferred.
func (k Keeper) finalizeTelemetry(
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
//
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

	proofIsRequired := proofRequirement != prooftypes.ProofRequirementReason_NOT_REQUIRED
	if proofIsRequired {
		// IMPORTANT: Proof validation and claims settlement timing:
		// 	- Proof validation (proof end blocker): Executes WITHIN proof submission window
		// 	- Claims settlement (tokenomics end blocker): Executes AFTER window closes
		// This ensures proofs are validated before claims are settled

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

		if claim.ProofValidationStatus != prooftypes.ClaimProofStatus_VALIDATED {
			// TODO_MAINNET_MIGRATION(@red-0ne): Slash the supplier in proportion to their stake.
			// TODO_POST_MAINNET: Consider allowing suppliers to RemoveClaim via a new message in case it was sent by accident

			// Proof was required but is invalid or not found.
			// Emit an event that a claim has expired and being removed without being settled.
			claimExpiredEvent := tokenomicstypes.EventClaimExpired{
				Claim:                    &claim,
				ExpirationReason:         expirationReason,
				NumRelays:                numClaimRelays,
				NumClaimedComputeUnits:   numClaimComputeUnits,
				NumEstimatedComputeUnits: numEstimatedComputeUnits,
				ClaimedUpokt:             &claimeduPOKT,
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
		Claim:                    &claim,
		NumRelays:                numClaimRelays,
		NumClaimedComputeUnits:   numClaimComputeUnits,
		NumEstimatedComputeUnits: numEstimatedComputeUnits,
		ClaimedUpokt:             &claimeduPOKT,
		ProofRequirement:         proofRequirement,
		SettlementResult:         *claimSettlementContext.settlementResult,
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
//
// TODO_POST_MIGRATION_INVESTIGATION(@olshansk): Find a mitigation plan for discarded claims.
// The tradeoff of avoiding chain halts versus handling faulty claims is worth it for now.
func (k Keeper) discardFaultyClaim(
	sdkCtx cosmostypes.Context,
	claim prooftypes.Claim,
	err error,
	logger cosmoslog.Logger,
) {
	logger.Error(tokenomicstypes.ErrTokenomicsSettlementInternal.Wrapf(
		"[UNEXPECTED ERROR] Critical error during claim settlement during session %q for operator %s and service %s. Claim will be discarded to prevent chain halt: %s",
		claim.SessionHeader.SessionId, claim.SupplierOperatorAddress, claim.SessionHeader.ServiceId, err,
	).Error())

	// Emit an event that a claim settlement failed and the claim is being discarded.

	dehydratedClaim := claim.GetDehydratedClaim()
	claimDiscardedEvent := tokenomicstypes.EventClaimDiscarded{
		Claim: &dehydratedClaim,
		Error: err.Error(),
	}
	if evtErr := sdkCtx.EventManager().EmitTypedEvent(&claimDiscardedEvent); evtErr != nil {
		logger.Error(fmt.Sprintf(
			"failed to emit claim discarded event for claim %q: %s",
			claim.SessionHeader.SessionId, evtErr,
		))
	}

	// Remove the faulty claim from the state.
	k.proofKeeper.RemoveClaim(sdkCtx, claim.SessionHeader.SessionId, claim.SupplierOperatorAddress)
}
