package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"

	"github.com/pokt-network/poktroll/telemetry"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	"github.com/pokt-network/poktroll/x/shared"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// SettlePendingClaims settles all pending (i.e. expiring) claims.
// If a claim is expired and requires a proof and a proof IS available -> it's settled.
// If a claim is expired and requires a proof and a proof IS NOT available -> it's deleted.
// If a claim is expired and does NOT require a proof -> it's settled.
// Events are emitted for each claim that is settled or removed.
// On-chain Claims & Proofs are deleted after they're settled or expired to free up space.
func (k Keeper) SettlePendingClaims(ctx sdk.Context) (
	numClaimsSettled, numClaimsExpired uint64,
	relaysPerServiceMap map[string]uint64,
	err error,
) {
	logger := k.Logger().With("method", "SettlePendingClaims")

	isSuccessful := false
	defer telemetry.EventSuccessCounter(
		"claims_settled",
		func() float32 { return float32(numClaimsSettled) },
		func() bool { return isSuccessful },
	)

	defer telemetry.EventSuccessCounter(
		"claims_expired",
		func() float32 { return float32(numClaimsExpired) },
		func() bool { return isSuccessful },
	)

	// TODO_BLOCKER(@Olshansk): Optimize this by indexing expiringClaims appropriately
	// and only retrieving the expiringClaims that need to be settled rather than all
	// of them and iterating through them one by one.
	expiringClaims := k.getExpiringClaims(ctx)

	blockHeight := ctx.BlockHeight()

	logger.Info(fmt.Sprintf("found %d expiring claims at block height %d", len(expiringClaims), blockHeight))

	relaysPerServiceMap = make(map[string]uint64)

	for _, claim := range expiringClaims {
		// Retrieve the number of compute units in the claim for the events emitted
		root := (smt.MerkleRoot)(claim.GetRootHash())

		// NB: Note that not every (Req, Res) pair in the session is inserted in
		// the tree for scalability reasons. This is the count of non-empty leaves
		// that matched the necessary difficulty and is therefore an estimation
		// of the total number of relays serviced and work done.
		claimComputeUnits := root.Sum()
		numRelaysInSessionTree := root.Count()

		sessionId := claim.SessionHeader.SessionId

		_, isProofFound := k.proofKeeper.GetProof(ctx, sessionId, claim.SupplierAddress)
		// Using the probabilistic proofs approach, determine if this expiring
		// claim required an on-chain proof
		isProofRequiredForClaim := k.isProofRequiredForClaim(ctx, &claim)
		if isProofRequiredForClaim {
			// If a proof is not found, the claim will expire and never be settled.
			if !isProofFound {
				// Emit an event that a claim has expired and being removed without being settled.
				claimExpiredEvent := types.EventClaimExpired{
					Claim:        &claim,
					ComputeUnits: claimComputeUnits,
				}
				if err := ctx.EventManager().EmitTypedEvent(&claimExpiredEvent); err != nil {
					return 0, 0, relaysPerServiceMap, err
				}
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
		if err := k.SettleSessionAccounting(ctx, &claim); err != nil {
			logger.Error(fmt.Sprintf("error settling session accounting for claim %q: %v", claim.SessionHeader.SessionId, err))
			return 0, 0, relaysPerServiceMap, err
		}

		claimSettledEvent := types.EventClaimSettled{
			Claim:         &claim,
			ComputeUnits:  claimComputeUnits,
			ProofRequired: isProofRequiredForClaim,
		}
		if err := ctx.EventManager().EmitTypedEvent(&claimSettledEvent); err != nil {
			return 0, 0, relaysPerServiceMap, err
		}

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

		numClaimsSettled++
		logger.Info(fmt.Sprintf("Successfully settled claim for session ID %q at block height %d", claim.SessionHeader.SessionId, blockHeight))
	}

	logger.Info(fmt.Sprintf("settled %d and expired %d claims at block height %d", numClaimsSettled, numClaimsExpired, blockHeight))

	isSuccessful = true
	return numClaimsSettled, numClaimsExpired, relaysPerServiceMap, nil
}

// getExpiringClaims returns all claims that are expiring at the current block height.
// This is the height at which the proof window closes.
// If the proof window closes and a proof IS NOT required -> settle the claim.
// If the proof window closes and a proof IS required -> only settle it if a proof is available.
func (k Keeper) getExpiringClaims(ctx sdk.Context) (expiringClaims []prooftypes.Claim) {
	blockHeight := ctx.BlockHeight()

	// TODO_BLOCKER(@bryanchriswhite): query the on-chain governance parameter once available.
	// `* 3` is just a random factor Olshansky added for now to make sure expiration
	// doesn't happen immediately after a session's grace period is complete.
	submitProofWindowEndHeight := shared.SessionGracePeriodBlocks * int64(3)

	// TODO_TECHDEBT: Optimize this by indexing claims appropriately
	// and only retrieving the claims that need to be settled rather than all
	// of them and iterating through them one by one.
	claims := k.proofKeeper.GetAllClaims(ctx)

	// Loop over all claims we need to check for expiration
	for _, claim := range claims {
		expirationHeight := claim.SessionHeader.SessionEndBlockHeight + submitProofWindowEndHeight
		if blockHeight >= expirationHeight {
			expiringClaims = append(expiringClaims, claim)
		}
	}

	// Return the actually expiring claims
	return expiringClaims
}

// isProofRequiredForClaim checks if a proof is required for a claim.
// If it is not, the claim will be settled without a proof.
// If it is, the claim will only be settled if a valid proof is available.
// TODO_BLOCKER(@bryanchriswhite, #419): Document safety assumptions of the probabilistic proofs mechanism.
func (k Keeper) isProofRequiredForClaim(ctx sdk.Context, claim *prooftypes.Claim) bool {
	// NB: Assumption that claim is non-nil and has a valid root sum because it
	// is retrieved from the store and validated, on-chain, at time of creation.
	root := (smt.MerkleRoot)(claim.GetRootHash())
	claimComputeUnits := root.Sum()
	// TODO_BLOCKER(@bryanchriswhite, #419): This is just VERY BASIC placeholder logic to have something
	// in place while we implement proper probabilistic proofs. If you're reading it,
	// do not overthink it and look at the documents linked in #419.
	if claimComputeUnits < k.proofKeeper.GetParams(ctx).ProofRequirementThreshold {
		return false
	}
	return true
}
