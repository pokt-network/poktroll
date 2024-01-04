package config

import "net/url"

type ProxyType int

const (
	ProxyTypeHTTP ProxyType = iota
	// TODO: Support other proxy types: HTTPS, TCP, UNIX socket, UDP, QUIC, WebRTC ...
)

// YAMLRelayMinerConfig is the structure used to unmarshal the RelayMiner config file
// TODO_DOCUMENT(@red-0ne): Add proper README documentation for yaml config files
// and update inline comments accordingly.
type YAMLRelayMinerConfig struct {
	PocketNode     YAMLRelayMinerPocketNodeConfig `yaml:"pocket_node"`
	SigningKeyName string                         `yaml:"signing_key_name"`
	SmtStorePath   string                         `yaml:"smt_store_path"`
	Proxies        []YAMLRelayMinerProxyConfig    `yaml:"proxies"`
	Suppliers      []YAMLRelayMinerSupplierConfig `yaml:"suppliers"`
}

// YAMLRelayMinerPocketNodeConfig is the structure used to unmarshal the pocket
// node URLs section of the RelayMiner config file
type YAMLRelayMinerPocketNodeConfig struct {
	QueryNodeRPCUrl  string `yaml:"query_node_rpc_url"`
	QueryNodeGRPCUrl string `yaml:"query_node_grpc_url"`
	TxNodeRPCUrl     string `yaml:"tx_node_rpc_url"`
}

// YAMLRelayMinerProxyConfig is the structure used to unmarshal the proxy
// section of the RelayMiner config file
type YAMLRelayMinerProxyConfig struct {
	ProxyName            string `yaml:"proxy_name"`
	Type                 string `yaml:"type"`
	Host                 string `yaml:"host"`
	XForwardedHostLookup bool   `yaml:"x_forwarded_host_lookup"`
}

// YAMLRelayMinerSupplierConfig is the structure used to unmarshal the supplier
// section of the RelayMiner config file
type YAMLRelayMinerSupplierConfig struct {
	ServiceId     string                              `yaml:"service_id"`
	Type          string                              `yaml:"type"`
	Hosts         []string                            `yaml:"hosts"`
	ServiceConfig YAMLRelayMinerSupplierServiceConfig `yaml:"service_config"`
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
	PocketNode     *RelayMinerPocketNodeConfig
	SigningKeyName string
	SmtStorePath   string
	Proxies        map[string]*RelayMinerProxyConfig
}

// RelayMinerPocketNodeConfig is the structure resulting from parsing the pocket
// node URLs section of the RelayMiner config file
type RelayMinerPocketNodeConfig struct {
	QueryNodeRPCUrl  *url.URL
	QueryNodeGRPCUrl *url.URL
	TxNodeRPCUrl     *url.URL
}

// RelayMinerProxyConfig is the structure resulting from parsing the proxy
// section of the RelayMiner config file.
// Each proxy embeds a map of supplier configs that are associated with it.
// Other proxy types may embed other fields in the future. eg. "https" may
// embed a TLS config.
type RelayMinerProxyConfig struct {
	// ProxyName is the name of the proxy server, used to identify it in the config
	ProxyName string
	// Type is the transport protocol used by the proxy server like (http, https, etc.)
	Type ProxyType
	// Host is the host on which the proxy server will listen for incoming
	// relay requests
	Host string
	// XForwardedHostLookup is a flag that indicates whether the proxy server
	// should lookup the host from the X-Forwarded-Host header before falling
	// back to the Host header.
	XForwardedHostLookup bool
	// Suppliers is a map of serviceIds -> RelayMinerSupplierConfig
	Suppliers map[string]*RelayMinerSupplierConfig
}

// RelayMinerSupplierConfig is the structure resulting from parsing the supplier
// section of the RelayMiner config file.
type RelayMinerSupplierConfig struct {
	// ServiceId is the serviceId corresponding to the current configuration.
	ServiceId string
	// Type is the transport protocol used by the supplier, it must match the
	// type of the proxy it is associated with.
	Type ProxyType
	// Hosts is a list of hosts advertised on-chain by the supplier, the corresponding
	// proxy server will accept relay requests for these hosts.
	Hosts []string
	// ServiceConfig is the config of the service that relays will be proxied to.
	// Other supplier types may embed other fields in the future. eg. "https" may
	// embed a TLS config.
	ServiceConfig *RelayMinerSupplierServiceConfig
}

// RelayMinerSupplierServiceConfig is the structure resulting from parsing the supplier
// service sub-section of the RelayMiner config file.
type RelayMinerSupplierServiceConfig struct {
	// Url is the URL of the service that relays will be proxied to.
	Url *url.URL
	// Authentication is the basic auth structure used to authenticate to the
	// request being proxied from the current proxy server.
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
