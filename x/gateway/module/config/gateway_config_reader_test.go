package config_test

import (
	"testing"

	sdkerrors "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/yaml"
	"github.com/pokt-network/poktroll/x/gateway/module/config"
)

func Test_ParseGatewayStakeConfig(t *testing.T) {
	tests := []struct {
		desc           string
		expectedError  *sdkerrors.Error
		expectedConfig *config.GatewayStakeConfig
		inputConfig    string
	}{
		// Valid Configs
		{
			desc: "valid gateway stake config",
			inputConfig: `
				stake_amount: 1000upokt
				`,
			expectedError: nil,
			expectedConfig: &config.GatewayStakeConfig{
				StakeAmount: sdk.NewCoin("upokt", sdkmath.NewInt(1000)),
			},
		},
		// Invalid Configs
		{
			desc:          "services_test: invalid service config with empty content",
			expectedError: config.ErrGatewayConfigEmptyContent,
			inputConfig:   ``,
		},
		{
			desc: "invalid stake denom",
			inputConfig: `
				stake_amount: 1000invalid
				`,
			expectedError: config.ErrGatewayConfigInvalidStake,
		},
		{
			desc: "negative stake amount",
			inputConfig: `
				stake_amount: -1000upokt
				`,
			expectedError: config.ErrGatewayConfigInvalidStake,
		},
		{
			desc: "zero stake amount",
			inputConfig: `
				stake_amount: 0upokt
				`,
			expectedError: config.ErrGatewayConfigInvalidStake,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			normalizedConfig := yaml.NormalizeYAMLIndentation(tt.inputConfig)
			supplierServiceConfig, err := config.ParseGatewayConfig([]byte(normalizedConfig))

			if tt.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedError)
				require.Contains(t, err.Error(), tt.expectedError.Error())
				require.Nil(t, supplierServiceConfig)
				return
			}

			require.NoError(t, err)

			require.Equal(t, tt.expectedConfig.StakeAmount, supplierServiceConfig.StakeAmount)
			require.Equal(t, tt.expectedConfig.StakeAmount.Denom, supplierServiceConfig.StakeAmount.Denom)
		})
	}
}
