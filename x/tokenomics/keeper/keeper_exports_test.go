// NB: This file contains exports of unexported members for testing purposes only.
package keeper

import (
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/proto/types/proof"
)

// ProofRequirementForClaim wraps the unexported proofRequirementForClaim function for testing purposes.
func (k Keeper) ProofRequirementForClaim(ctx cosmostypes.Context, claim *proof.Claim) (proof.ProofRequirementReason, error) {
	return k.proofRequirementForClaim(ctx, claim)
}
