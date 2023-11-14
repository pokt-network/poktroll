package cli

import (
	"fmt"
	"net/url"
	"os"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"gopkg.in/yaml.v2"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// ParsedStakeConfig is a parsed version of the stake config file
type ParsedStakeConfig struct {
	stake    sdk.Coin
	services []*sharedtypes.SupplierServiceConfig
}

// StakeConfig is the structure of the stake config file
type StakeConfig struct {
	Stake    string         `json:"stake"`
	Services []StakeService `json:"services"`
}

// StakeService is the structure describing a single service stake entry in the stake config file
type StakeService struct {
	ServiceId string            `json:"service_id"`
	Endpoints []ServiceEndpoint `json:"endpoints"`
}

// ServiceEndpoint is the structure describing a single service endpoint in the stake config file
type ServiceEndpoint struct {
	Url     string            `json:"url"`
	RPCType string            `json:"rpc_type"`
	Config  map[string]string `json:"config"`
}

// parseStakeConfig parses the stake config file into a parsedStakeConfig
func parseStakeConfig(configFile string) (*ParsedStakeConfig, error) {
	// Read the stake config file into memory
	configContent, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	// Unmarshal the stake config file into a stakeConfig
	var stakeConfig *StakeConfig
	if err := yaml.Unmarshal(configContent, &stakeConfig); err != nil {
		return nil, err
	}

	// Prepare the parsedStakeConfig
	parsedStakeConfig := &ParsedStakeConfig{}

	// Parse the stake amount and assign it to the parsedStakeConfig
	parsedStakeConfig.stake, err = sdk.ParseCoinNormalized(stakeConfig.Stake)
	if err != nil {
		return nil, err
	}

	// Prepare the services slice
	var services []*sharedtypes.SupplierServiceConfig

	// Populate the services slice
	for _, svc := range stakeConfig.Services {
		// Validate the serviceId
		// TODO_TECH_DEBT: This should be validated against some governance state
		// defining the network's supported services
		if svc.ServiceId == "" {
			return nil, fmt.Errorf("invalid serviceId")
		}

		// Create a supplied service config with the serviceId
		service := &sharedtypes.SupplierServiceConfig{
			Service:   &sharedtypes.Service{Id: svc.ServiceId},
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
		services = append(services, service)
	}

	parsedStakeConfig.services = services

	return parsedStakeConfig, nil
}

func parseEndpointEntry(endpoint ServiceEndpoint) (*sharedtypes.SupplierEndpoint, error) {
	endpointEntry := &sharedtypes.SupplierEndpoint{}

	// Endpoint URL
	endpointUrl, err := validateEndpointURL(endpoint)
	if err != nil {
		return nil, err
	}
	endpointEntry.Url = endpointUrl

	// Endpoint config
	endpointEntry.Configs = parseEndpointConfigs(endpoint)

	// Endpoint RPC type
	endpointEntry.RpcType = parseEndpointRPCType(endpoint)

	return endpointEntry, nil
}

func validateEndpointURL(endpoint ServiceEndpoint) (string, error) {
	// Validate the endpoint URL
	if _, err := url.Parse(endpoint.Url); err != nil {
		return "", err
	}

	return endpoint.Url, nil
}

func parseEndpointConfigs(endpoint ServiceEndpoint) []*sharedtypes.ConfigOption {
	// Prepare the endpoint configs slice
	endpointConfigs := []*sharedtypes.ConfigOption{}

	// If we have an endpoint config entry, parse it into a slice of ConfigOption
	if endpoint.Config != nil {
		// Iterate over the endpoint config entries and add them to the slice of ConfigOption
		for key, value := range endpoint.Config {
			var configKey sharedtypes.ConfigOptions

			// Make sure the config key is valid
			switch key {
			case "timeout":
				configKey = sharedtypes.ConfigOptions_TIMEOUT
			default:
				configKey = sharedtypes.ConfigOptions_UNKNOWN_CONFIG
			}

			config := &sharedtypes.ConfigOption{
				Key:   configKey,
				Value: value,
			}
			endpointConfigs = append(endpointConfigs, config)
		}
	}

	return endpointConfigs
}

func parseEndpointRPCType(endpoint ServiceEndpoint) sharedtypes.RPCType {
	switch endpoint.RPCType {
	case "json_rpc":
		return sharedtypes.RPCType_JSON_RPC
	default:
		return sharedtypes.RPCType_UNKNOWN_RPC
	}
}
