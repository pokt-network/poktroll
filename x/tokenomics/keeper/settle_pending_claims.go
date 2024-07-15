package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	poktrand "github.com/pokt-network/poktroll/pkg/crypto/rand"
	"github.com/pokt-network/poktroll/telemetry"
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
	numClaimsSettled, numClaimsExpired uint64,
	relaysPerServiceMap map[string]uint64,
	computeUnitsPerServiceMap map[string]uint64,
	err error,
) {
	logger := k.Logger().With("method", "SettlePendingClaims")

	expiringClaims, err := k.getExpiringClaims(ctx)
	if err != nil {
		return 0, 0, relaysPerServiceMap, computeUnitsPerServiceMap, err
	}

	blockHeight := ctx.BlockHeight()

	logger.Info(fmt.Sprintf("found %d expiring claims at block height %d", len(expiringClaims), blockHeight))

	relaysPerServiceMap = make(map[string]uint64)
	computeUnitsPerServiceMap = make(map[string]uint64)

	logger.Debug("settling expiring claims")
	for _, claim := range expiringClaims {
		var (
			numClaimComputeUnits   uint64
			numRelaysInSessionTree uint64
			proofRequirement       prooftypes.ProofRequirementReason
		)

		// NB: Note that not every (Req, Res) pair in the session is inserted in
		// the tree for scalability reasons. This is the count of non-empty leaves
		// that matched the necessary difficulty and is therefore an estimation
		// of the total number of relays serviced and work done.
		numClaimComputeUnits, err = claim.GetNumComputeUnits()
		if err != nil {
			return 0, 0, relaysPerServiceMap, computeUnitsPerServiceMap, err
		}

		numRelaysInSessionTree, err = claim.GetNumRelays()
		if err != nil {
			return 0, 0, relaysPerServiceMap, computeUnitsPerServiceMap, err
		}

		sessionId := claim.SessionHeader.SessionId

		_, isProofFound := k.proofKeeper.GetProof(ctx, sessionId, claim.SupplierAddress)
		// Using the probabilistic proofs approach, determine if this expiring
		// claim required an on-chain proof
		proofRequirement, err = k.proofRequirementForClaim(ctx, &claim)
		if err != nil {
			return 0, 0, relaysPerServiceMap, computeUnitsPerServiceMap, err
		}

		logger := k.logger.With(
			"session_id", sessionId,
			"supplier_address", claim.SupplierAddress,
			"num_claim_compute_units", numClaimComputeUnits,
			"num_relays_in_session_tree", numRelaysInSessionTree,
			"proof_requirement", proofRequirement,
		)

		if proofRequirement != prooftypes.ProofRequirementReason_NOT_REQUIRED {
			// If a proof is not found, the claim will expire and never be settled.
			if !isProofFound {
				// Emit an event that a claim has expired and being removed without being settled.
				claimExpiredEvent := types.EventClaimExpired{
					Claim:           &claim,
					NumComputeUnits: numClaimComputeUnits,
					NumRelays:       numRelaysInSessionTree,
				}
				if err = ctx.EventManager().EmitTypedEvent(&claimExpiredEvent); err != nil {
					return 0, 0, relaysPerServiceMap, computeUnitsPerServiceMap, err
				}

				logger.Info("claim expired; required proof not found")

				// The claim & proof are no longer necessary, so there's no need for them
				// to take up on-chain space.
				k.proofKeeper.RemoveClaim(ctx, sessionId, claim.SupplierAddress)

				numClaimsExpired++
				continue
			}
			// NB: If a proof is found, it is valid because verification is done
			// at the time of submission.
		}

		// Manage the mint & burn accounting for the claim.
		if err = k.SettleSessionAccounting(ctx, &claim); err != nil {
			logger.Error(fmt.Sprintf("error settling session accounting for claim %q: %v", claim.SessionHeader.SessionId, err))
			return 0, 0, relaysPerServiceMap, computeUnitsPerServiceMap, err
		}

		claimSettledEvent := types.EventClaimSettled{
			Claim:            &claim,
			NumRelays:        numRelaysInSessionTree,
			NumComputeUnits:  numClaimComputeUnits,
			ProofRequirement: proofRequirement,
		}

		if err = ctx.EventManager().EmitTypedEvent(&claimSettledEvent); err != nil {
			return 0, 0, relaysPerServiceMap, computeUnitsPerServiceMap, err
		}

		if err = ctx.EventManager().EmitTypedEvent(&prooftypes.EventProofUpdated{
			Claim:           &claim,
			Proof:           nil,
			NumRelays:       0,
			NumComputeUnits: 0,
		}); err != nil {
			return 0, 0, relaysPerServiceMap, computeUnitsPerServiceMap, err
		}

		logger.Info("claim settled")

		// The claim & proof are no longer necessary, so there's no need for them
		// to take up on-chain space.
		k.proofKeeper.RemoveClaim(ctx, sessionId, claim.SupplierAddress)
		// Whether or not the proof is required, the supplier may have submitted one
		// so we need to delete it either way. If we don't have the if structure,
		// a safe error will be printed, but it can be confusing to the operator
		// or developer.
		if isProofFound {
			k.proofKeeper.RemoveProof(ctx, sessionId, claim.SupplierAddress)
		}

		relaysPerServiceMap[claim.SessionHeader.Service.Id] += numRelaysInSessionTree
		computeUnitsPerServiceMap[claim.SessionHeader.Service.Id] += numClaimComputeUnits

		numClaimsSettled++
		logger.Info(fmt.Sprintf("Successfully settled claim for session ID %q at block height %d", claim.SessionHeader.SessionId, blockHeight))
	}

	logger.Info(fmt.Sprintf("settled %d and expired %d claims at block height %d", numClaimsSettled, numClaimsExpired, blockHeight))

	return numClaimsSettled, numClaimsExpired, relaysPerServiceMap, computeUnitsPerServiceMap, nil
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

		for _, claim := range claimsRes.GetClaims() {
			expiringClaims = append(expiringClaims, claim)
		}

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
func (k Keeper) proofRequirementForClaim(ctx sdk.Context, claim *prooftypes.Claim) (_ prooftypes.ProofRequirementReason, err error) {
	logger := k.logger.With("method", "proofRequirementForClaim")

	var requirementReason = prooftypes.ProofRequirementReason_NOT_REQUIRED

	// Defer telemetry calls so that they reference the final values the relevant variables.
	defer func() {
		telemetry.ProofRequirementCounter(requirementReason, err)
	}()

	// NB: Assumption that claim is non-nil and has a valid root sum because it
	// is retrieved from the store and validated, on-chain, at time of creation.
	var numClaimComputeUnits uint64
	numClaimComputeUnits, err = claim.GetNumComputeUnits()
	if err != nil {
		return requirementReason, err
	}

	proofParams := k.proofKeeper.GetParams(ctx)

	// Require a proof if the claim's compute units meets or exceeds the threshold.
	//
	// TODO_BLOCKER(@bryanchriswhite, #419): This is just VERY BASIC placeholder logic to have something
	// in place while we implement proper probabilistic proofs. If you're reading it,
	// do not overthink it and look at the documents linked in #419.
	//
	// TODO_IMPROVE(@bryanchriswhite, @red-0ne): It might make sense to include
	// whether there was a proof submission error downstream from here. This would
	// require a more comprehensive metrics API.
	if numClaimComputeUnits >= proofParams.GetProofRequirementThreshold() {
		requirementReason = prooftypes.ProofRequirementReason_THRESHOLD

		logger.Info(fmt.Sprintf(
			"claim requires proof due to compute units (%d) exceeding threshold (%d)",
			numClaimComputeUnits,
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
