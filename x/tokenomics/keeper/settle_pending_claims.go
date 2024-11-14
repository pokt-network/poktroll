package keeper

import (
	"context"
	"fmt"

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
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// SettlePendingClaims settles all pending (i.e. expiring) claims.
// If a claim is expired and requires a proof and a proof IS available -> it's settled.
// If a claim is expired and requires a proof and a proof IS NOT available -> it's deleted.
// If a claim is expired and does NOT require a proof -> it's settled.
// Events are emitted for each claim that is settled or removed.
// On-chain Claims & Proofs are deleted after they're settled or expired to free up space.
//
// TODO_TECHDEBT: Refactor this function to return a struct instead of multiple return values.
func (k Keeper) SettlePendingClaims(ctx cosmostypes.Context) (
	settledResult tokenomicstypes.PendingClaimsResult,
	expiredResult tokenomicstypes.PendingClaimsResult,
	err error,
) {
	logger := k.Logger().With("method", "SettlePendingClaims")

	expiringClaims, err := k.getExpiringClaims(ctx)
	if err != nil {
		return settledResult, expiredResult, err
	}

	// Capture the applications initial stake which will be used to calculate the
	// max share any claim could burn from the application stake.
	// This ensures that each supplier can calculate the maximum amount it can take
	// from an application's stake.
	applicationInitialStakeMap, err := k.getApplicationInitialStakeMap(ctx, expiringClaims)
	if err != nil {
		return settledResult, expiredResult, err
	}

	blockHeight := ctx.BlockHeight()

	logger.Info(fmt.Sprintf("found %d expiring claims at block height %d", len(expiringClaims), blockHeight))

	// Initialize results structs.
	settledResult = tokenomicstypes.NewClaimSettlementResult()
	expiredResult = tokenomicstypes.NewClaimSettlementResult()

	// A map from a supplier operator address to the number of expired claims that
	// supplier has in this session.
	// Expired claims due to reasons such as invalid or missing proofs when required.
	supplierToExpiredClaimCount := make(map[string]uint64)
	logger.Debug("settling expiring claims")
	for _, claim := range expiringClaims {
		var (
			proofRequirement     prooftypes.ProofRequirementReason
			numClaimRelays       uint64
			numClaimComputeUnits uint64
		)

		sessionId := claim.GetSessionHeader().GetSessionId()

		// NB: Not every (Req, Res) pair in the session is inserted into the tree due
		// to the relay mining difficulty. This is the count of non-empty leaves that
		// matched the necessary difficulty.
		numClaimRelays, err = claim.GetNumRelays()
		if err != nil {
			return settledResult, expiredResult, err
		}

		// DEV_NOTE: We are assuming that (numClaimComputeUnits := numClaimRelays * service.ComputeUnitsPerRelay)
		// because this code path is only reached if that has already been validated.
		numClaimComputeUnits, err = claim.GetNumClaimedComputeUnits()
		if err != nil {
			return settledResult, expiredResult, err
		}

		// Get the relay mining difficulty for the service that this claim is for.
		serviceId := claim.GetSessionHeader().GetServiceId()
		relayMiningDifficulty, found := k.serviceKeeper.GetRelayMiningDifficulty(ctx, serviceId)
		if !found {
			relayMiningDifficulty = servicekeeper.NewDefaultRelayMiningDifficulty(ctx, logger, serviceId, servicekeeper.TargetNumRelays)
		}
		// numEstimatedComputeUnits is the probabilistic estimation of the off-chain
		// work done by the relay miner in this session. It is derived from the claimed
		// work and the relay mining difficulty.
		numEstimatedComputeUnits, err := claim.GetNumEstimatedComputeUnits(relayMiningDifficulty)
		if err != nil {
			return settledResult, expiredResult, err
		}

		sharedParams := k.sharedKeeper.GetParams(ctx)
		// claimeduPOKT is the amount of uPOKT that the supplier would receive if the
		// claim is settled. It is derived from the claimed number of relays, the current
		// service mining difficulty and the global network parameters.
		claimeduPOKT, err := claim.GetClaimeduPOKT(sharedParams, relayMiningDifficulty)
		if err != nil {
			return settledResult, expiredResult, err
		}

		proof, isProofFound := k.proofKeeper.GetProof(ctx, sessionId, claim.SupplierOperatorAddress)
		// Using the probabilistic proofs approach, determine if this expiring
		// claim required an on-chain proof
		proofRequirement, err = k.proofKeeper.ProofRequirementForClaim(ctx, &claim)
		if err != nil {
			return settledResult, expiredResult, err
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

		proofIsRequired := (proofRequirement != prooftypes.ProofRequirementReason_NOT_REQUIRED)
		if proofIsRequired {
			expirationReason := tokenomicstypes.ClaimExpirationReason_EXPIRATION_REASON_UNSPECIFIED // EXPIRATION_REASON_UNSPECIFIED is the default

			if isProofFound {
				if err = k.proofKeeper.EnsureValidProof(ctx, &proof); err != nil {
					logger.Warn(fmt.Sprintf("Proof was found but is invalid due to %v", err))
					expirationReason = tokenomicstypes.ClaimExpirationReason_PROOF_INVALID
				}
			} else {
				expirationReason = tokenomicstypes.ClaimExpirationReason_PROOF_MISSING
			}

			// If the proof is missing or invalid -> expire it
			if expirationReason != tokenomicstypes.ClaimExpirationReason_EXPIRATION_REASON_UNSPECIFIED {
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
					return settledResult, expiredResult, err
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
				supplierToExpiredClaimCount[claim.SupplierOperatorAddress]++

				// The claim & proof are no longer necessary, so there's no need for them
				// to take up on-chain space.
				k.proofKeeper.RemoveClaim(ctx, sessionId, claim.SupplierOperatorAddress)
				if isProofFound {
					k.proofKeeper.RemoveProof(ctx, sessionId, claim.SupplierOperatorAddress)
				}

				expiredResult.NumClaims++
				expiredResult.NumRelays += numClaimRelays
				expiredResult.NumComputeUnits += numClaimComputeUnits

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
		if err = k.ProcessTokenLogicModules(ctx, &claim, applicationInitialStake); err != nil {
			logger.Error(fmt.Sprintf("error processing token logic modules for claim %q: %v", claim.SessionHeader.SessionId, err))
			return settledResult, expiredResult, err
		}

		claimSettledEvent := tokenomicstypes.EventClaimSettled{
			Claim:                    &claim,
			NumRelays:                numClaimRelays,
			NumClaimedComputeUnits:   numClaimComputeUnits,
			NumEstimatedComputeUnits: numEstimatedComputeUnits,
			ClaimedUpokt:             &claimeduPOKT,
			ProofRequirement:         proofRequirement,
		}

		if err = ctx.EventManager().EmitTypedEvent(&claimSettledEvent); err != nil {
			return settledResult, expiredResult, err
		}

		if err = ctx.EventManager().EmitTypedEvent(&prooftypes.EventProofUpdated{
			Claim:                    &claim,
			Proof:                    nil,
			NumRelays:                0,
			NumClaimedComputeUnits:   0,
			NumEstimatedComputeUnits: numEstimatedComputeUnits,
			ClaimedUpokt:             &claimeduPOKT,
		}); err != nil {
			return settledResult, expiredResult, err
		}

		logger.Info("claim settled")

		// The claim & proof are no longer necessary, so there's no need for them
		// to take up on-chain space.
		k.proofKeeper.RemoveClaim(ctx, sessionId, claim.SupplierOperatorAddress)
		// Whether or not the proof is required, the supplier may have submitted one
		// so we need to delete it either way. If we don't have the if structure,
		// a safe error will be printed, but it can be confusing to the operator
		// or developer.
		if isProofFound {
			k.proofKeeper.RemoveProof(ctx, sessionId, claim.SupplierOperatorAddress)
		}

		settledResult.NumClaims++
		settledResult.NumRelays += numClaimRelays
		settledResult.NumComputeUnits += numClaimComputeUnits
		settledResult.RelaysPerServiceMap[claim.SessionHeader.ServiceId] += numClaimRelays

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

	// Slash all the suppliers that have been marked for slashing slashingCount times.
	for supplierOperatorAddress, slashingCount := range supplierToExpiredClaimCount {
		if err := k.slashSupplierStake(ctx, supplierOperatorAddress, slashingCount); err != nil {
			logger.Error(fmt.Sprintf("error slashing supplier %s: %s", supplierOperatorAddress, err))
			return settledResult, expiredResult, err
		}
	}

	logger.Info(fmt.Sprintf(
		"settled %d and expired %d claims at block height %d",
		settledResult.NumClaims,
		expiredResult.NumClaims,
		blockHeight,
	))

	return settledResult, expiredResult, nil
}

// getExpiringClaims returns all claims that are expiring at the current block height.
// This is the height at which the proof window closes.
// If the proof window closes and a proof IS NOT required -> settle the claim.
// If the proof window closes and a proof IS required -> only settle it if a proof is available.
func (k Keeper) getExpiringClaims(ctx cosmostypes.Context) (expiringClaims []prooftypes.Claim, err error) {
	// TODO_IMPROVE(@bryanchriswhite):
	//   1. Move height logic up to SettlePendingClaims.
	//   2. Ensure that claims are only settled or expired on a session end height.
	//     2a. This likely also requires adding validation to the shared module params.
	blockHeight := ctx.BlockHeight()

	// NB: This error can be safely ignored as on-chain SharedQueryClient implementation cannot return an error.
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
	supplierOperatorAddress string,
	slashingCount uint64,
) error {
	logger := k.logger.With("method", "slashSupplierStake")

	proofParams := k.proofKeeper.GetParams(ctx)
	slashingPenaltyPerExpiredClaim := proofParams.GetProofMissingPenalty()

	totalSlashingAmt := slashingPenaltyPerExpiredClaim.Amount.Mul(math.NewIntFromUint64(slashingCount))
	totalSlashingCoin := cosmostypes.NewCoin(volatile.DenomuPOKT, totalSlashingAmt)

	supplierToSlash, isSupplierFound := k.supplierKeeper.GetSupplier(ctx, supplierOperatorAddress)
	if !isSupplierFound {
		return tokenomicstypes.ErrTokenomicsSupplierNotFound.Wrapf(
			"cannot slash supplier with operator address: %q",
			supplierOperatorAddress,
		)
	}

	slashedSupplierInitialStakeCoin := supplierToSlash.GetStake()

	var remainingStakeCoin cosmostypes.Coin
	if slashedSupplierInitialStakeCoin.IsGTE(totalSlashingCoin) {
		remainingStakeCoin = slashedSupplierInitialStakeCoin.Sub(totalSlashingCoin)
	} else {
		// TODO_MAINNET: Consider emitting an event for this case.
		logger.Warn(fmt.Sprintf(
			"total slashing amount (%s) is greater than supplier %q stake (%s)",
			totalSlashingCoin,
			supplierOperatorAddress,
			supplierToSlash.GetStake(),
		))

		// Set the remaining stake to 0 if the slashing amount is greater than the stake.
		remainingStakeCoin = cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(0))
		// Total slashing amount is the whole supplier's stake.
		totalSlashingCoin = cosmostypes.NewCoin(volatile.DenomuPOKT, slashedSupplierInitialStakeCoin.Amount)
	}

	// Since staking mints tokens to the supplier module account, to have a correct
	// accounting, the slashing amount needs to be sent from the supplier module
	// account to the tokenomics module account.
	if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, suppliertypes.ModuleName, tokenomicstypes.ModuleName, cosmostypes.NewCoins(totalSlashingCoin)); err != nil {
		return err
	}

	// Update telemetry information
	if totalSlashingCoin.Amount.IsInt64() {
		defer telemetry.SlashedTokensFromModule(suppliertypes.ModuleName, float32(totalSlashingCoin.Amount.Int64()))
	}

	supplierToSlash.Stake = &remainingStakeCoin

	logger.Info(fmt.Sprintf(
		"slashing supplier owner with address %q operated by %q by %s, remaining stake: %s",
		supplierToSlash.GetOwnerAddress(),
		supplierToSlash.GetOperatorAddress(),
		totalSlashingCoin,
		supplierToSlash.GetStake(),
	))

	events := make([]cosmostypes.Msg, 0)

	// Check if the supplier's stake is below the minimum and unstake it if necessary.
	// TODO_BETA(@bryanchriswhite, #612): Use minimum stake governance parameter once available.
	minSupplierStakeCoin := cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(1))
	// TODO_MAINNET(@red-0ne): SettlePendingClaims is called at the end of every block,
	// but not every block corresponds to the end of a session. This may lead to a situation
	// where a force unstaked supplier may still be able to interact with a Gateway or Application.
	// However, claims are only processed when sessions end.
	// INVESTIGATION: This requires an investigation if the race condition exists
	// at all and fixed only if it does.
	if supplierToSlash.GetStake().IsLT(minSupplierStakeCoin) {
		sharedParams := k.sharedKeeper.GetParams(ctx)
		sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
		currentHeight := sdkCtx.BlockHeight()
		unstakeSessionEndHeight := uint64(sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight))

		logger.Warn(fmt.Sprintf(
			"unstaking supplier %q owned by %q due to stake (%s) below the minimum (%s)",
			supplierToSlash.GetOperatorAddress(),
			supplierToSlash.GetOwnerAddress(),
			supplierToSlash.GetStake(),
			minSupplierStakeCoin,
		))

		// TODO_MAINNET: Should we just remove the supplier if the stake is
		// below the minimum, at the risk of making the off-chain actors have an
		// inconsistent session supplier list? See the comment above for more details.
		supplierToSlash.UnstakeSessionEndHeight = unstakeSessionEndHeight

		sessionEndHeight := sharedtypes.GetSettlementSessionEndHeight(&sharedParams, currentHeight)
		unbondingHeight := sharedtypes.GetSupplierUnbondingHeight(&sharedParams, &supplierToSlash)
		events = append(events, &suppliertypes.EventSupplierUnbondingBegin{
			Supplier:         &supplierToSlash,
			Reason:           suppliertypes.SupplierUnbondingReason_BELOW_MIN_STAKE,
			SessionEndHeight: sessionEndHeight,
			UnbondingHeight:  unbondingHeight,
		})
	}

	k.supplierKeeper.SetSupplier(ctx, supplierToSlash)

	// Emit an event that a supplier has been slashed.
	events = append(events, &tokenomicstypes.EventSupplierSlashed{
		SupplierOperatorAddr: supplierOperatorAddress,
		NumExpiredClaims:     slashingCount,
		SlashingAmount:       &totalSlashingCoin,
	})

	if err := ctx.EventManager().EmitTypedEvents(events...); err != nil {
		err = suppliertypes.ErrSupplierEmitEvent.Wrapf("(%+v): %s", events, err)
		logger.Error(err.Error())
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
