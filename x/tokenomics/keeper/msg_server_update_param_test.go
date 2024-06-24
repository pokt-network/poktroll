package keeper_test

import (
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

func TestMsgUpdateParam_UpdateMinRelayDifficultyBitsOnly(t *testing.T) {
	var expectedComputeUnitsToTokensMultiplier int64 = 8

	// Set the parameters to their default values
	k, msgSrv, ctx := setupMsgServer(t)
	defaultParams := tokenomicstypes.DefaultParams()
	require.NoError(t, k.SetParams(ctx, defaultParams))

	// Ensure the default values are different from the new values we want to set
	require.NotEqual(t, uint64(expectedComputeUnitsToTokensMultiplier), defaultParams.ComputeUnitsToTokensMultiplier)

	// Update the min relay difficulty bits
	updateParamMsg := &tokenomicstypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      tokenomicstypes.ParamComputeUnitsToTokensMultiplier,
		AsType:    &tokenomicstypes.MsgUpdateParam_AsInt64{AsInt64: expectedComputeUnitsToTokensMultiplier},
	}
	res, err := msgSrv.UpdateParam(ctx, updateParamMsg)
	require.NoError(t, err)

	// Ensure the new values are set correctly
	require.Equal(t, uint64(expectedComputeUnitsToTokensMultiplier), res.Params.ComputeUnitsToTokensMultiplier)
}
