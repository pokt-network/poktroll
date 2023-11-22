package config

import (
	"fmt"
	"net/url"

	"gopkg.in/yaml.v2"
)

// YAMLRelayMinerConfig is the structure used to unmarshal the RelayMiner config file
// TODO_DOCUMENT(@red-0ne): Add proper README documentation for yaml config files.
type YAMLRelayMinerConfig struct {
	QueryNodeUrl            string            `yaml:"query_node_url"`
	NetworkNodeUrl          string            `yaml:"network_node_url"`
	PocketNodeWebsocketUrl  string            `yaml:"pocket_node_websocket_url"`
	SigningKeyName          string            `yaml:"signing_key_name"`
	ProxiedServiceEndpoints map[string]string `yaml:"proxied_service_endpoints"`
	SmtStorePath            string            `yaml:"smt_store_path"`
}

// RelayMinerConfig is the structure describing the RelayMiner config
type RelayMinerConfig struct {
	QueryNodeUrl            *url.URL
	NetworkNodeUrl          *url.URL
	PocketNodeWebsocketUrl  string
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

	// Check that the query node URL is provided
	if yamlRelayMinerConfig.QueryNodeUrl == "" {
		return nil, ErrRelayMinerConfigInvalidQueryNodeUrl.Wrapf("query node url is required")
	}

	// Parse the query node URL
	queryNodeUrl, err := url.Parse(yamlRelayMinerConfig.QueryNodeUrl)
	if err != nil {
		return nil, ErrRelayMinerConfigInvalidQueryNodeUrl.Wrapf("%s", err)
	}

	// Check that the network node URL is provided
	if yamlRelayMinerConfig.NetworkNodeUrl == "" {
		return nil, ErrRelayMinerConfigInvalidNetworkNodeUrl.Wrapf("network node url is required")
	}

	// Parse the network node URL
	networkNodeUrl, err := url.Parse(yamlRelayMinerConfig.NetworkNodeUrl)
	if err != nil {
		return nil, ErrRelayMinerConfigInvalidNetworkNodeUrl.Wrapf("%s", err)
	}

	// Parse the websocket URL of the Pocket Node to connect to for subscribing to on-chain events.
	pocketNodeWebsocketUrl := fmt.Sprintf("ws://%s/websocket", queryNodeUrl.Host)

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
		QueryNodeUrl:            queryNodeUrl,
		NetworkNodeUrl:          networkNodeUrl,
		PocketNodeWebsocketUrl:  pocketNodeWebsocketUrl,
		SigningKeyName:          yamlRelayMinerConfig.SigningKeyName,
		ProxiedServiceEndpoints: proxiedServiceEndpoints,
		SmtStorePath:            yamlRelayMinerConfig.SmtStorePath,
	}

	return relayMinerCMDConfig, nil
}
