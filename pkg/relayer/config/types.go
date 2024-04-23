package config

import "net/url"

type RelayMinerServerType int

const (
	RelayMinerServerTypeHTTP RelayMinerServerType = iota
	// TODO: Support other RelayMinerServerType:
	// RelayMinerServerTypeHTTPS
	// RelayMinerServerTypeTCP
	// RelayMinerServerTypeUDP
	// RelayMinerServerTypeQUIC
	// RelayMinerServerTypeWebRTC
	// RelayMinerServerTypeUNIXSocket
	// Etc...
)

// YAMLRelayMinerConfig is the structure used to unmarshal the RelayMiner config file
type YAMLRelayMinerConfig struct {
	PocketNode     YAMLRelayMinerPocketNodeConfig `yaml:"pocket_node"`
	SigningKeyName string                         `yaml:"signing_key_name"`
	SmtStorePath   string                         `yaml:"smt_store_path"`
	Metrics        YAMLRelayMinerMetricsConfig    `yaml:"metrics"`
	Suppliers      []YAMLRelayMinerSupplierConfig `yaml:"suppliers"`
}

// YAMLRelayMinerPocketNodeConfig is the structure used to unmarshal the pocket
// node URLs section of the RelayMiner config file.
type YAMLRelayMinerPocketNodeConfig struct {
	QueryNodeRPCUrl  string `yaml:"query_node_rpc_url"`
	QueryNodeGRPCUrl string `yaml:"query_node_grpc_url"`
	TxNodeRPCUrl     string `yaml:"tx_node_rpc_url"`
}

// YAMLRelayMinerMetricsConfig is the structure used to unmarshal the metrics
// section of the RelayMiner config file.
type YAMLRelayMinerMetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Addr    string `yaml:"addr"`
}

// YAMLRelayMinerSupplierConfig is the structure used to unmarshal the supplier
// section of the RelayMiner config file
type YAMLRelayMinerSupplierConfig struct {
	ServiceId            string                              `yaml:"service_id"`
	ListenUrl            string                              `yaml:"listen_url"`
	XForwardedHostLookup bool                                `yaml:"x_forwarded_host_lookup"`
	ServiceConfig        YAMLRelayMinerSupplierServiceConfig `yaml:"service_config"`
}

// YAMLRelayMinerSupplierServiceConfig is the structure used to unmarshal the supplier
// service sub-section of the RelayMiner config file.
type YAMLRelayMinerSupplierServiceConfig struct {
	BackendUrl               string                                      `yaml:"backend_url"`
	Authentication           YAMLRelayMinerSupplierServiceAuthentication `yaml:"authentication,omitempty"`
	Headers                  map[string]string                           `yaml:"headers,omitempty"`
	PubliclyExposedEndpoints []string                                    `yaml:"publicly_exposed_endpoints"`
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
	PocketNode     *RelayMinerPocketNodeConfig
	Servers        map[string]*RelayMinerServerConfig
	Metrics        *RelayMinerMetricsConfig
	SigningKeyName string
	SmtStorePath   string
}

// RelayMinerPocketNodeConfig is the structure resulting from parsing the pocket
// node URLs section of the RelayMiner config file
type RelayMinerPocketNodeConfig struct {
	QueryNodeRPCUrl  *url.URL
	QueryNodeGRPCUrl *url.URL
	TxNodeRPCUrl     *url.URL
}

// RelayMinerServerConfig is the structure resulting from parsing the supplier's
// server section of the RelayMiner config file.
// Each server section embeds a map of supplier configs that are associated with it.
// TODO_IMPROVE: Other server types may embed other fields in the future; eg. "https" may embed a TLS config.
type RelayMinerServerConfig struct {
	// ServerType is the transport protocol used by the server like (http, https, etc.)
	ServerType RelayMinerServerType
	// ListenAddress is the host on which the relay miner server will listen
	// for incoming relay requests
	ListenAddress string
	// XForwardedHostLookup is a flag that indicates whether the relay miner server
	// should lookup the host from the X-Forwarded-Host header before falling
	// back to the Host header.
	XForwardedHostLookup bool
	// SupplierConfigsMap is a map of serviceIds -> RelayMinerSupplierConfig
	SupplierConfigsMap map[string]*RelayMinerSupplierConfig
}

// RelayMinerMetricsConfig is the structure resulting from parsing the metrics
// section of the RelayMiner config file
type RelayMinerMetricsConfig struct {
	Enabled bool
	Addr    string
}

// RelayMinerSupplierConfig is the structure resulting from parsing the supplier
// section of the RelayMiner config file.
type RelayMinerSupplierConfig struct {
	// ServiceId is the serviceId corresponding to the current configuration.
	ServiceId string
	// ServerType is the transport protocol used by the supplier, it must match the
	// type of the relay miner server it is associated with.
	ServerType RelayMinerServerType
	// PubliclyExposedEndpoints is a list of hosts advertised on-chain by the supplier,
	// the corresponding relay miner server will accept relay requests for these hosts.
	PubliclyExposedEndpoints []string
	// ServiceConfig is the config of the service that relays will be proxied to.
	// Other supplier types may embed other fields in the future. eg. "https" may
	// embed a TLS config.
	ServiceConfig *RelayMinerSupplierServiceConfig
}

// RelayMinerSupplierServiceConfig is the structure resulting from parsing the supplier
// service sub-section of the RelayMiner config file.
type RelayMinerSupplierServiceConfig struct {
	// BackendUrl is the URL of the service that relays will be proxied to.
	BackendUrl *url.URL
	// Authentication is the basic auth structure used to authenticate to the
	// request being proxied from the current relay miner server.
	// If the service the relay requests are forwarded to requires basic auth
	// then this field must be populated.
	// TODO_TECHDEBT(@red-0ne): Pass the authentication to the service instance
	// when the relay request is forwarded to it.
	Authentication *RelayMinerSupplierServiceAuthentication
	// Headers is a map of headers to be used for other authentication means.
	// If the service the relay requests are forwarded to requires header based
	// authentication then this field must be populated accordingly.
	// For example: { "Authorization": "Bearer <token>" }
	// TODO_TECHDEBT(@red-0ne): Add these headers to the forwarded request
	// before sending it to the service instance.
	Headers map[string]string
}

// RelayMinerSupplierServiceAuthentication is the structure resulting from parsing
// the supplier service basic auth of the RelayMiner config file when the
// supplier is of type "http"
type RelayMinerSupplierServiceAuthentication struct {
	Username string
	Password string
}
