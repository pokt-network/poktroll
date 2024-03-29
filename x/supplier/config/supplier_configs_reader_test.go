package config_test

import (
	"testing"

	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/yaml"
	"github.com/pokt-network/poktroll/x/gateway/module/config"
)

func Test_ParseGatewayStakeConfig(t *testing.T) {
	tests := []struct {
		desc           string
		expectedErr    *sdkerrors.Error
		expectedConfig *config.GatewayStakeConfig
		inputConfig    string
	}{
		// Valid Configs
		{
			desc: "valid gateway stake config",
			inputConfig: `
				stake_amount: 1000upokt
				`,
			expectedErr: nil,
			expectedConfig: &config.GatewayStakeConfig{
				StakeAmount: sdk.NewCoin("upokt", math.NewInt(1000)),
			},
		},
		// Invalid Configs
		{
			desc:        "services_test: invalid service config with empty content",
			expectedErr: config.ErrGatewayConfigEmptyContent,
			inputConfig: ``,
		},
		{
			desc: "invalid stake denom",
			inputConfig: `
				stake_amount: 1000invalid
				`,
			expectedErr: config.ErrGatewayConfigInvalidStake,
		},
		{
			desc: "negative stake amount",
			inputConfig: `
				stake_amount: -1000upokt
				`,
			expectedErr: config.ErrGatewayConfigInvalidStake,
		},
		{
			desc: "zero stake amount",
			inputConfig: `
				stake_amount: 0upokt
				`,
			expectedErr: config.ErrGatewayConfigInvalidStake,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			normalizedConfig := yaml.NormalizeYAMLIndentation(tt.inputConfig)
			supplierServiceConfig, err := config.ParseGatewayConfig([]byte(normalizedConfig))

			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
				require.Contains(t, err.Error(), tt.expectedErr.Error())
				require.Nil(t, supplierServiceConfig)
				return
			}

			require.NoError(t, err)

			require.Equal(t, tt.expectedConfig.StakeAmount, supplierServiceConfig.StakeAmount)
			require.Equal(t, tt.expectedConfig.StakeAmount.Denom, supplierServiceConfig.StakeAmount.Denom)
		})
	}
}
