package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)
	params := servicetypes.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
}

func TestParams_ValidateAddServiceFee(t *testing.T) {
	tests := []struct {
		desc          string
		addServiceFee any
		expectedErr   error
	}{
		{
			desc:          "invalid type",
			addServiceFee: "100upokt",
			expectedErr:   servicetypes.ErrServiceParamInvalid,
		},
		{
			desc:          "valid AddServiceFee",
			addServiceFee: &servicetypes.MinAddServiceFee,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := servicetypes.ValidateAddServiceFee(test.addServiceFee)
			if test.expectedErr != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParams_ValidateTargetNumRelays(t *testing.T) {
	tests := []struct {
		desc            string
		targetNumRelays any
		expectedErr     error
	}{
		{
			desc:            "invalid type",
			targetNumRelays: "999",
			expectedErr:     servicetypes.ErrServiceParamInvalid,
		},
		{
			desc:            "invalid value",
			targetNumRelays: 0,
			expectedErr:     servicetypes.ErrServiceParamInvalid,
		},
		{
			desc:            "valid TargetNumRelays",
			targetNumRelays: uint64(999),
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := servicetypes.ValidateTargetNumRelays(test.targetNumRelays)
			if test.expectedErr != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
