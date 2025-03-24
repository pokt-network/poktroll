package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/pocket/app/volatile"
	keepertest "github.com/pokt-network/pocket/testutil/keeper"
	suppliertypes "github.com/pokt-network/pocket/x/supplier/types"
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
			expectedErr: suppliertypes.ErrSupplierParamInvalid.Wrapf(
				"min_stake amount must be positive: got -1%s",
				volatile.DenomuPOKT,
			),
		},
		{
			desc: "MinStake equal to zero",
			minStake: &cosmostypes.Coin{
				Denom:  volatile.DenomuPOKT,
				Amount: math.NewInt(0),
			},
			expectedErr: suppliertypes.ErrSupplierParamInvalid.Wrapf(
				"min_stake amount must be greater than 0: got 0%s",
				volatile.DenomuPOKT,
			),
		},
		{
			desc:     "valid MinStake",
			minStake: &suppliertypes.DefaultMinStake,
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

func TestParams_ValidateStakingFee(t *testing.T) {
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
			desc: "SupplierStakingFee less than zero",
			minStake: &cosmostypes.Coin{
				Denom:  volatile.DenomuPOKT,
				Amount: math.NewInt(-1),
			},
			expectedErr: suppliertypes.ErrSupplierParamInvalid.Wrapf(
				"staking_fee amount must be positive: got -1%s",
				volatile.DenomuPOKT,
			),
		},
		{
			desc: "zero SupplierStakingFee",
			minStake: &cosmostypes.Coin{
				Denom:  volatile.DenomuPOKT,
				Amount: math.NewInt(0),
			},
		},
		{
			desc:     "valid SupplierStakingFee",
			minStake: &suppliertypes.DefaultStakingFee,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := suppliertypes.ValidateStakingFee(test.minStake)
			if test.expectedErr != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
