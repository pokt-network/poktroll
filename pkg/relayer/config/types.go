package config

import (
	"net/url"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

type RelayMinerServerType int

const (
	RelayMinerServerTypeHTTP RelayMinerServerType = iota
	// TODO_FUTURE: Support other RelayMinerServerType:
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
	DefaultSigningKeyNames       []string                       `yaml:"default_signing_key_names"`
	DefaultRequestTimeoutSeconds uint64                         `yaml:"default_request_timeout_seconds"`
	DefaultMaxBodySize           string                         `yaml:"default_max_body_size"`
	Metrics                      YAMLRelayMinerMetricsConfig    `yaml:"metrics"`
	PocketNode                   YAMLRelayMinerPocketNodeConfig `yaml:"pocket_node"`
	Pprof                        YAMLRelayMinerPprofConfig      `yaml:"pprof"`
	SmtStorePath                 string                         `yaml:"smt_store_path"`
	Suppliers                    []YAMLRelayMinerSupplierConfig `yaml:"suppliers"`
	Ping                         YAMLRelayMinerPingConfig       `yaml:"ping"`
	EnableOverServicing          bool                           `yaml:"enable_over_servicing"`

	// TODO_IMPROVE: Add a EnableErrorPropagation flag to control whether errors (i.e. non-2XX HTTP status codes)
	// are propagated back to the client or masked as internal errors.
	//
	// The risk of (the current default behaviour) where RelayMiners propagate
	// non-2XX HTTP status codes back to the client is that it PATH (or other clients)
	// will sanction them. This is non-ideal but will indirectly lead to better Supplier
	// behaviour in the long run until the following is implemented and communicated.
	//
	// See this discussion for more details: https://github.com/pokt-network/poktroll/pull/1608/files#r2175684381
	// EnableErrorPropagation bool `yaml:"enable_error_propagation"`
}

// YAMLRelayMinerPingConfig represents the configuration to expose a ping server.
type YAMLRelayMinerPingConfig struct {
	Enabled bool `yaml:"enabled"`
	// Addr is the address to bind to (format: 'hostname:port') where 'hostname' can be a DNS name or an IP
	Addr string `yaml:"addr"`
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
	ListenUrl             string                                         `yaml:"listen_url"`
	ServiceConfig         YAMLRelayMinerSupplierServiceConfig            `yaml:"service_config"`
	RPCTypeServiceConfigs map[string]YAMLRelayMinerSupplierServiceConfig `yaml:"rpc_type_service_configs"`
	ServiceId             string                                         `yaml:"service_id"`
	SigningKeyNames       []string                                       `yaml:"signing_key_names"`
	RequestTimeoutSeconds uint64                                         `yaml:"request_timeout_seconds"`
	MaxBodySize           string                                         `yaml:"max_body_size"`
	XForwardedHostLookup  bool                                           `yaml:"x_forwarded_host_lookup"`
}

// YAMLRelayMinerSupplierServiceConfig is the structure used to unmarshal the supplier
// service sub-section of the RelayMiner config file.
type YAMLRelayMinerSupplierServiceConfig struct {
	Authentication       YAMLRelayMinerSupplierServiceAuthentication `yaml:"authentication,omitempty"`
	BackendUrl           string                                      `yaml:"backend_url"`
	Headers              map[string]string                           `yaml:"headers,omitempty"`
	ForwardPocketHeaders bool                                        `yaml:"forward_pocket_headers"`
}

// YAMLRelayMinerSupplierServiceAuthentication is the structure used to unmarshal
// the supplier service basic auth of the RelayMiner config file when the
// supplier is of type "http"
type YAMLRelayMinerSupplierServiceAuthentication struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

// YAMLRelayMinerPprofConfig is the structure used to unmarshal the config
// for `pprof`.
type YAMLRelayMinerPprofConfig struct {
	Enabled bool   `yaml:"enabled,omitempty"`
	Addr    string `yaml:"addr,omitempty"`
}

// RelayMinerConfig is the structure describing the RelayMiner config
type RelayMinerConfig struct {
	DefaultSigningKeyNames       []string
	DefaultRequestTimeoutSeconds uint64
	DefaultMaxBodySize           int64
	Metrics                      *RelayMinerMetricsConfig
	PocketNode                   *RelayMinerPocketNodeConfig
	Pprof                        *RelayMinerPprofConfig
	Servers                      map[string]*RelayMinerServerConfig
	SmtStorePath                 string
	Ping                         *RelayMinerPingConfig
	EnableOverServicing          bool
}

// TODO_TECHDEBT(@red-0ne): Remove this structure altogether. See the discussion here for ref:
// https://github.com/pokt-network/poktroll/pull/1037/files#r1928599958
// RelayMinerPingConfig is the structure resulting from parsing the ping
// server configuration.
type RelayMinerPingConfig struct {
	Enabled bool
	// Addr is the address to bind to (format: hostname:port) where 'hostname' can be a DNS name or an IP
	Addr string
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
	// MaxBodySize sets the largest request or response body size (in bytes) that the RelayMiner will accept for this service.
	MaxBodySize int64
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

	// TODO_TECHDEBT(@commoddity): Rename to DefaultServiceConfig.
	// The `ServiceConfig` field will be renamed to `DefaultServiceConfig` in the future to make
	// its reponsibility more explicit. It is kept named `ServiceConfig` in the current release
	// for backwards compatibility.
	//
	// ServiceConfig is the default config of the service that relays will be
	// proxied to. It is required in all cases to provide a service config for the supplier.
	//
	// It is used if a request has no matching service config in RPCTypeServiceConfigs.
	ServiceConfig *RelayMinerSupplierServiceConfig

	// RPCTypeServiceConfigs is a map of RPC types to service configs.
	// Used to select an alternate service config for a given RPC type.
	//
	// For example, if a service exposes two separate endpoints for REST and JSON-RPC,
	// it can be configured to handle them separately by providing two RPC-type specific
	// service configs.
	//
	// If the supplier is configured to handle multiple RPC types, the service config
	// will be selected based on the `Rpc-Type` header of the request, which is set
	// by the client.
	//
	// If the RPC type is not present in the map, the default service config is used.
	RPCTypeServiceConfigs map[sharedtypes.RPCType]*RelayMinerSupplierServiceConfig

	// SigningKeyNames: a list of key names that can accept relays for that supplier.
	// If empty, we copy the values from `DefaultSigningKeyNames`.
	SigningKeyNames []string

	// RequestTimeoutSeconds is the timeout in seconds for the relay requests forwarded
	// to the backend service.
	RequestTimeoutSeconds uint64
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
	Authentication *RelayMinerSupplierServiceAuthentication
	// Headers is a map of headers to be used for other authentication means.
	// If the service the relay requests are forwarded to requires header based
	// authentication then this field must be populated accordingly.
	// For example: { "Authorization": "Bearer <token>" }
	Headers map[string]string
	// ForwardPocketHeaders toggles if headers prefixed with 'Pocket-' should be forwarded to
	// the backend service servicing the relay requests.
	ForwardPocketHeaders bool
}

// RelayMinerSupplierServiceAuthentication is the structure resulting from parsing
// the supplier service basic auth of the RelayMiner config file when the
// supplier is of type "http".
type RelayMinerSupplierServiceAuthentication struct {
	Username string
	Password string
}

// RelayMinerPprofConfig is the structure resulting from parsing the pprof config
// section of a RelayMiner config.
type RelayMinerPprofConfig struct {
	Enabled bool
	Addr    string
}
