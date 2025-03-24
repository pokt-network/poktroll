package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/x/application/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := keepertest.ApplicationKeeper(t)
	params := types.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
}

func TestParams_ValidateMaxDelegatedGateways(t *testing.T) {
	tests := []struct {
		desc                 string
		maxDelegatedGateways any
		expectedErr          error
	}{
		{
			desc:                 "invalid type",
			maxDelegatedGateways: "0",
			expectedErr:          types.ErrAppParamInvalid.Wrap("invalid parameter type: string"),
		},
		{
			desc:                 "MaxDelegatedGateways less than 1",
			maxDelegatedGateways: uint64(0),
			expectedErr:          types.ErrAppParamInvalid.Wrapf("max_delegated_gateways must be greater than 0: got %d", 0),
		},
		{
			desc:                 "valid MaxDelegatedGateways",
			maxDelegatedGateways: types.DefaultMaxDelegatedGateways,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := types.ValidateMaxDelegatedGateways(test.maxDelegatedGateways)
			if test.expectedErr != nil {
				require.ErrorContains(t, err, test.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParams_ValidateMinStake(t *testing.T) {
	tests := []struct {
		desc        string
		minStake    any
		expectedErr error
	}{
		{
			desc:        "invalid type",
			minStake:    "0",
			expectedErr: types.ErrAppParamInvalid.Wrapf("invalid parameter type: string"),
		},
		{
			desc: "MinStake with invalid denom",
			minStake: &cosmostypes.Coin{
				Denom:  "💩coin",
				Amount: math.NewInt(1),
			},
			expectedErr: types.ErrAppParamInvalid.Wrapf(
				"invalid min_stake denom %q; expected %q",
				"💩coin", volatile.DenomuPOKT,
			),
		},
		{
			desc: "MinStake less than zero",
			minStake: &cosmostypes.Coin{
				Denom:  volatile.DenomuPOKT,
				Amount: math.NewInt(-1),
			},
			expectedErr: types.ErrAppParamInvalid.Wrapf("invalid min_stake amount: -1%s <= 0", volatile.DenomuPOKT),
		},
		{
			desc:        "valid MinStake",
			minStake:    &types.DefaultMinStake,
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := types.ValidateMinStake(test.minStake)
			if test.expectedErr != nil {
				require.ErrorContains(t, err, test.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
