package keeper

import (
	"context"
	"fmt"

	cosmoslog "cosmossdk.io/log"
	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/telemetry"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicekeeper "github.com/pokt-network/poktroll/x/service/keeper"
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
	err error,
) {
	logger := k.Logger().With("method", "SettlePendingClaims")

	expiringClaims, err := k.GetExpiringClaims(ctx)
	if err != nil {
		return settledResults, expiredResults, err
	}

	// Capture the applications initial stake which will be used to calculate the
	// max share any claim could burn from the application stake.
	// This ensures that each supplier can calculate the maximum amount it can take
	// from an application's stake.
	applicationInitialStakeMap, err := k.getApplicationInitialStakeMap(ctx, expiringClaims)
	if err != nil {
		return settledResults, expiredResults, err
	}

	blockHeight := ctx.BlockHeight()

	logger.Info(fmt.Sprintf("found %d expiring claims at block height %d", len(expiringClaims), blockHeight))

	// Initialize results structs.
	settledResults = make(tlm.ClaimSettlementResults, 0)
	expiredResults = make(tlm.ClaimSettlementResults, 0)

	logger.Debug("settling expiring claims")
	for _, claim := range expiringClaims {
		var (
			proofRequirement prooftypes.ProofRequirementReason
			claimeduPOKT     cosmostypes.Coin
			numClaimRelays,
			numClaimComputeUnits,
			numEstimatedComputeUnits uint64
		)

		sessionId := claim.GetSessionHeader().GetSessionId()

		// NB: Not every (Req, Res) pair in the session is inserted into the tree due
		// to the relay mining difficulty. This is the count of non-empty leaves that
		// matched the necessary difficulty.
		numClaimRelays, err = claim.GetNumRelays()
		if err != nil {
			return settledResults, expiredResults, err
		}

		// DEV_NOTE: We are assuming that (numClaimComputeUnits := numClaimRelays * service.ComputeUnitsPerRelay)
		// because this code path is only reached if that has already been validated.
		numClaimComputeUnits, err = claim.GetNumClaimedComputeUnits()
		if err != nil {
			return settledResults, expiredResults, err
		}

		// Get the relay mining difficulty for the service that this claim is for.
		serviceId := claim.GetSessionHeader().GetServiceId()
		relayMiningDifficulty, found := k.serviceKeeper.GetRelayMiningDifficulty(ctx, serviceId)
		if !found {
			targetNumRelays := k.serviceKeeper.GetParams(ctx).TargetNumRelays
			relayMiningDifficulty = servicekeeper.NewDefaultRelayMiningDifficulty(
				ctx,
				logger,
				serviceId,
				targetNumRelays,
				targetNumRelays,
			)
		}
		// numEstimatedComputeUnits is the probabilistic estimation of the offchain
		// work done by the relay miner in this session. It is derived from the claimed
		// work and the relay mining difficulty.
		numEstimatedComputeUnits, err = claim.GetNumEstimatedComputeUnits(relayMiningDifficulty)
		if err != nil {
			return settledResults, expiredResults, err
		}

		sharedParams := k.sharedKeeper.GetParams(ctx)
		// claimeduPOKT is the amount of uPOKT that the supplier would receive if the
		// claim is settled. It is derived from the claimed number of relays, the current
		// service mining difficulty and the global network parameters.
		claimeduPOKT, err = claim.GetClaimeduPOKT(sharedParams, relayMiningDifficulty)
		if err != nil {
			return settledResults, expiredResults, err
		}

		// Using the probabilistic proofs approach, determine if this expiring
		// claim required an onchain proof
		proofRequirement, err = k.proofKeeper.ProofRequirementForClaim(ctx, &claim)
		if err != nil {
			return settledResults, expiredResults, err
		}

		logger = k.logger.With(
			"session_id", sessionId,
			"supplier_operator_address", claim.SupplierOperatorAddress,
			"num_claim_compute_units", numClaimComputeUnits,
			"num_relays_in_session_tree", numClaimRelays,
			"num_estimated_compute_units", numEstimatedComputeUnits,
			"claimed_upokt", claimeduPOKT,
			"proof_requirement", proofRequirement,
		)

		// Initialize a ClaimSettlementResult to accumulate the results prior to executing state transitions.
		ClaimSettlementResult := tlm.NewClaimSettlementResult(claim)

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
				// TODO_BETA(@red-0ne): Slash the supplier in proportion to their stake.
				// TODO_POST_MAINNET: Consider allowing suppliers to RemoveClaim via a new
				// message in case it was sent by accident

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
					return settledResults, expiredResults, err
				}

				logger.Info(fmt.Sprintf(
					"claim expired due to %s",
					tokenomicstypes.ClaimExpirationReason_name[int32(expirationReason)]),
				)

				// Collect all the slashed supplier operator addresses to later check
				// if they have to be unstaked because of stake below the minimum.
				// The unstaking check is not done here because the slashed supplier may
				// have other valid claims and the protocol might want to touch the supplier
				// owner or operator balances if the stake is negative.

				// The claim is no longer necessary, so there's no need for it to take up onchain space.
				k.proofKeeper.RemoveClaim(ctx, sessionId, claim.SupplierOperatorAddress)

				// Append the settlement result to the expired results.
				expiredResults.Append(ClaimSettlementResult)

				// Telemetry - defer telemetry calls so that they reference the final values the relevant variables.
				defer k.finalizeTelemetry(
					prooftypes.ClaimProofStage_EXPIRED,
					claim.SessionHeader.ServiceId,
					claim.SessionHeader.ApplicationAddress,
					claim.SupplierOperatorAddress,
					numClaimRelays,
					numClaimComputeUnits,
					err,
				)

				continue
			}
		}

		// If this code path is reached, then either:
		// 1. The claim does not require a proof.
		// 2. The claim requires a proof and a valid proof was found.

		appAddress := claim.GetSessionHeader().GetApplicationAddress()
		applicationInitialStake := applicationInitialStakeMap[appAddress]

		// TODO_MAINNET(@red-0ne): Add tests to ensure that a zero application stake
		// is handled correctly.
		if applicationInitialStake.IsZero() {
			logger.Error(fmt.Sprintf("application %q has a zero initial stake", appAddress))

			continue
		}

		// Manage the mint & burn accounting for the claim.
		if err = k.ProcessTokenLogicModules(ctx, ClaimSettlementResult, applicationInitialStake); err != nil {
			logger.Error(fmt.Sprintf("error processing token logic modules for claim %q: %v", claim.SessionHeader.SessionId, err))
			return settledResults, expiredResults, err
		}

		// Append the token logic module processing ClaimSettlementResult to the settled results.
		settledResults.Append(ClaimSettlementResult)

		claimSettledEvent := tokenomicstypes.EventClaimSettled{
			Claim:                    &claim,
			NumRelays:                numClaimRelays,
			NumClaimedComputeUnits:   numClaimComputeUnits,
			NumEstimatedComputeUnits: numEstimatedComputeUnits,
			ClaimedUpokt:             &claimeduPOKT,
			ProofRequirement:         proofRequirement,
			SettlementResult:         *ClaimSettlementResult,
		}

		if err = ctx.EventManager().EmitTypedEvent(&claimSettledEvent); err != nil {
			return settledResults, expiredResults, err
		}

		logger.Info("claim settled")

		// The claim & proof are no longer necessary, so there's no need for them
		// to take up onchain space.
		k.proofKeeper.RemoveClaim(ctx, sessionId, claim.SupplierOperatorAddress)

		logger.Debug(fmt.Sprintf("Successfully settled claim for session ID %q at block height %d", claim.SessionHeader.SessionId, blockHeight))

		// Telemetry - defer telemetry calls so that they reference the final values the relevant variables.
		defer k.finalizeTelemetry(
			prooftypes.ClaimProofStage_SETTLED,
			claim.SessionHeader.ServiceId,
			claim.SessionHeader.ApplicationAddress,
			claim.SupplierOperatorAddress,
			numClaimRelays,
			numClaimComputeUnits,
			err,
		)
	}
	// Execute all the pending mint, burn, and transfer operations.
	if err = k.ExecutePendingSettledResults(ctx, settledResults); err != nil {
		return settledResults, expiredResults, err
	}

	// Slash all suppliers who failed to submit a required proof.
	if err = k.ExecutePendingExpiredResults(ctx, expiredResults); err != nil {
		return settledResults, expiredResults, err
	}

	logger.Info(fmt.Sprintf(
		"settled %d and expired %d claims at block height %d",
		settledResults.GetNumClaims(),
		expiredResults.GetNumClaims(),
		blockHeight,
	))

	return settledResults, expiredResults, nil
}

// ExecutePendingExpiredResults executes all pending supplier slashing operations.
// IMPORTANT: If the execution of any such operation fails, the chain will halt.
// In this case, the state is left how it was immediately prior to the execution of
// the operation which failed.
func (k Keeper) ExecutePendingExpiredResults(ctx cosmostypes.Context, expiredResults tlm.ClaimSettlementResults) error {
	logger := k.logger.With("method", "ExecutePendingExpiredResults")

	// Slash any supplier(s) which failed to submit a required proof, per-claim.
	for _, expiredResult := range expiredResults {
		if err := k.slashSupplierStake(ctx, expiredResult); err != nil {
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
// TODO_MAINNET(@bryanchriswhite): Make this more "atomic", such that it reverts the state back to just prior
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

		recepientAddr, err := cosmostypes.AccAddressFromBech32(transfer.RecipientAddress)
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
			recepientAddr,
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
			"executing operation: transfering %s coins from the %q module account to account address %q, reason: %q",
			transfer.Coin, transfer.SenderModule, transfer.RecipientAddress, transfer.OpReason.String(),
		))
	}
	return nil
}

// GetExpiringClaims returns all claims that are expiring at the current block height.
// This is the height at which the proof window closes.
// If the proof window closes and a proof IS NOT required -> settle the claim.
// If the proof window closes and a proof IS required -> only settle it if a proof is available.
// DEV_NOTE: It is exported for testing purposes.
func (k Keeper) GetExpiringClaims(ctx cosmostypes.Context) (expiringClaims []prooftypes.Claim, _ error) {
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

	var nextKey []byte
	for {
		claimsRes, err := k.proofKeeper.AllClaims(ctx, &prooftypes.QueryAllClaimsRequest{
			Pagination: &query.PageRequest{
				Key: nextKey,
			},
			Filter: &prooftypes.QueryAllClaimsRequest_SessionEndHeight{
				SessionEndHeight: uint64(expiringSessionEndHeight),
			},
		})
		if err != nil {
			return nil, err
		}

		expiringClaims = append(expiringClaims, claimsRes.GetClaims()...)

		// Continue if there are more claims to fetch.
		nextKey = claimsRes.Pagination.GetNextKey()
		if nextKey != nil {
			continue
		}

		break
	}

	// Return the actually expiring claims
	return expiringClaims, nil
}

// slashSupplierStake slashes the stake of a supplier and transfers the total
// slashing amount from the supplier bank module to the tokenomics module account.
func (k Keeper) slashSupplierStake(
	ctx cosmostypes.Context,
	ClaimSettlementResult *tokenomicstypes.ClaimSettlementResult,
) error {
	logger := k.logger.With("method", "slashSupplierStake")

	supplierOperatorAddress := ClaimSettlementResult.GetClaim().SupplierOperatorAddress
	proofParams := k.proofKeeper.GetParams(ctx)
	slashingCoin := *proofParams.GetProofMissingPenalty()

	supplierToSlash, isSupplierFound := k.supplierKeeper.GetSupplier(ctx, supplierOperatorAddress)
	if !isSupplierFound {
		return tokenomicstypes.ErrTokenomicsSupplierNotFound.Wrapf(
			"cannot slash supplier with operator address: %q",
			supplierOperatorAddress,
		)
	}

	slashedSupplierInitialStakeCoin := supplierToSlash.GetStake()

	var remainingStakeCoin cosmostypes.Coin
	if slashedSupplierInitialStakeCoin.IsGTE(slashingCoin) {
		remainingStakeCoin = slashedSupplierInitialStakeCoin.Sub(slashingCoin)
	} else {
		// TODO_MAINNET: Consider emitting an event for this case.
		logger.Warn(fmt.Sprintf(
			"total slashing amount (%s) is greater than supplier %q stake (%s)",
			slashingCoin,
			supplierOperatorAddress,
			supplierToSlash.GetStake(),
		))

		// Set the remaining stake to 0 if the slashing amount is greater than the stake.
		remainingStakeCoin = cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(0))
		// Total slashing amount is the whole supplier's stake.
		slashingCoin = cosmostypes.NewCoin(volatile.DenomuPOKT, slashedSupplierInitialStakeCoin.Amount)
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
	// TODO_MAINNET(@red-0ne): SettlePendingClaims is called at the end of every block,
	// but not every block corresponds to the end of a session. This may lead to a situation
	// where a force unstaked supplier may still be able to interact with a Gateway or Application.
	// However, claims are only processed when sessions end.
	// INVESTIGATION: This requires an investigation if the race condition exists
	// at all and fixed only if it does.
	if supplierToSlash.GetStake().IsLT(*minSupplierStakeCoin) {
		sharedParams := k.sharedKeeper.GetParams(ctx)
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

		// TODO_MAINNET: Should we just remove the supplier if the stake is
		// below the minimum, at the risk of making the offchain actors have an
		// inconsistent session supplier list? See the comment above for more details.
		supplierToSlash.UnstakeSessionEndHeight = uint64(unstakeSessionEndHeight)

		unbondingEndHeight := sharedtypes.GetSupplierUnbondingEndHeight(&sharedParams, &supplierToSlash)
		events = append(events, &suppliertypes.EventSupplierUnbondingBegin{
			Supplier:           &supplierToSlash,
			Reason:             suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_BELOW_MIN_STAKE,
			SessionEndHeight:   unstakeSessionEndHeight,
			UnbondingEndHeight: unbondingEndHeight,
		})
	}

	k.supplierKeeper.SetSupplier(ctx, supplierToSlash)

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

// getApplicationInitialStakeMap returns a map from an application address to the
// initial stake of the application. This is used to calculate the maximum share
// any claim could burn from the application stake.
func (k Keeper) getApplicationInitialStakeMap(
	ctx context.Context,
	expiringClaims []prooftypes.Claim,
) (applicationInitialStakeMap map[string]cosmostypes.Coin, err error) {
	applicationInitialStakeMap = make(map[string]cosmostypes.Coin)
	for _, claim := range expiringClaims {
		appAddress := claim.SessionHeader.ApplicationAddress
		// The same application is participating in other claims being settled,
		// so we already capture its initial stake.
		if _, isAppFound := applicationInitialStakeMap[appAddress]; isAppFound {
			continue
		}

		app, isAppFound := k.applicationKeeper.GetApplication(ctx, appAddress)
		if !isAppFound {
			err := apptypes.ErrAppNotFound.Wrapf(
				"trying to settle a claim for an application that does not exist (which should never happen) with address: %q",
				appAddress,
			)
			return nil, err
		}

		applicationInitialStakeMap[appAddress] = *app.GetStake()
	}

	return applicationInitialStakeMap, nil
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
