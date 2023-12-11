package config

import (
	"net/url"

	"gopkg.in/yaml.v2"
)

// YAMLAppGateServerConfig is the structure used to unmarshal the AppGateServer config file
// TODO_DOCUMENT(@red-0ne): Add proper README documentation for yaml config files.
type YAMLAppGateServerConfig struct {
	SelfSigning            bool   `yaml:"self_signing"`
	SigningKey             string `yaml:"signing_key"`
	ListeningEndpoint      string `yaml:"listening_endpoint"`
	QueryNodeGRPCUrl       string `yaml:"query_node_grpc_url"`
	GRPCInsecure           bool   `yaml:"grpc_insecure"`
	PocketNodeWebsocketUrl string `yaml:"pocket_node_websocket_url"`
}

// AppGateServerConfig is the structure describing the AppGateServer config
type AppGateServerConfig struct {
	SelfSigning            bool
	SigningKey             string
	ListeningEndpoint      *url.URL
	QueryNodeGRPCUrl       *url.URL
	GRPCInsecure           bool
	PocketNodeWebsocketUrl *url.URL
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

	if yamlAppGateServerConfig.PocketNodeWebsocketUrl == "" {
		return nil, ErrAppGateConfigInvalidPocketNodeWebsocketUrl
	}

	pocketNodeWebsocketUrl, err := url.Parse(yamlAppGateServerConfig.PocketNodeWebsocketUrl)
	if err != nil {
		return nil, ErrAppGateConfigInvalidPocketNodeWebsocketUrl.Wrapf("%s", err)
	}

	// Populate the appGateServerConfig with the values from the yamlAppGateServerConfig
	appGateServerConfig := &AppGateServerConfig{
		SelfSigning:            yamlAppGateServerConfig.SelfSigning,
		SigningKey:             yamlAppGateServerConfig.SigningKey,
		ListeningEndpoint:      listeningEndpoint,
		QueryNodeGRPCUrl:       queryNodeGRPCUrl,
		GRPCInsecure:           yamlAppGateServerConfig.GRPCInsecure,
		PocketNodeWebsocketUrl: pocketNodeWebsocketUrl,
	}

	return appGateServerConfig, nil
}
