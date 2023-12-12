package config

import (
	"net/url"

	"gopkg.in/yaml.v2"
)

// YAMLRelayMinerConfig is the structure used to unmarshal the RelayMiner config file
// TODO_DOCUMENT(@red-0ne): Add proper README documentation for yaml config files.
type YAMLRelayMinerConfig struct {
	QueryNodeGRPCUrl        string            `yaml:"query_node_grpc_url"`
	TxNodeGRPCUrl           string            `yaml:"tx_node_grpc_url"`
	QueryNodeRPCUrl         string            `yaml:"query_node_rpc_url"`
	SigningKeyName          string            `yaml:"signing_key_name"`
	ProxiedServiceEndpoints map[string]string `yaml:"proxied_service_endpoints"`
	SmtStorePath            string            `yaml:"smt_store_path"`
}

// RelayMinerConfig is the structure describing the RelayMiner config
type RelayMinerConfig struct {
	QueryNodeGRPCUrl        *url.URL
	TxNodeGRPCUrl           *url.URL
	QueryNodeRPCUrl         *url.URL
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

	// Check that the tx node GRPC URL is provided
	if yamlRelayMinerConfig.TxNodeGRPCUrl == "" {
		return nil, ErrRelayMinerConfigInvalidTxNodeGRPCUrl.Wrapf("tx node grpc url is required")
	}

	// Parse the tx node GRPC URL
	txNodeGRPCUrl, err := url.Parse(yamlRelayMinerConfig.TxNodeGRPCUrl)
	if err != nil {
		return nil, ErrRelayMinerConfigInvalidTxNodeGRPCUrl.Wrapf("%s", err)
	}

	// Check that the query node GRPC URL is provided and default to the tx node GRPC URL if not
	if yamlRelayMinerConfig.QueryNodeGRPCUrl == "" {
		yamlRelayMinerConfig.QueryNodeGRPCUrl = yamlRelayMinerConfig.TxNodeGRPCUrl
	}

	// Parse the query node GRPC URL
	queryNodeGRPCUrl, err := url.Parse(yamlRelayMinerConfig.QueryNodeGRPCUrl)
	if err != nil {
		return nil, ErrRelayMinerConfigInvalidQueryNodeGRPCUrl.Wrapf("%s", err)
	}

	// Check that the network node websocket URL is provided
	if yamlRelayMinerConfig.QueryNodeRPCUrl == "" {
		return nil, ErrRelayMinerConfigInvalidQueryNodeRPCUrl.Wrapf("node rpc url is required")
	}

	// Parse the rpc URL of the Pocket Node to connect to for subscribing to on-chain events.
	queryNodeRPCUrl, err := url.Parse(yamlRelayMinerConfig.QueryNodeRPCUrl)
	if err != nil {
		return nil, ErrRelayMinerConfigInvalidQueryNodeRPCUrl.Wrapf("%s", err)
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
		TxNodeGRPCUrl:           txNodeGRPCUrl,
		QueryNodeRPCUrl:         queryNodeRPCUrl,
		SigningKeyName:          yamlRelayMinerConfig.SigningKeyName,
		ProxiedServiceEndpoints: proxiedServiceEndpoints,
		SmtStorePath:            yamlRelayMinerConfig.SmtStorePath,
	}

	return relayMinerCMDConfig, nil
}
