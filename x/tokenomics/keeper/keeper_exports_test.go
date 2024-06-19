// NB: This file contains exports of unexported members for testing purposes only.
package keeper

import (
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

// IsProofRequiredForClaim wraps the unexported isProofRequiredForClaim function for testing purposes.
func (k Keeper) IsProofRequiredForClaim(ctx cosmostypes.Context, claim *prooftypes.Claim) (bool, error) {
	return k.isProofRequiredForClaim(ctx, claim)
}
