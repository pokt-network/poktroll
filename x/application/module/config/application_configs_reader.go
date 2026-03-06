package config

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"gopkg.in/yaml.v2"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// YAMLApplicationConfig is the structure describing a single service stake entry in the stake config file
// TODO_DOCUMENT(@red-0ne): Add additional documentation on app config files.
type YAMLApplicationConfig struct {
	StakeAmount          string   `yaml:"stake_amount"`
	ServiceIds           []string `yaml:"service_ids"`
	GatewayAddresses     []string `yaml:"gateway_addresses"`
	PerSessionSpendLimit string   `yaml:"per_session_spend_limit"`
}

type ApplicationStakeConfig struct {
	// StakeAmount is the amount of upokt tokens that the application is willing to stake
	StakeAmount sdk.Coin
	// Services is the list of services that the application is willing to stake for
	Services []*sharedtypes.ApplicationServiceConfig
	// GatewayAddresses is an optional list of gateway addresses to delegate to
	GatewayAddresses []string
	// PerSessionSpendLimit is an optional per-session spend limit in upokt
	PerSessionSpendLimit *sdk.Coin
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
		if err := sharedtypes.IsValidServiceId(serviceId); err != nil {
			return nil, ErrApplicationConfigInvalidServiceId.Wrapf("%v", err.Error())
		}

		appServiceConfig := &sharedtypes.ApplicationServiceConfig{
			ServiceId: serviceId,
		}

		applicationServiceConfig = append(applicationServiceConfig, appServiceConfig)
	}

	// Validate gateway addresses if provided
	for _, gwAddr := range parsedAppConfig.GatewayAddresses {
		if _, err := sdk.AccAddressFromBech32(gwAddr); err != nil {
			return nil, ErrApplicationConfigInvalidGateway.Wrapf("invalid gateway address %q: %s", gwAddr, err)
		}
	}

	// Parse and validate the optional per-session spend limit
	var perSessionSpendLimit *sdk.Coin
	if parsedAppConfig.PerSessionSpendLimit != "" {
		spendLimit, err := sdk.ParseCoinNormalized(parsedAppConfig.PerSessionSpendLimit)
		if err != nil {
			return nil, ErrApplicationConfigInvalidSpendLimit.Wrap(err.Error())
		}
		if spendLimit.IsNegative() {
			return nil, ErrApplicationConfigInvalidSpendLimit.Wrap("per-session spend limit cannot be negative")
		}
		if spendLimit.Denom != "upokt" {
			return nil, ErrApplicationConfigInvalidSpendLimit.Wrapf(
				"invalid spend limit denom, expecting: upokt, got: %s",
				spendLimit.Denom,
			)
		}
		perSessionSpendLimit = &spendLimit
	}

	return &ApplicationStakeConfig{
		StakeAmount:          stakeAmount,
		Services:             applicationServiceConfig,
		GatewayAddresses:     parsedAppConfig.GatewayAddresses,
		PerSessionSpendLimit: perSessionSpendLimit,
	}, nil
}
