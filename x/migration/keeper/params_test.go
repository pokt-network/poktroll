package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/x/migration/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := keepertest.MigrationKeeper(t)
	params := types.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
}

func TestParams_ValidateWaiveMorseClaimGasFees(t *testing.T) {
	tests := []struct {
		desc         string
		newParameter any
		expectedErr  error
	}{
		{
			desc:         "invalid type",
			newParameter: "420",
			expectedErr:  types.ErrMigrationParamInvalid.Wrapf("invalid parameter type: string"),
		},
		{
			desc:         "valid WaiveMorseClaimGasFees",
			newParameter: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := types.ValidateParamIsBool(test.newParameter)
			if test.expectedErr != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParams_ValidateAllowMorseAccountImportOverwrite(t *testing.T) {
	tests := []struct {
		desc                             string
		allowMorseAccountImportOverwrite any
		expectedErr                      error
	}{
		{
			desc:                             "invalid type",
			allowMorseAccountImportOverwrite: "420",
			expectedErr:                      types.ErrMigrationParamInvalid.Wrapf("invalid parameter type: string"),
		},
		{
			desc:                             "valid AllowMorseAccountImportOverwrite",
			allowMorseAccountImportOverwrite: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := types.ValidateParamIsBool(test.allowMorseAccountImportOverwrite)
			if test.expectedErr != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParams_ValidateNewParameter(t *testing.T) {
	tests := []struct {
		desc                        string
		morseAccountClaimingEnabled any
		expectedErr                 error
	}{
		{
			desc:                        "invalid type",
			morseAccountClaimingEnabled: "420",
			expectedErr:                 types.ErrMigrationParamInvalid.Wrapf("invalid parameter type: string"),
		},
		{
			desc:                        "valid NewParameterName",
			morseAccountClaimingEnabled: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := types.ValidateParamIsBool(test.morseAccountClaimingEnabled)
			if test.expectedErr != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
