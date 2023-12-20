package config

import (
	"net/url"

	yaml "gopkg.in/yaml.v2"
)

// YAMLRelayMinerConfig is the structure used to unmarshal the RelayMiner config file
// TODO_DOCUMENT(@red-0ne): Add proper README documentation for yaml config files.
type YAMLRelayMinerConfig struct {
	QueryNodeRPCUrl         string            `yaml:"query_node_rpc_url"`
	QueryNodeGRPCUrl        string            `yaml:"query_node_grpc_url"`
	TxNodeGRPCUrl           string            `yaml:"tx_node_grpc_url"`
	SigningKeyName          string            `yaml:"signing_key_name"`
	SmtStorePath            string            `yaml:"smt_store_path"`
	ProxiedServiceEndpoints map[string]string `yaml:"proxied_service_endpoints"`
}

// RelayMinerConfig is the structure describing the RelayMiner config
type RelayMinerConfig struct {
	QueryNodeRPCUrl         *url.URL
	QueryNodeGRPCUrl        *url.URL
	TxNodeGRPCUrl           *url.URL
	SigningKeyName          string
	SmtStorePath            string
	ProxiedServiceEndpoints map[string]*url.URL
}

// ParseRelayMinerConfigs parses the relay miner config file into a RelayMinerConfig
func ParseRelayMinerConfigs(configContent []byte) (*RelayMinerConfig, error) {
	var yamlRelayMinerConfig YAMLRelayMinerConfig

	if len(configContent) == 0 {
		return nil, ErrRelayMinerConfigEmpty
	}

	// Unmarshal the stake config file into a yamlAppGateConfig
	if err := yaml.Unmarshal(configContent, &yamlRelayMinerConfig); err != nil {
		return nil, ErrRelayMinerConfigUnmarshalYAML.Wrap(err.Error())
	}

	// Check that the tx node GRPC URL is provided
	if len(yamlRelayMinerConfig.TxNodeGRPCUrl) == 0 {
		return nil, ErrRelayMinerConfigInvalidTxNodeGRPCUrl.Wrap("tx node grpc url is required")
	}

	// Parse the tx node GRPC URL
	txNodeGRPCUrl, err := url.Parse(yamlRelayMinerConfig.TxNodeGRPCUrl)
	if err != nil {
		return nil, ErrRelayMinerConfigInvalidTxNodeGRPCUrl.Wrap(err.Error())
	}

	// Check that the query node GRPC URL is provided and default to the tx node GRPC URL if not
	if len(yamlRelayMinerConfig.QueryNodeGRPCUrl) == 0 {
		yamlRelayMinerConfig.QueryNodeGRPCUrl = yamlRelayMinerConfig.TxNodeGRPCUrl
	}

	// Parse the query node GRPC URL
	queryNodeGRPCUrl, err := url.Parse(yamlRelayMinerConfig.QueryNodeGRPCUrl)
	if err != nil {
		return nil, ErrRelayMinerConfigInvalidQueryNodeGRPCUrl.Wrap(err.Error())
	}

	// Check that the network node websocket URL is provided
	if len(yamlRelayMinerConfig.QueryNodeRPCUrl) == 0 {
		return nil, ErrRelayMinerConfigInvalidQueryNodeRPCUrl.Wrap("query node rpc url is required")
	}

	// Parse the rpc URL of the Pocket Node to connect to for subscribing to on-chain events.
	queryNodeRPCUrl, err := url.Parse(yamlRelayMinerConfig.QueryNodeRPCUrl)
	if err != nil {
		return nil, ErrRelayMinerConfigInvalidQueryNodeRPCUrl.Wrap(err.Error())
	}

	if len(yamlRelayMinerConfig.SigningKeyName) == 0 {
		return nil, ErrRelayMinerConfigInvalidSigningKeyName
	}

	if len(yamlRelayMinerConfig.SmtStorePath) == 0 {
		return nil, ErrRelayMinerConfigInvalidSmtStorePath
	}

	if yamlRelayMinerConfig.ProxiedServiceEndpoints == nil {
		return nil, ErrRelayMinerConfigInvalidServiceEndpoint.Wrap("proxied service endpoints are required")
	}

	if len(yamlRelayMinerConfig.ProxiedServiceEndpoints) == 0 {
		return nil, ErrRelayMinerConfigInvalidServiceEndpoint.Wrap("no proxied service endpoints provided")
	}

	// Parse the proxied service endpoints
	proxiedServiceEndpoints := make(map[string]*url.URL, len(yamlRelayMinerConfig.ProxiedServiceEndpoints))
	for serviceId, endpointUrl := range yamlRelayMinerConfig.ProxiedServiceEndpoints {
		endpoint, err := url.Parse(endpointUrl)
		if err != nil {
			return nil, ErrRelayMinerConfigInvalidServiceEndpoint.Wrap(err.Error())
		}
		proxiedServiceEndpoints[serviceId] = endpoint
	}

	relayMinerCMDConfig := &RelayMinerConfig{
		QueryNodeRPCUrl:         queryNodeRPCUrl,
		QueryNodeGRPCUrl:        queryNodeGRPCUrl,
		TxNodeGRPCUrl:           txNodeGRPCUrl,
		SigningKeyName:          yamlRelayMinerConfig.SigningKeyName,
		SmtStorePath:            yamlRelayMinerConfig.SmtStorePath,
		ProxiedServiceEndpoints: proxiedServiceEndpoints,
	}

	return relayMinerCMDConfig, nil
}
