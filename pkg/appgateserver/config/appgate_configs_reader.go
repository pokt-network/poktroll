package config

import (
	"net/url"

	"gopkg.in/yaml.v2"
)

// YAMLAppGateConfig is the structure used to unmarshal the AppGateServer config file
type YAMLAppGateConfig struct {
	SelfSigning       bool   `yaml:"self_signing"`
	SigningKey        string `yaml:"signing_key"`
	ListeningEndpoint string `yaml:"listening_endpoint"`
	QueryNodeUrl      string `yaml:"query_node_url"`
}

// AppGateConfig is the structure describing the AppGateServer config
type AppGateConfig struct {
	SelfSigning       bool
	SigningKey        string
	ListeningEndpoint *url.URL
	QueryNodeUrl      *url.URL
}

// ParseAppGateConfigs parses the stake config file into a AppGateConfig
// NOTE: If SelfSigning is not defined in the config file, it will default to false
func ParseAppGateConfigs(configContent []byte) (*AppGateConfig, error) {
	var yamlAppGateConfig YAMLAppGateConfig

	// Unmarshal the stake config file into a yamlAppGateConfig
	if err := yaml.Unmarshal(configContent, &yamlAppGateConfig); err != nil {
		return nil, ErrAppGateConfigUnmarshalYAML.Wrapf("%s", err)
	}

	if yamlAppGateConfig.SigningKey == "" {
		return nil, ErrAppGateConfigEmptySigningKey
	}

	if yamlAppGateConfig.ListeningEndpoint == "" {
		return nil, ErrAppGateConfigInvalidListeningEndpoint
	}

	listeningEndpoint, err := url.Parse(yamlAppGateConfig.ListeningEndpoint)
	if err != nil {
		return nil, ErrAppGateConfigInvalidListeningEndpoint.Wrapf("%s", err)
	}

	if yamlAppGateConfig.QueryNodeUrl == "" {
		return nil, ErrAppGateConfigInvalidQueryNodeUrl
	}

	queryNodeUrl, err := url.Parse(yamlAppGateConfig.QueryNodeUrl)
	if err != nil {
		return nil, ErrAppGateConfigInvalidQueryNodeUrl.Wrapf("%s", err)
	}

	// Populate the supplierServiceConfig
	appGateCMDConfig := &AppGateConfig{
		SelfSigning:       yamlAppGateConfig.SelfSigning,
		SigningKey:        yamlAppGateConfig.SigningKey,
		ListeningEndpoint: listeningEndpoint,
		QueryNodeUrl:      queryNodeUrl,
	}

	return appGateCMDConfig, nil
}
