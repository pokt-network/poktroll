package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
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
			numClaimComputeUnits uint64
			numClaimRelays       uint64
			proofRequirement     prooftypes.ProofRequirementReason
		)

		// NB: Note that not every (Req, Res) pair in the session is inserted in
		// the tree for scalability reasons. This is the count of non-empty leaves
		// that matched the necessary difficulty and is therefore an estimation
		// of the total number of relays serviced and work done.
		numClaimComputeUnits, err = claim.GetNumComputeUnits()
		if err != nil {
			return settledResult, expiredResult, err
		}

		numClaimRelays, err = claim.GetNumRelays()
		if err != nil {
			return settledResult, expiredResult, err
		}

		sessionId := claim.SessionHeader.SessionId

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
				// Proof was required but not found.
				// Emit an event that a claim has expired and being removed without being settled.
				claimExpiredEvent := types.EventClaimExpired{
					Claim:            &claim,
					NumComputeUnits:  numClaimComputeUnits,
					NumRelays:        numClaimRelays,
					ExpirationReason: expirationReason,
					// TODO_CONSIDERATION: Add the error to the event if the proof was invalid.
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
				expiredResult.NumComputeUnits += numClaimComputeUnits
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
			NumComputeUnits:  numClaimComputeUnits,
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
		settledResult.NumComputeUnits += numClaimComputeUnits
		settledResult.RelaysPerServiceMap[claim.SessionHeader.ServiceId] += numClaimRelays

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

	allClaims := k.proofKeeper.GetAllClaims(ctx)
	_ = allClaims

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
