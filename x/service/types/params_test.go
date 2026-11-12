package types_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/x/service/types"
)

// Mutating the returned defaults must not change the package-level minimum that
// ValidateAddServiceFee compares against.
func TestDefaultParams_DoesNotAliasGlobals(t *testing.T) {
	params := types.DefaultParams()
	*params.AddServiceFee = sdk.Coin{}

	require.Equal(t, sdk.NewCoin(pocket.DenomuPOKT, math.NewInt(1)), types.MinAddServiceFee)
	require.NoError(t, types.DefaultParams().ValidateBasic())
}
