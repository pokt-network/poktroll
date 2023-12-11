package config

import (
	"net/url"

	"gopkg.in/yaml.v2"
)

// YAMLRelayMinerConfig is the structure used to unmarshal the RelayMiner config file
// TODO_DOCUMENT(@red-0ne): Add proper README documentation for yaml config files.
type YAMLRelayMinerConfig struct {
	QueryNodeGRPCUrl        string            `yaml:"query_node_grpc_url"`
	NetworkNodeGRPCUrl      string            `yaml:"network_node_grpc_url"`
	GRPCInsecure            bool              `yaml:"grpc_insecure"`
	PocketNodeWebsocketUrl  string            `yaml:"pocket_node_websocket_url"`
	SigningKeyName          string            `yaml:"signing_key_name"`
	ProxiedServiceEndpoints map[string]string `yaml:"proxied_service_endpoints"`
	SmtStorePath            string            `yaml:"smt_store_path"`
}

// RelayMinerConfig is the structure describing the RelayMiner config
type RelayMinerConfig struct {
	QueryNodeGRPCUrl        *url.URL
	NetworkNodeGRPCUrl      *url.URL
	GRPCInsecure            bool
	PocketNodeWebsocketUrl  *url.URL
	SigningKeyName          string
	ProxiedServiceEndpoints map[string]*url.URL
	SmtStorePath            string
}

// ParseRelayMinerConfigs parses the relay miner config file into a RelayMinerConfig
func ParseRelayMinerConfigs(configContent []byte) (*RelayMinerConfig, error) {
	var yamlRelayMinerConfig YAMLRelayMinerConfig

	// Unmarshal the stake config file into a yamlAppGateConfig
	if err := yaml.Unmarshal(configContent, &yamlRelayMinerConfig); err != nil {
		return nil, ErrRelayMinerConfigUnmarshalYAML.Wrapf("%s", err)
	}

	// Check that the query node GRPC URL is provided
	if yamlRelayMinerConfig.QueryNodeGRPCUrl == "" {
		return nil, ErrRelayMinerConfigInvalidQueryNodeGRPCUrl.Wrapf("query node url is required")
	}

	// Parse the query node GRPC URL
	queryNodeGRPCUrl, err := url.Parse(yamlRelayMinerConfig.QueryNodeGRPCUrl)
	if err != nil {
		return nil, ErrRelayMinerConfigInvalidQueryNodeGRPCUrl.Wrapf("%s", err)
	}

	// Check that the network node GRPC URL is provided
	if yamlRelayMinerConfig.NetworkNodeGRPCUrl == "" {
		return nil, ErrRelayMinerConfigInvalidNetworkNodeGRPCUrl.Wrapf("network node url is required")
	}

	// Parse the network node GRPC URL
	networkNodeGRPCUrl, err := url.Parse(yamlRelayMinerConfig.NetworkNodeGRPCUrl)
	if err != nil {
		return nil, ErrRelayMinerConfigInvalidNetworkNodeGRPCUrl.Wrapf("%s", err)
	}

	// Check that the network node websocket URL is provided
	if yamlRelayMinerConfig.PocketNodeWebsocketUrl == "" {
		return nil, ErrRelayMinerConfigInvalidPocketNodeWebsocketUrl.Wrapf("pocket node websocket url is required")
	}

	// Parse the websocket URL of the Pocket Node to connect to for subscribing to on-chain events.
	pocketNodeWebsocketUrl, err := url.Parse(yamlRelayMinerConfig.PocketNodeWebsocketUrl)
	if err != nil {
		return nil, ErrRelayMinerConfigInvalidPocketNodeWebsocketUrl.Wrapf("%s", err)
	}

	if yamlRelayMinerConfig.SigningKeyName == "" {
		return nil, ErrRelayMinerConfigInvalidSigningKeyName
	}

	if yamlRelayMinerConfig.SmtStorePath == "" {
		return nil, ErrRelayMinerConfigInvalidSmtStorePath
	}

	if yamlRelayMinerConfig.ProxiedServiceEndpoints == nil {
		return nil, ErrRelayMinerConfigInvalidServiceEndpoint.Wrapf("proxied service endpoints are required")
	}

	if len(yamlRelayMinerConfig.ProxiedServiceEndpoints) == 0 {
		return nil, ErrRelayMinerConfigInvalidServiceEndpoint.Wrapf("no proxied service endpoints provided")
	}

	// Parse the proxied service endpoints
	proxiedServiceEndpoints := make(map[string]*url.URL, len(yamlRelayMinerConfig.ProxiedServiceEndpoints))
	for serviceId, endpointUrl := range yamlRelayMinerConfig.ProxiedServiceEndpoints {
		endpoint, err := url.Parse(endpointUrl)
		if err != nil {
			return nil, ErrRelayMinerConfigInvalidServiceEndpoint.Wrapf("%s", err)
		}
		proxiedServiceEndpoints[serviceId] = endpoint
	}

	relayMinerCMDConfig := &RelayMinerConfig{
		QueryNodeGRPCUrl:        queryNodeGRPCUrl,
		NetworkNodeGRPCUrl:      networkNodeGRPCUrl,
		GRPCInsecure:            yamlRelayMinerConfig.GRPCInsecure,
		PocketNodeWebsocketUrl:  pocketNodeWebsocketUrl,
		SigningKeyName:          yamlRelayMinerConfig.SigningKeyName,
		ProxiedServiceEndpoints: proxiedServiceEndpoints,
		SmtStorePath:            yamlRelayMinerConfig.SmtStorePath,
	}

	return relayMinerCMDConfig, nil
}
