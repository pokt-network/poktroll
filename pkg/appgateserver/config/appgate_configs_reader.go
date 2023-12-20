package config

import (
	"net/url"

	"gopkg.in/yaml.v2"
)

// YAMLAppGateServerConfig is the structure used to unmarshal the AppGateServer config file
// TODO_DOCUMENT(@red-0ne): Add proper README documentation for yaml config files.
type YAMLAppGateServerConfig struct {
	QueryNodeRPCUrl   string `yaml:"query_node_rpc_url"`
	QueryNodeGRPCUrl  string `yaml:"query_node_grpc_url"`
	SigningKey        string `yaml:"signing_key"`
	SelfSigning       bool   `yaml:"self_signing"`
	ListeningEndpoint string `yaml:"listening_endpoint"`
}

// AppGateServerConfig is the structure describing the AppGateServer config
type AppGateServerConfig struct {
	QueryNodeRPCUrl   *url.URL
	QueryNodeGRPCUrl  *url.URL
	SigningKey        string
	SelfSigning       bool
	ListeningEndpoint *url.URL
}

// ParseAppGateServerConfigs parses the stake config file into a AppGateConfig
// NOTE: If SelfSigning is not defined in the config file, it will default to false
func ParseAppGateServerConfigs(configContent []byte) (*AppGateServerConfig, error) {
	var yamlAppGateServerConfig YAMLAppGateServerConfig

	if len(configContent) == 0 {
		return nil, ErrAppGateConfigEmpty
	}

	// Unmarshal the stake config file into a yamlAppGateConfig
	if err := yaml.Unmarshal(configContent, &yamlAppGateServerConfig); err != nil {
		return nil, ErrAppGateConfigUnmarshalYAML.Wrap(err.Error())
	}

	if len(yamlAppGateServerConfig.SigningKey) == 0 {
		return nil, ErrAppGateConfigEmptySigningKey
	}

	if len(yamlAppGateServerConfig.ListeningEndpoint) == 0 {
		return nil, ErrAppGateConfigInvalidListeningEndpoint
	}

	listeningEndpoint, err := url.Parse(yamlAppGateServerConfig.ListeningEndpoint)
	if err != nil {
		return nil, ErrAppGateConfigInvalidListeningEndpoint.Wrap(err.Error())
	}

	if len(yamlAppGateServerConfig.QueryNodeGRPCUrl) == 0 {
		return nil, ErrAppGateConfigInvalidQueryNodeGRPCUrl
	}

	queryNodeGRPCUrl, err := url.Parse(yamlAppGateServerConfig.QueryNodeGRPCUrl)
	if err != nil {
		return nil, ErrAppGateConfigInvalidQueryNodeGRPCUrl.Wrap(err.Error())
	}

	if len(yamlAppGateServerConfig.QueryNodeRPCUrl) == 0 {
		return nil, ErrAppGateConfigInvalidQueryNodeRPCUrl
	}

	queryNodeRPCUrl, err := url.Parse(yamlAppGateServerConfig.QueryNodeRPCUrl)
	if err != nil {
		return nil, ErrAppGateConfigInvalidQueryNodeRPCUrl.Wrap(err.Error())
	}

	// Populate the appGateServerConfig with the values from the yamlAppGateServerConfig
	appGateServerConfig := &AppGateServerConfig{
		QueryNodeRPCUrl:   queryNodeRPCUrl,
		QueryNodeGRPCUrl:  queryNodeGRPCUrl,
		SigningKey:        yamlAppGateServerConfig.SigningKey,
		SelfSigning:       yamlAppGateServerConfig.SelfSigning,
		ListeningEndpoint: listeningEndpoint,
	}

	return appGateServerConfig, nil
}
