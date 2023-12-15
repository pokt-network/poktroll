package config

import "net/url"

// YAMLRelayMinerConfig is the structure used to unmarshal the RelayMiner config file
// TODO_DOCUMENT(@red-0ne): Add proper README documentation for yaml config files.
type YAMLRelayMinerConfig struct {
	SigningKeyName string                         `yaml:"signing_key_name"`
	SmtStorePath   string                         `yaml:"smt_store_path"`
	Pocket         YAMLRelayMinerPocketConfig     `yaml:"pocket"`
	Proxies        []YAMLRelayMinerProxyConfig    `yaml:"proxies"`
	Suppliers      []YAMLRelayMinerSupplierConfig `yaml:"suppliers"`
}

// YAMLRelayMinerPocketConfig is the structure used to unmarshal the pocket
// node URLs section of the RelayMiner config file
type YAMLRelayMinerPocketConfig struct {
	QueryNodeGRPCUrl string `yaml:"query_node_grpc_url"`
	TxNodeGRPCUrl    string `yaml:"tx_node_grpc_url"`
	QueryNodeRPCUrl  string `yaml:"query_node_rpc_url"`
}

// YAMLRelayMinerProxyConfig is the structure used to unmarshal the proxy
// section of the RelayMiner config file
type YAMLRelayMinerProxyConfig struct {
	Name string `yaml:"name"`
	Host string `yaml:"host"`
	Type string `yaml:"type"`
}

// YAMLRelayMinerSupplierConfig is the structure used to unmarshal the supplier
// section of the RelayMiner config file
type YAMLRelayMinerSupplierConfig struct {
	Name          string                              `yaml:"name"`
	Type          string                              `yaml:"type"`
	ServiceConfig YAMLRelayMinerSupplierServiceConfig `yaml:"service_config"`
	Hosts         []string                            `yaml:"hosts"`
	ProxyNames    []string                            `yaml:"proxy_names"`
}

// YAMLRelayMinerSupplierServiceConfig is the structure used to unmarshal the supplier
// service sub-section of the RelayMiner config file
type YAMLRelayMinerSupplierServiceConfig struct {
	Url            string                                      `yaml:"url"`
	Authentication YAMLRelayMinerSupplierServiceAuthentication `yaml:"authentication,omitempty"`
	Headers        map[string]string                           `yaml:"headers,omitempty"`
}

// YAMLRelayMinerSupplierServiceAuthentication is the structure used to unmarshal
// the supplier service basic auth of the RelayMiner config file when the
// supplier is of type "http"
type YAMLRelayMinerSupplierServiceAuthentication struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

// RelayMinerConfig is the structure describing the RelayMiner config
type RelayMinerConfig struct {
	SigningKeyName string
	SmtStorePath   string
	Pocket         *RelayMinerPocketConfig
	Proxies        map[string]*RelayMinerProxyConfig
}

// RelayMinerPocketConfig is the structure resulting from parsing the pocket
// node URLs section of the RelayMiner config file
type RelayMinerPocketConfig struct {
	QueryNodeGRPCUrl *url.URL
	TxNodeGRPCUrl    *url.URL
	QueryNodeRPCUrl  *url.URL
}

// RelayMinerProxyConfig is the structure resulting from parsing the proxy
// section of the RelayMiner config file.
// Each proxy embeds a map of supplier configs that are associated with it.
// Other proxy types may embed other fields in the future. eg. "https" may
// embed a TLS config.
type RelayMinerProxyConfig struct {
	// Name is the name of the proxy server, used to identify it in the config
	Name string
	// Host is the host on which the proxy server will listen for incoming
	// relay requests
	Host string
	// Type is the transport protocol used by the proxy server like (http, https, etc.)
	Type string
	// Suppliers is a map of serviceIds -> RelayMinerSupplierConfig
	Suppliers map[string]*RelayMinerSupplierConfig
}

// RelayMinerSupplierConfig is the structure resulting from parsing the supplier
// section of the RelayMiner config file.
type RelayMinerSupplierConfig struct {
	// Name is the serviceId corresponding to the current configuration.
	Name string
	// Type is the transport protocol used by the supplier, it must match the
	// type of the proxy it is associated with.
	Type string
	// Hosts is a list of hosts advertised on-chain by the supplier, the corresponding
	// proxy server will accept relay requests for these hosts.
	Hosts []string
	// ServiceConfig is the config of the service that relays will be proxied to.
	ServiceConfig *RelayMinerSupplierServiceConfig
}

// RelayMinerSupplierServiceConfig is the structure resulting from parsing the supplier
// service sub-section of the RelayMiner config file.
// If the supplier is of type "http", it may embed a basic auth structure and
// a map of headers to be used for other authentication means.
// Other supplier types may embed other fields in the future. eg. "https" may
// embed a TLS config.
type RelayMinerSupplierServiceConfig struct {
	// Url is the URL of the service that relays will be proxied to.
	Url *url.URL
	// Authentication is the basic auth structure used to authenticate to the
	// request being proxied from the current proxy server.
	Authentication *RelayMinerSupplierServiceAuthentication
	// Headers is a map of headers to be used for other authentication means.
	Headers map[string]string
}

// RelayMinerSupplierServiceAuthentication is the structure resulting from parsing
// the supplier service basic auth of the RelayMiner config file when the
// supplier is of type "http"
type RelayMinerSupplierServiceAuthentication struct {
	Username string
	Password string
}
