package tokenomics

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/app/volatile"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// NumComputeUnitsToCoin converts compute units to uPOKT to mint based on global
// network parameters.
func NumComputeUnitsToCoin(params sharedtypes.Params, numClaimComputeUnits uint64) (sdk.Coin, error) {
	// CUTTM is a GLOBAL network wide parameter.
	upoktAmount := math.NewInt(int64(numClaimComputeUnits * params.GetComputeUnitsToTokensMultiplier()))
	if upoktAmount.IsNegative() {
		return sdk.Coin{}, tokenomicstypes.ErrTokenomicsCalculation.Wrapf(
			"num compute units to coin (%d) * CUTTM (%d) resulted in a negative amount: %d",
			numClaimComputeUnits,
			params.GetComputeUnitsToTokensMultiplier(),
			upoktAmount,
		)
	}

	return sdk.NewCoin(volatile.DenomuPOKT, upoktAmount), nil
}
