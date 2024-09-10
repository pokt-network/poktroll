package config

import (
	"gopkg.in/yaml.v2"

	sdk "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// YAMLApplicationConfig is the structure describing a single service stake entry in the stake config file
// TODO_DOCUMENT(@red-0ne): Add additional documentation on app config files.
type YAMLApplicationConfig struct {
	StakeAmount string   `yaml:"stake_amount"`
	ServiceIds  []string `yaml:"service_ids"`
}

type ApplicationStakeConfig struct {
	// StakeAmount is the amount of upokt tokens that the application is willing to stake
	StakeAmount sdk.Coin
	// Services is the list of services that the application is willing to stake for
	Services []*sharedtypes.ApplicationServiceConfig
}

// ParseApplicationConfig parses the stake config file and returns a slice of ApplicationServiceConfig
func ParseApplicationConfigs(configContent []byte) (*ApplicationStakeConfig, error) {
	var parsedAppConfig YAMLApplicationConfig

	if len(configContent) == 0 {
		return nil, ErrApplicationConfigEmptyContent
	}

	// Unmarshal the stake config file into a applicationServiceConfig
	if err := yaml.Unmarshal(configContent, &parsedAppConfig); err != nil {
		return nil, ErrApplicationConfigUnmarshalYAML.Wrapf("%s", err)
	}

	if len(parsedAppConfig.ServiceIds) == 0 || parsedAppConfig.ServiceIds == nil {
		return nil, ErrApplicationConfigInvalidServiceId.Wrap("serviceIds cannot be empty")
	}

	if parsedAppConfig.StakeAmount == "" {
		return nil, ErrApplicationConfigInvalidStake.Wrap("stake amount cannot be empty")
	}

	stakeAmount, err := sdk.ParseCoinNormalized(parsedAppConfig.StakeAmount)
	if err != nil {
		return nil, ErrApplicationConfigInvalidStake.Wrap(err.Error())
	}

	if err := stakeAmount.Validate(); err != nil {
		return nil, ErrApplicationConfigInvalidStake.Wrap(err.Error())
	}

	if stakeAmount.IsZero() {
		return nil, ErrApplicationConfigInvalidStake.Wrap("stake amount cannot be zero")
	}

	if stakeAmount.Denom != "upokt" {
		return nil, ErrApplicationConfigInvalidStake.Wrapf(
			"invalid stake denom, expecting: upokt, got: %s",
			stakeAmount.Denom,
		)
	}

	// Prepare the applicationServiceConfig
	applicationServiceConfig := make(
		[]*sharedtypes.ApplicationServiceConfig,
		0,
		len(parsedAppConfig.ServiceIds),
	)

	for _, serviceId := range parsedAppConfig.ServiceIds {
		// Validate serviceId
		if !sharedtypes.IsValidServiceId(serviceId) {
			return nil, ErrApplicationConfigInvalidServiceId.Wrapf("%s", serviceId)
		}

		appServiceConfig := &sharedtypes.ApplicationServiceConfig{
			Service: &sharedtypes.Service{Id: serviceId},
		}

		applicationServiceConfig = append(applicationServiceConfig, appServiceConfig)
	}

	return &ApplicationStakeConfig{
		StakeAmount: stakeAmount,
		Services:    applicationServiceConfig,
	}, nil
}
