package config

import (
	"context"
	"net/url"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"gopkg.in/yaml.v2"

	"github.com/pokt-network/poktroll/pkg/polylog"
	_ "github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// YAMLStakeConfig is the structure describing the supplier stake config file.
type YAMLStakeConfig struct {
	OwnerAddress           string              `yaml:"owner_address"`
	OperatorAddress        string              `yaml:"operator_address"`
	StakeAmount            string              `yaml:"stake_amount"`
	Services               []*YAMLStakeService `yaml:"services"`
	DefaultRevSharePercent map[string]float32  `yaml:"default_rev_share_percent"`
}

// YAMLStakeService is the structure describing a single service entry in the
// stake config file.
type YAMLStakeService struct {
	ServiceId       string                `yaml:"service_id"`
	RevSharePercent map[string]float32    `yaml:"rev_share_percent"`
	Endpoints       []YAMLServiceEndpoint `yaml:"endpoints"`
}

// YAMLServiceEndpoint is the structure describing a single service endpoint in
// the service section of the stake config file.
type YAMLServiceEndpoint struct {
	PubliclyExposedUrl string            `yaml:"publicly_exposed_url"`
	RPCType            string            `yaml:"rpc_type"`
	Config             map[string]string `yaml:"config,omitempty"`
}

// SupplierStakeConfig is the structure describing the parsed supplier stake config.
type SupplierStakeConfig struct {
	OwnerAddress    string
	OperatorAddress string
	StakeAmount     sdk.Coin
	Services        []*sharedtypes.SupplierServiceConfig
}

// ParseSupplierServiceConfig parses the stake config file into a SupplierServiceConfig.
func ParseSupplierConfigs(ctx context.Context, configContent []byte) (*SupplierStakeConfig, error) {
	var stakeConfig *YAMLStakeConfig

	logger := polylog.Ctx(ctx)

	if len(configContent) == 0 {
		return nil, ErrSupplierConfigEmptyContent
	}

	// Unmarshal the stake config file into a stakeConfig
	if err := yaml.Unmarshal(configContent, &stakeConfig); err != nil {
		return nil, ErrSupplierConfigUnmarshalYAML.Wrapf("%s", err)
	}

	// Validate required owner address.
	if _, err := sdk.AccAddressFromBech32(stakeConfig.OwnerAddress); err != nil {
		return nil, ErrSupplierConfigInvalidOwnerAddress.Wrap("invalid owner address")
	}

	// If the operator address is not set, default it to the owner address.
	if stakeConfig.OperatorAddress == "" {
		stakeConfig.OperatorAddress = stakeConfig.OwnerAddress
		logger.Info().Msg("operator address not set, defaulting to owner address")
	}

	// Validate operator address.
	if _, err := sdk.AccAddressFromBech32(stakeConfig.OperatorAddress); err != nil {
		return nil, ErrSupplierConfigInvalidOperatorAddress.Wrap("invalid operator address")
	}

	// Validate the stake amount
	if len(stakeConfig.StakeAmount) == 0 {
		return nil, ErrSupplierConfigInvalidStake.Wrap("stake amount cannot be empty")
	}

	stakeAmount, err := sdk.ParseCoinNormalized(stakeConfig.StakeAmount)
	if err != nil {
		return nil, ErrSupplierConfigInvalidStake.Wrap(err.Error())
	}

	if err := stakeAmount.Validate(); err != nil {
		return nil, ErrSupplierConfigInvalidStake.Wrap(err.Error())
	}

	if stakeAmount.IsZero() {
		return nil, ErrSupplierConfigInvalidStake.Wrap("stake amount cannot be zero")
	}

	if stakeAmount.Denom != "upokt" {
		return nil, ErrSupplierConfigInvalidStake.Wrapf(
			"invalid stake denom, expecting: upokt, got: %s",
			stakeAmount.Denom,
		)
	}

	defaultRevSharePercent := map[string]float32{}
	if len(stakeConfig.DefaultRevSharePercent) == 0 {
		// Ensure that if no default rev share is provided, the owner address is set
		// to 100% rev share.
		if stakeConfig.OwnerAddress == "" {
			return nil, ErrSupplierConfigInvalidOwnerAddress.Wrap("owner address cannot be empty")
		}
		defaultRevSharePercent[stakeConfig.OwnerAddress] = 100
	} else {
		defaultRevSharePercent = stakeConfig.DefaultRevSharePercent
	}

	// Validate the services
	if len(stakeConfig.Services) == 0 {
		return nil, ErrSupplierConfigInvalidServiceId.Wrap("serviceIds cannot be empty")
	}

	// Prepare the supplierServiceConfig
	supplierServiceConfig := make([]*sharedtypes.SupplierServiceConfig, 0, len(stakeConfig.Services))

	// Populate the services slice
	for _, svc := range stakeConfig.Services {
		// Validate the serviceId
		if !sharedtypes.IsValidServiceId(svc.ServiceId) {
			return nil, ErrSupplierConfigInvalidServiceId.Wrapf("%s", svc.ServiceId)
		}

		if len(svc.Endpoints) == 0 {
			return nil, ErrSupplierConfigNoEndpoints.Wrapf("%s", svc.ServiceId)
		}

		// Create a supplied service config with the serviceId
		service := &sharedtypes.SupplierServiceConfig{
			ServiceId: svc.ServiceId,
			RevShare:  []*sharedtypes.ServiceRevenueShare{},
			Endpoints: []*sharedtypes.SupplierEndpoint{},
		}

		// Iterate over the service endpoints and add their parsed representation to the supplied service config
		for _, endpoint := range svc.Endpoints {
			parsedEndpointEntry, err := parseEndpointEntry(endpoint)
			if err != nil {
				return nil, err
			}
			service.Endpoints = append(service.Endpoints, parsedEndpointEntry)
		}

		serviceConfigRevShare := svc.RevSharePercent
		// If the service does not have a rev share, use the default one.
		if serviceConfigRevShare == nil {
			serviceConfigRevShare = defaultRevSharePercent
		}

		for address, revSharePercent := range serviceConfigRevShare {
			service.RevShare = append(service.RevShare, &sharedtypes.ServiceRevenueShare{
				Address:            address,
				RevSharePercentage: revSharePercent,
			})
		}

		if err := sharedtypes.ValidateServiceRevShare(service.RevShare); err != nil {
			return nil, err
		}

		supplierServiceConfig = append(supplierServiceConfig, service)
	}

	return &SupplierStakeConfig{
		OwnerAddress:    stakeConfig.OwnerAddress,
		OperatorAddress: stakeConfig.OperatorAddress,
		StakeAmount:     stakeAmount,
		Services:        supplierServiceConfig,
	}, nil
}

func parseEndpointEntry(endpoint YAMLServiceEndpoint) (*sharedtypes.SupplierEndpoint, error) {
	endpointEntry := &sharedtypes.SupplierEndpoint{}
	var err error

	// Endpoint URL
	if endpointEntry.Url, err = validateEndpointURL(endpoint); err != nil {
		return nil, err
	}

	// Endpoint config
	if endpointEntry.Configs, err = parseEndpointConfigs(endpoint); err != nil {
		return nil, err
	}

	// Endpoint RPC type
	if endpointEntry.RpcType, err = parseEndpointRPCType(endpoint); err != nil {
		return nil, err
	}

	return endpointEntry, nil
}

// parseEndpointConfigs parses the endpoint config entries into a slice of
// ConfigOption compatible with the SupplierEndpointConfig.
// It accepts a nil config entry or a map of valid config keys.
func parseEndpointConfigs(endpoint YAMLServiceEndpoint) ([]*sharedtypes.ConfigOption, error) {
	// Prepare the endpoint configs slice
	endpointConfigs := []*sharedtypes.ConfigOption{}

	// If we have an endpoint config entry, parse it into a slice of ConfigOption
	if endpoint.Config == nil {
		return endpointConfigs, nil
	}

	// Iterate over the endpoint config entries and add them to the slice of ConfigOption
	for key, value := range endpoint.Config {
		var configKey sharedtypes.ConfigOptions

		// Make sure the config key is valid
		switch key {
		case "timeout":
			configKey = sharedtypes.ConfigOptions_TIMEOUT
		default:
			return nil, ErrSupplierConfigInvalidEndpointConfig.Wrapf("%s", key)
		}

		config := &sharedtypes.ConfigOption{
			Key:   configKey,
			Value: value,
		}
		endpointConfigs = append(endpointConfigs, config)
	}

	return endpointConfigs, nil
}

// parseEndpointRPCType parses the endpoint RPC type into a sharedtypes.RPCType
func parseEndpointRPCType(endpoint YAMLServiceEndpoint) (sharedtypes.RPCType, error) {
	switch strings.ToLower(endpoint.RPCType) {
	case "json_rpc":
		return sharedtypes.RPCType_JSON_RPC, nil
	case "rest":
		return sharedtypes.RPCType_REST, nil
	default:
		return sharedtypes.RPCType_UNKNOWN_RPC, ErrSupplierConfigInvalidRPCType.Wrapf("%s", endpoint.RPCType)
	}
}

// validateEndpointURL validates the endpoint URL, making sure that the string provided is a valid URL
func validateEndpointURL(endpoint YAMLServiceEndpoint) (string, error) {
	// Validate the endpoint URL
	if _, err := url.Parse(endpoint.PubliclyExposedUrl); err != nil {
		return "", ErrSupplierConfigInvalidURL.Wrapf("%s", err)
	}

	return endpoint.PubliclyExposedUrl, nil
}
