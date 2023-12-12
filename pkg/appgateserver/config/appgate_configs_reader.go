package config

import (
	"net/url"

	"gopkg.in/yaml.v2"
)

// YAMLAppGateServerConfig is the structure used to unmarshal the AppGateServer config file
// TODO_DOCUMENT(@red-0ne): Add proper README documentation for yaml config files.
type YAMLAppGateServerConfig struct {
	SelfSigning       bool   `yaml:"self_signing"`
	SigningKey        string `yaml:"signing_key"`
	ListeningEndpoint string `yaml:"listening_endpoint"`
	QueryNodeGRPCUrl  string `yaml:"query_node_grpc_url"`
	QueryNodeRPCUrl   string `yaml:"query_node_rpc_url"`
}

// AppGateServerConfig is the structure describing the AppGateServer config
type AppGateServerConfig struct {
	SelfSigning       bool
	SigningKey        string
	ListeningEndpoint *url.URL
	QueryNodeGRPCUrl  *url.URL
	QueryNodeRPCUrl   *url.URL
}

// ParseAppGateServerConfigs parses the stake config file into a AppGateConfig
// NOTE: If SelfSigning is not defined in the config file, it will default to false
func ParseAppGateServerConfigs(configContent []byte) (*AppGateServerConfig, error) {
	var yamlAppGateServerConfig YAMLAppGateServerConfig

	// Unmarshal the stake config file into a yamlAppGateConfig
	if err := yaml.Unmarshal(configContent, &yamlAppGateServerConfig); err != nil {
		return nil, ErrAppGateConfigUnmarshalYAML.Wrapf("%s", err)
	}

	if yamlAppGateServerConfig.SigningKey == "" {
		return nil, ErrAppGateConfigEmptySigningKey
	}

	if yamlAppGateServerConfig.ListeningEndpoint == "" {
		return nil, ErrAppGateConfigInvalidListeningEndpoint
	}

	listeningEndpoint, err := url.Parse(yamlAppGateServerConfig.ListeningEndpoint)
	if err != nil {
		return nil, ErrAppGateConfigInvalidListeningEndpoint.Wrapf("%s", err)
	}

	if yamlAppGateServerConfig.QueryNodeGRPCUrl == "" {
		return nil, ErrAppGateConfigInvalidQueryNodeGRPCUrl
	}

	queryNodeGRPCUrl, err := url.Parse(yamlAppGateServerConfig.QueryNodeGRPCUrl)
	if err != nil {
		return nil, ErrAppGateConfigInvalidQueryNodeGRPCUrl.Wrapf("%s", err)
	}

	if yamlAppGateServerConfig.QueryNodeRPCUrl == "" {
		return nil, ErrAppGateConfigInvalidQueryNodeRPCUrl
	}

	queryNodeRPCUrl, err := url.Parse(yamlAppGateServerConfig.QueryNodeRPCUrl)
	if err != nil {
		return nil, ErrAppGateConfigInvalidQueryNodeRPCUrl.Wrapf("%s", err)
	}

	// Populate the appGateServerConfig with the values from the yamlAppGateServerConfig
	appGateServerConfig := &AppGateServerConfig{
		SelfSigning:       yamlAppGateServerConfig.SelfSigning,
		SigningKey:        yamlAppGateServerConfig.SigningKey,
		ListeningEndpoint: listeningEndpoint,
		QueryNodeGRPCUrl:  queryNodeGRPCUrl,
		QueryNodeRPCUrl:   queryNodeRPCUrl,
	}

	return appGateServerConfig, nil
}
