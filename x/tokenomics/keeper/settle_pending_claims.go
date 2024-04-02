package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"

	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

const (
	// TODO_BLOCKER/TODO_UPNEXT(@Olshansk): Implement this properly. Using a constant
	// for "probabilistic proofs" is just a simple placeholder mechanism to get
	// #359 over the finish line.
	ProofRequiredComputeUnits = 100
)

// SettlePendingClaims settles all pending (i.e. expiring) claims.
// If a claim is expired and requires a proof and a proof IS available -> it's settled.
// If a claim is expired and requires a proof and a proof IS NOT available -> it's deleted.
// If a claim is expired and does NOT require a proof -> it's settled.
// Events are emitted for each claim that is settled or removed.
// On-chain Claims & Proofs are deleted after they're settled or expired to free up space.
func (k Keeper) SettlePendingClaims(ctx sdk.Context) (numClaimsSettled, numClaimsExpired uint64, err error) {
	logger := k.Logger().With("method", "SettlePendingClaims")

	// TODO_BLOCKER(@Olshansk): Optimize this by indexing expiringClaims appropriately
	// and only retrieving the expiringClaims that need to be settled rather than all
	// of them and iterating through them one by one.
	expiringClaims, err := k.getExpiringClaims(ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("error getting expiring claims: %v", err))
		return 0, 0, err
	}

	blockHeight := ctx.BlockHeight()

	logger.Info(fmt.Sprintf("found %d expiring claims at block height %d", len(expiringClaims), blockHeight))

	for _, claim := range expiringClaims {
		// Retrieve the number of compute units in the claim for the events emitted
		root := (smt.MerkleRoot)(claim.GetRootHash())
		claimComputeUnits := root.Sum()

		sessionId := claim.SessionHeader.SessionId

		// Using the probabilistic proofs approach, determine if this expiring
		// claim required an on-chain proof
		isProofRequiredForClaim, err := k.isProofRequiredForClaim(ctx, &claim)
		if err != nil {
			logger.Error(fmt.Sprintf("error checking if proof is required for claim %s: %v", sessionId, err))
			return 0, 0, err
		}
		if isProofRequiredForClaim {
			_, isProofFound := k.proofKeeper.GetProof(ctx, sessionId, claim.SupplierAddress)
			// If a proof is not found, the claim will expire and never be settled.
			if !isProofFound {
				// Emit an event that a claim has expired and being removed without being settled.
				claimExpiredEvent := types.EventClaimExpired{
					Claim:        &claim,
					ComputeUnits: claimComputeUnits,
				}
				if err := ctx.EventManager().EmitTypedEvent(&claimExpiredEvent); err != nil {
					return 0, 0, err
				}
				// The claim & proof are no longer necessary, so there's no need for them
				// to take up on-chain space.
				k.proofKeeper.RemoveClaim(ctx, sessionId, claim.SupplierAddress)
				continue
			}
			// NB: If a proof is found, it is valid because verification is done
			// at the time of submission.
		}

		// Manage the mint & burn accounting for the claim.
		if err := k.SettleSessionAccounting(ctx, &claim); err != nil {
			logger.Error(fmt.Sprintf("error settling session accounting for claim %s: %v", claim.SessionHeader.SessionId, err))
			return 0, 0, err
		}

		claimSettledEvent := types.EventClaimSettled{
			Claim:         &claim,
			ComputeUnits:  claimComputeUnits,
			ProofRequired: isProofRequiredForClaim,
		}
		if err := ctx.EventManager().EmitTypedEvent(&claimSettledEvent); err != nil {
			return 0, 0, err
		}

		// The claim & proof are no longer necessary, so there's no need for them
		// to take up on-chain space.
		k.proofKeeper.RemoveClaim(ctx, sessionId, claim.SupplierAddress)
		k.proofKeeper.RemoveProof(ctx, sessionId, claim.SupplierAddress)

		numClaimsSettled++
		logger.Info(fmt.Sprintf("Successfully settled claim %s at block height %d", claim.SessionHeader.SessionId, blockHeight))
	}

	logger.Info(fmt.Sprintf("settled %d claims and expired %d at block height %d", numClaimsSettled, numClaimsExpired, blockHeight))

	return numClaimsSettled, numClaimsExpired, nil

}

// getExpiringClaims returns all claims that are expiring at the current block height.
// This is the height at which the proof window closes.
// If the proof window closes and a proof IS NOT required -> settle the claim.
// If the proof window closes and a proof IS required -> only settle it if a proof is available.
func (k Keeper) getExpiringClaims(ctx sdk.Context) (expiringClaims []prooftypes.Claim, err error) {
	blockHeight := ctx.BlockHeight()

	// TODO_BLOCKER: query the on-chain governance parameter once available.
	// `* 3` is just a random factor Olshansky added for now to make sure expiration
	// doesn't happen immediately after a session's grace period is complete.
	submitProofWindowEndHeight := sessionkeeper.GetSessionGracePeriodBlockCount() * 3

	// TODO_BLOCKER(@Olshansk): Optimize this by indexing claims appropriately
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
	return expiringClaims, nil
}

// isProofRequiredForClaim checks if a proof is required for a claim.
// If it is not, the claim will be settled without a proof.
// If it is, the claim will only be settled if a valid proof is available.
// TODO_TECHDEBT(#419): Document safety assumptions of the probabilistic proofs mechanism.
func (k Keeper) isProofRequiredForClaim(_ sdk.Context, claim *prooftypes.Claim) (bool, error) {
	// NB: Assumption that claim is non-nil and has a valid root sum because it
	// is retrieved from the store.
	root := (smt.MerkleRoot)(claim.GetRootHash())
	claimComputeUnits := root.Sum()
	if claimComputeUnits < ProofRequiredComputeUnits {
		return false, nil
	}
	return true, nil
}
