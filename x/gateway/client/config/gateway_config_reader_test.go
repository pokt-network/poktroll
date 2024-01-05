package config_test

import (
	"testing"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/status"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/yaml"
	"github.com/pokt-network/poktroll/x/gateway/client/config"
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
				StakeAmount: sdk.NewCoin("upokt", sdk.NewInt(1000)),
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
				stat, ok := status.FromError(tt.expectedError)
				require.True(t, ok)
				require.Contains(t, stat.Message(), tt.expectedError.Error())
				require.Nil(t, supplierServiceConfig)
				return
			}

			require.NoError(t, err)

			require.Equal(t, tt.expectedConfig.StakeAmount, supplierServiceConfig.StakeAmount)
			require.Equal(t, tt.expectedConfig.StakeAmount.Denom, supplierServiceConfig.StakeAmount.Denom)
		})
	}
}
