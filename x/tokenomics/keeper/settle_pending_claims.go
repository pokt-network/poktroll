package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

const ()

// SettleExpiringClaims settles all pending claims.
func (k Keeper) SettleExpiringClaims(ctx sdk.Context) error {
	logger := k.Logger().With("method", "SettleExpiringClaims")

	// TODO_BLOCKER(@Olshansk): Optimize this by indexing expiringClaims appropriately
	// and only retrieving the expiringClaims that need to be settled rather than all
	// of them and iterating through them one by one.
	expiringClaims, err := k.getExpiringClaims(ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("error getting expiring claims: %v", err))
		return err
	}

	blockHeight := ctx.BlockHeight()

	numClaimsSettled := 0
	for _, claim := range expiringClaims {
		isProofRequiredForClaim, err := k.isProofRequiredForClaim(ctx)
		if err != nil {
			logger.Error(fmt.Sprintf("error checking if proof is required for claim %s: %v", claim.SessionHeader.SessionId, err))
			return err
		}

		// Using the probabilistic proofs approach, determine if this expiring
		// claim required an on-chain proof
		if isProofRequiredForClaim {
			_, isProofFound := k.proofKeeper.GetProof(ctx, claim.SessionHeader.SessionId, claim.SupplierAddress)
			if err != nil {
				logger.Error(fmt.Sprintf("error getting proof for claim %s: %v", claim.SessionHeader.SessionId, err))
				return err
			}
			// If a proof is not found, the claim will expire and never be settled.
			if !isProofFound {
				claimExpiredEvent := types.EventClaimExpired{
					SupplierAddress:         claim.SupplierAddress,
					ApplicationAddress:      claim.SessionHeader.ApplicationAddress,
					SessionStartBlockHeight: uint64(claim.SessionHeader.SessionStartBlockHeight),
					ServiceId:               claim.SessionHeader.Service.Id,
				}
				if err := ctx.EventManager().EmitTypedEvent(&claimExpiredEvent); err != nil {
					return err
				}

				continue
			}
			// If a proof is found, it is valid because verification is done
			// at the time of submission.
		}

		// Manage the mint & burn accounting for the claim.
		if err := k.SettleSessionAccounting(ctx, &claim); err != nil {
			logger.Error(fmt.Sprintf("error settling session accounting for claim %s: %v", claim.SessionHeader.SessionId, err))
			return err
		}

		// The claim & proof are no longer necessary, so there's no need for them
		// to take up on-chain space.
		// TODO_BLOCKER(@Olshansk): Decide if we should be doing this or not.
		// It could be used for data analysis and historical purposes, but not needed
		// for functionality.
		k.proofKeeper.RemoveClaim(ctx, claim.SessionHeader.SessionId, claim.SupplierAddress)
		k.proofKeeper.RemoveProof(ctx, claim.SessionHeader.SessionId, claim.SupplierAddress)

		numClaimsSettled++
		logger.Info(fmt.Sprintf("Successfully settled claim %s at block height %d", claim.SessionHeader.SessionId, blockHeight))
	}

	logger.Info(fmt.Sprintf("settled %d claims at block height %d", numClaimsSettled, blockHeight))

	return nil

}

// getExpiringClaims returns all claims that are expiring at the current block height.
// This is the height at which the proof window closes.
// If the proof window closes and a proof IS NOT required -> settle the claim.
// If the proof window closes and a proof IS required -> only settle it if a proof is available.
func (k Keeper) getExpiringClaims(ctx sdk.Context) (expiringClaims []prooftypes.Claim, err error) {

	blockHeight := ctx.BlockHeight()

	// TODO_BLOCKER: query the on-chain governance parameter once available.
	submitProofWindowEndHeight := sessionkeeper.GetSessionGracePeriodBlockCount()

	// TODO_BLOCKER(@Olshansk): Optimize this by indexing claims appropriately
	// and only retrieving the claims that need to be settled rather than all
	// of them and iterating through them one by one.
	claims := k.proofKeeper.GetAllClaims(ctx)

	// Loop over all claims we need to check for expiration
	for _, claim := range claims {
		expirationHeight := claim.SessionHeader.SessionEndBlockHeight + submitProofWindowEndHeight
		if expirationHeight == blockHeight {
			expiringClaims = append(expiringClaims, claim)
		}
	}

	// Return the actually expiring claims
	return expiringClaims, nil
}

// TODO_UPNEXT(@Olshansk): Implement this function. For now, require a proof
// for each claim
func (k Keeper) isProofRequiredForClaim(_ sdk.Context) (bool, error) {
	return true, nil
}

// RemoveClaim(ctx context.Context, sessionId, supplierAddr string)
// GetProof(ctx context.Context, sessionId, supplierAddr string) (proof prooftypes.Proof, isProofFound bool)
