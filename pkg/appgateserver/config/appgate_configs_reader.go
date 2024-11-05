package config

import (
	"net/url"

	"gopkg.in/yaml.v2"
)

// YAMLAppGateServerConfig is the structure used to unmarshal the AppGateServer config file
// TODO_MAINNET(@red-0ne): Rename self_signing parameter to `sovereign` in code, configs
// and documentation
type YAMLAppGateServerConfig struct {
	ListeningEndpoint string                         `yaml:"listening_endpoint"`
	Metrics           YAMLAppGateServerMetricsConfig `yaml:"metrics"`
	QueryNodeGRPCUrl  string                         `yaml:"query_node_grpc_url"`
	QueryNodeRPCUrl   string                         `yaml:"query_node_rpc_url"`
	SelfSigning       bool                           `yaml:"self_signing"`
	SigningKey        string                         `yaml:"signing_key"`
	Pprof             YAMLAppGateServerPprofConfig   `yaml:"pprof"`
}

// YAMLAppGateServerMetricsConfig is the structure used to unmarshal the metrics
// section of the AppGateServer config file
type YAMLAppGateServerMetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Addr    string `yaml:"addr"`
}

// YAMLAppGateServerPprofConfig is the structure used to unmarshal the config
// for `pprof`.
type YAMLAppGateServerPprofConfig struct {
	Enabled bool   `yaml:"enabled,omitempty"`
	Addr    string `yaml:"addr,omitempty"`
}

// AppGateServerConfig is the structure describing the AppGateServer config
type AppGateServerConfig struct {
	ListeningEndpoint *url.URL
	Metrics           *AppGateServerMetricsConfig
	QueryNodeGRPCUrl  *url.URL
	QueryNodeRPCUrl   *url.URL
	SelfSigning       bool
	SigningKey        string
	Pprof             *AppGateServerPprofConfig
}

// AppGateServerMetricsConfig is the structure resulting from parsing the metrics
// section of the AppGateServer config file.
type AppGateServerMetricsConfig struct {
	Enabled bool
	Addr    string
}

// AppGateServerPprofConfig is the structure resulting from parsing the pprof
// section of the AppGateServer config file.
type AppGateServerPprofConfig struct {
	Enabled bool
	Addr    string
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

	// Not doing additinal validation on metrics, as the server would not start if the value is invalid,
	// providing the user with a descriptive error message.
	appGateServerConfig.Metrics = &AppGateServerMetricsConfig{
		Enabled: yamlAppGateServerConfig.Metrics.Enabled,
		Addr:    yamlAppGateServerConfig.Metrics.Addr,
	}

	appGateServerConfig.Pprof = &AppGateServerPprofConfig{
		Enabled: yamlAppGateServerConfig.Pprof.Enabled,
		Addr:    yamlAppGateServerConfig.Pprof.Addr,
	}

	return appGateServerConfig, nil
}
