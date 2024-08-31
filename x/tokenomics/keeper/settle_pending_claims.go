package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	poktrand "github.com/pokt-network/poktroll/pkg/crypto/rand"
	"github.com/pokt-network/poktroll/telemetry"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
	tokenomictypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// SettlePendingClaims settles all pending (i.e. expiring) claims.
// If a claim is expired and requires a proof and a proof IS available -> it's settled.
// If a claim is expired and requires a proof and a proof IS NOT available -> it's deleted.
// If a claim is expired and does NOT require a proof -> it's settled.
// Events are emitted for each claim that is settled or removed.
// On-chain Claims & Proofs are deleted after they're settled or expired to free up space.
//
// TODO_TECHDEBT: Refactor this function to return a struct instead of multiple return values.
func (k Keeper) SettlePendingClaims(ctx sdk.Context) (
	settledResult types.PendingClaimsResult,
	expiredResult types.PendingClaimsResult,
	err error,
) {
	logger := k.Logger().With("method", "SettlePendingClaims")

	expiringClaims, err := k.getExpiringClaims(ctx)
	if err != nil {
		return settledResult, expiredResult, err
	}

	blockHeight := ctx.BlockHeight()

	logger.Info(fmt.Sprintf("found %d expiring claims at block height %d", len(expiringClaims), blockHeight))

	// Initialize results structs.
	settledResult = types.NewClaimSettlementResult()
	expiredResult = types.NewClaimSettlementResult()

	logger.Debug("settling expiring claims")
	for _, claim := range expiringClaims {
		var (
			proofRequirement              prooftypes.ProofRequirementReason
			numClaimRelays                uint64
			numClaimComputeUnits          uint64
			numClaimEstimatedComputeUnits uint64
		)

		// NB: Not every (Req, Res) pair in the session is inserted into the tree due
		// to the relay mining difficulty. This is the count of non-empty leaves that
		// matched the necessary difficulty and is therefore an estimation of the total
		// number of relays serviced and work done.
		numClaimRelays, err = claim.GetNumRelays()
		if err != nil {
			return settledResult, expiredResult, err
		}

		// Retrieve the service to which the claim is associated.
		service, serviceFound := k.serviceKeeper.GetService(ctx, claim.SessionHeader.Service.Id)
		if !serviceFound {
			return settledResult, expiredResult, tokenomictypes.ErrTokenomicsServiceNotFound.Wrapf("service with ID %q not found", claim.SessionHeader.Service.Id)
		}

		numComputeUnits := k.claimedToEstimatedComputeUnits(ctx, numClaimRelays)

		sessionId := claim.SessionHeader.SessionId

		proof, isProofFound := k.proofKeeper.GetProof(ctx, sessionId, claim.SupplierOperatorAddress)
		// Using the probabilistic proofs approach, determine if this expiring
		// claim required an on-chain proof
		proofRequirement, err = k.proofRequirementForClaim(ctx, claim)
		if err != nil {
			return settledResult, expiredResult, err
		}

		logger = k.logger.With(
			"session_id", sessionId,
			"supplier_operator_address", claim.SupplierOperatorAddress,
			"num_claim_compute_units", numClaimEstimatedComputeUnits,
			"num_relays_in_session_tree", numClaimRelays,
			"proof_requirement", proofRequirement,
		)

		proofIsRequired := (proofRequirement != prooftypes.ProofRequirementReason_NOT_REQUIRED)
		if proofIsRequired {
			expirationReason := types.ClaimExpirationReason_EXPIRATION_REASON_UNSPECIFIED // EXPIRATION_REASON_UNSPECIFIED is the default

			if isProofFound {
				if err = k.proofKeeper.EnsureValidProof(ctx, &proof); err != nil {
					logger.Warn(fmt.Sprintf("Proof was found but is invalid due to %v", err))
					expirationReason = types.ClaimExpirationReason_PROOF_INVALID
				}
			} else {
				expirationReason = types.ClaimExpirationReason_PROOF_MISSING
			}

			// If the proof is missing or invalid -> expire it
			if expirationReason != types.ClaimExpirationReason_EXPIRATION_REASON_UNSPECIFIED {
				// TODO_BETA(@red-0ne, @olshansk): Slash the supplier in proportion
				// to their stake. Consider allowing suppliers to RemoveClaim via a new
				// message in case it was sent by accident

				// Proof was required but not found.
				// Emit an event that a claim has expired and being removed without being settled.
				claimExpiredEvent := types.EventClaimExpired{
					Claim:            &claim,
					ExpirationReason: expirationReason,
					NumRelays:        numClaimRelays,
					NumComputeUnits:  numClaimedComputeUnits,
				}
				if err = ctx.EventManager().EmitTypedEvent(&claimExpiredEvent); err != nil {
					return settledResult, expiredResult, err
				}

				logger.Info("claim expired; required proof not found")

				// The claim & proof are no longer necessary, so there's no need for them
				// to take up on-chain space.
				k.proofKeeper.RemoveClaim(ctx, sessionId, claim.SupplierOperatorAddress)
				if isProofFound {
					k.proofKeeper.RemoveProof(ctx, sessionId, claim.SupplierOperatorAddress)
				}

				expiredResult.NumClaims++
				expiredResult.NumRelays += numClaimRelays
				expiredResult.NumComputeUnits += numClaimedComputeUnits
				continue
			}
		}

		// If this code path is reached, then either:
		// 1. The claim does not require a proof.
		// 2. The claim requires a proof and a valid proof was found.

		// Manage the mint & burn accounting for the claim.
		if err = k.ProcessTokenLogicModules(ctx, &claim); err != nil {
			logger.Error(fmt.Sprintf("error processing token logic modules for claim %q: %v", claim.SessionHeader.SessionId, err))
			return settledResult, expiredResult, err
		}

		claimSettledEvent := types.EventClaimSettled{
			Claim:            &claim,
			NumRelays:        numClaimRelays,
			NumComputeUnits:  numClaimedComputeUnits,
			ProofRequirement: proofRequirement,
		}

		if err = ctx.EventManager().EmitTypedEvent(&claimSettledEvent); err != nil {
			return settledResult, expiredResult, err
		}

		if err = ctx.EventManager().EmitTypedEvent(&prooftypes.EventProofUpdated{
			Claim:           &claim,
			Proof:           nil,
			NumRelays:       0,
			NumComputeUnits: 0,
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
		settledResult.NumComputeUnits += numClaimedComputeUnits
		settledResult.RelaysPerServiceMap[claim.SessionHeader.Service.Id] += numClaimRelays

		logger.Info(fmt.Sprintf("Successfully settled claim for session ID %q at block height %d", claim.SessionHeader.SessionId, blockHeight))
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
func (k Keeper) getExpiringClaims(ctx sdk.Context) (expiringClaims []prooftypes.Claim, err error) {
	blockHeight := ctx.BlockHeight()

	// NB: This error can be safely ignored as on-chain SharedQueryClient implementation cannot return an error.
	sharedParams, _ := k.sharedQuerier.GetParams(ctx)
	claimWindowSizeBlocks := sharedParams.GetClaimWindowOpenOffsetBlocks() + sharedParams.GetClaimWindowCloseOffsetBlocks()
	proofWindowSizeBlocks := sharedParams.GetProofWindowOpenOffsetBlocks() + sharedParams.GetProofWindowCloseOffsetBlocks()

	// expiringSessionEndHeight is the session end height of the session whose proof
	// window has most recently closed.
	expiringSessionEndHeight := blockHeight -
		int64(claimWindowSizeBlocks+
			proofWindowSizeBlocks+1)

	// DEV_NOTE: This is being kept in for an easy uncomment plus debugging purposes.
	// allClaims := k.proofKeeper.GetAllClaims(ctx)
	// _ = allClaims

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

// proofRequirementForClaim checks if a proof is required for a claim.
// If it is not, the claim will be settled without a proof.
// If it is, the claim will only be settled if a valid proof is available.
// TODO_BLOCKER(@bryanchriswhite, #419): Document safety assumptions of the probabilistic proofs mechanism.
func (k Keeper) proofRequirementForClaim(ctx sdk.Context, claim prooftypes.Claim) (_ prooftypes.ProofRequirementReason, err error) {
	logger := k.logger.With("method", "proofRequirementForClaim")

	// Defer telemetry calls so that they reference the final values the relevant variables.
	var requirementReason = prooftypes.ProofRequirementReason_NOT_REQUIRED
	defer func() {
		telemetry.ProofRequirementCounter(requirementReason, err)
	}()

	// Get the number of claimed compute units in the claim.
	numClaimComputeUnits, err := claim.GetNumComputeUnits()
	if err != nil {
		return requirementReason, err
	}

	// Get the number of claimed compute units in the claim.
	numRelays, err := claim.GetNumRelays()
	if err != nil {
		return requirementReason, err
	}

	numEstimatedComputeUnits := claimedToEstimatedComputeUnits(ctx, num)

	// Retrieve the relay mining difficulty for the claim's service to determine
	// the estimated number of compute units handled off-chain.
	relayMiningDifficulty, found := k.GetRelayMiningDifficulty(ctx, claim.SessionHeader.Service.Id)
	if !found {
		var numRelays uint64
		numRelays, err = claim.GetNumRelays()
		if err != nil {
			return requirementReason, err
		}
		serviceId := claim.GetSessionHeader().GetService().GetId()
		relayMiningDifficulty = newDefaultRelayMiningDifficulty(ctx, logger, serviceId, numRelays)
	}

	// The number of estimated compute unites is a multiplier of the number of
	// claimed compute units since it depends on whether the relay is minable or not.
	difficultyMultiplier := protocol.GetDifficultyFromHash([32]byte(relayMiningDifficulty.TargetHash))
	numEstimatedComputeUnits := numClaimComputeUnits * uint64(difficultyMultiplier)

	logger.Info(fmt.Sprintf("Estimated (%d) serviced compute units from (%d) claimed compute units"+
		"with a difficulty multiplier of (%d) for relay difficulty (%v)", numEstimatedComputeUnits, numClaimComputeUnits, difficultyMultiplier, relayMiningDifficulty.TargetHash))

	proofParams := k.proofKeeper.GetParams(ctx)

	// TODO_BETA(@olshansk): Evaluate how the proof requirement threshold should
	// be a function of the stake.

	// Require a proof if the claim's compute units meets or exceeds the threshold.//
	// TODO_IMPROVE(@red-0ne): It might make sense to include
	// whether there was a proof submission error downstream from here. This would
	// require a more comprehensive metrics API.
	if numEstimatedComputeUnits >= proofParams.GetProofRequirementThreshold() {
		requirementReason = prooftypes.ProofRequirementReason_THRESHOLD

		logger.Info(fmt.Sprintf(
			"claim requires proof due to estimated serviced compute units (%d) exceeding threshold (%d)",
			numEstimatedComputeUnits,
			proofParams.GetProofRequirementThreshold(),
		))
		return requirementReason, nil
	}

	// Get the hash of the claim to seed the random number generator.
	var claimHash []byte
	claimHash, err = claim.GetHash()
	if err != nil {
		return requirementReason, err
	}

	// Sample a pseudo-random value between 0 and 1 to determine if a proof is required probabilistically.
	// TODO_BETA(@red-0ne): Ensure that the randomness is seeded by values after the
	// claim window is closed.
	var randFloat float32
	randFloat, err = poktrand.SeededFloat32(claimHash[:])
	if err != nil {
		return requirementReason, err
	}

	// Require a proof probabilistically based on the proof_request_probability param.
	// NB: A random value between 0 and 1 will be less than or equal to proof_request_probability
	// with probability equal to the proof_request_probability.
	if randFloat <= proofParams.GetProofRequestProbability() {
		requirementReason = prooftypes.ProofRequirementReason_PROBABILISTIC

		logger.Info(fmt.Sprintf(
			"claim requires proof due to random sample (%.2f) being less than or equal to probability (%.2f)",
			randFloat,
			proofParams.GetProofRequestProbability(),
		))
		return requirementReason, nil
	}

	logger.Info(fmt.Sprintf(
		"claim does not require proof due to compute units (%d) being less than the threshold (%d) and random sample (%.2f) being greater than probability (%.2f)",
		numClaimComputeUnits,
		proofParams.GetProofRequirementThreshold(),
		randFloat,
		proofParams.GetProofRequestProbability(),
	))
	return requirementReason, nil
}
