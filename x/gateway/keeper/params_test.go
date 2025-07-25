package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.GatewayKeeper(t)
	params := gatewaytypes.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
}

func TestParams_ValidateMinStake(t *testing.T) {
	tests := []struct {
		desc        string
		minStake    any
		expectedErr error
	}{
		{
			desc:        "invalid type",
			minStake:    int64(-1),
			expectedErr: gatewaytypes.ErrGatewayParamInvalid.Wrapf("invalid type for %s: int64; expected *cosmostypes.Coin", gatewaytypes.ParamMinStake),
		},
		{
			desc: "MinStake less than zero",
			minStake: &cosmostypes.Coin{
				Denom:  pocket.DenomuPOKT,
				Amount: math.NewInt(-1),
			},
			expectedErr: gatewaytypes.ErrGatewayParamInvalid.Wrapf("min stake amount must be greater than zero: -1%s", pocket.DenomuPOKT),
		},
		{
			desc:     "valid MinStake",
			minStake: &gatewaytypes.DefaultMinStake,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := gatewaytypes.ValidateMinStake(tt.minStake)
			if tt.expectedErr != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
