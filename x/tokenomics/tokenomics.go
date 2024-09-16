package tokenomics

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// NumComputeUnitsToCoin calculates the amount of uPOKT to mint based on the number of compute units.
func NumComputeUnitsToCoin(params types.Params, numClaimComputeUnits uint64) (sdk.Coin, error) {
	// CUPR is a LOCAL service specific parameter
	upoktAmount := math.NewInt(int64(numClaimComputeUnits * params.ComputeUnitsToTokensMultiplier))
	if upoktAmount.IsNegative() {
		return sdk.Coin{}, types.ErrTokenomicsRootHashInvalid.Wrap("sum * compute_units_to_tokens_multiplier is negative")
	}

	return sdk.NewCoin(volatile.DenomuPOKT, upoktAmount), nil
}
