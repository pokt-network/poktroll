package config

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"gopkg.in/yaml.v2"
)

// YAMLStakeGateway is the structure describing the gateway stake config file
type YAMLStakeGateway struct {
	StakeAmount string `yaml:"stake_amount"`
}

// GatewayStakeConfig is the structure describing the gateway stake config
type GatewayStakeConfig struct {
	StakeAmount sdk.Coin
}

// ParseGatewayConfig parses the gateway stake yaml config file into a StakeGatewayConfig struct
func ParseGatewayConfig(configContent []byte) (*GatewayStakeConfig, error) {
	var stakeConfig *YAMLStakeGateway

	if len(configContent) == 0 {
		return nil, ErrGatewayConfigEmptyContent
	}

	// Unmarshal the stake config file into a stakeConfig
	if err := yaml.Unmarshal(configContent, &stakeConfig); err != nil {
		return nil, ErrGatewayConfigUnmarshalYAML.Wrap(err.Error())
	}

	// Validate the stake config
	if len(stakeConfig.StakeAmount) == 0 {
		return nil, ErrGatewayConfigInvalidStake
	}

	// Parse the stake amount to a coin struct
	stakeAmount, err := sdk.ParseCoinNormalized(stakeConfig.StakeAmount)
	if err != nil {
		return nil, ErrGatewayConfigInvalidStake.Wrap(err.Error())
	}

	// Basic validation of the stake amount
	if err := stakeAmount.Validate(); err != nil {
		return nil, ErrGatewayConfigInvalidStake.Wrap(err.Error())
	}

	if stakeAmount.IsZero() {
		return nil, ErrGatewayConfigInvalidStake.Wrap("stake amount cannot be zero")
	}

	// Only allow upokt coins staking
	if stakeAmount.Denom != "upokt" {
		return nil, ErrGatewayConfigInvalidStake.Wrapf(
			"invalid stake denom, expecting: upokt, got: %s",
			stakeAmount.Denom,
		)
	}

	return &GatewayStakeConfig{
		StakeAmount: stakeAmount,
	}, nil
}
