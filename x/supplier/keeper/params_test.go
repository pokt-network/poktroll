package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

func TestGetParams(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	params := suppliertypes.DefaultParams()

	require.NoError(t, supplierModuleKeepers.SetParams(ctx, params))
	require.EqualValues(t, params, supplierModuleKeepers.Keeper.GetParams(ctx))
}

func TestParams_ValidateMinStake(t *testing.T) {
	tests := []struct {
		desc        string
		minStake    any
		expectedErr error
	}{
		{
			desc:        "invalid type",
			minStake:    "420",
			expectedErr: suppliertypes.ErrSupplierParamInvalid.Wrapf("invalid parameter type: string"),
		},
		{
			desc: "MinStake less than zero",
			minStake: &cosmostypes.Coin{
				Denom:  volatile.DenomuPOKT,
				Amount: math.NewInt(-1),
			},
			expectedErr: gatewaytypes.ErrGatewayParamInvalid.Wrapf("min stake amount must be greater than zero: -1%s", volatile.DenomuPOKT),
		},
		{
			desc:     "valid MinStake",
			minStake: int64(420),
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := suppliertypes.ValidateMinStake(test.minStake)
			if test.expectedErr != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
