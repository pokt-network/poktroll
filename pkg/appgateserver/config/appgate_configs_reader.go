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
	QueryNodeUrl      string `yaml:"query_node_url"`
}

// AppGateServerConfig is the structure describing the AppGateServer config
type AppGateServerConfig struct {
	SelfSigning       bool
	SigningKey        string
	ListeningEndpoint *url.URL
	QueryNodeUrl      *url.URL
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

	if yamlAppGateServerConfig.QueryNodeUrl == "" {
		return nil, ErrAppGateConfigInvalidQueryNodeUrl
	}

	queryNodeUrl, err := url.Parse(yamlAppGateServerConfig.QueryNodeUrl)
	if err != nil {
		return nil, ErrAppGateConfigInvalidQueryNodeUrl.Wrapf("%s", err)
	}

	// Populate the appGateServerConfig with the values from the yamlAppGateServerConfig
	appGateServerConfig := &AppGateServerConfig{
		SelfSigning:       yamlAppGateServerConfig.SelfSigning,
		SigningKey:        yamlAppGateServerConfig.SigningKey,
		ListeningEndpoint: listeningEndpoint,
		QueryNodeUrl:      queryNodeUrl,
	}

	return appGateServerConfig, nil
}
