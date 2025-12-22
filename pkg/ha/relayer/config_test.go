package relayer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	require.Equal(t, "0.0.0.0:8080", cfg.ListenAddr)
	require.Equal(t, "redis://localhost:6379", cfg.Redis.URL)
	require.Equal(t, "ha:relays", cfg.Redis.StreamPrefix)
	require.Equal(t, int64(100000), cfg.Redis.MaxStreamLen)
	require.Equal(t, ValidationModeOptimistic, cfg.DefaultValidationMode)
	require.Equal(t, int64(30), cfg.DefaultRequestTimeoutSeconds)
	require.Equal(t, int64(10*1024*1024), cfg.DefaultMaxBodySizeBytes)
	require.Equal(t, int64(2), cfg.GracePeriodExtraBlocks)
	require.True(t, cfg.Metrics.Enabled)
	require.Equal(t, "0.0.0.0:9090", cfg.Metrics.Addr)
	require.True(t, cfg.HealthCheck.Enabled)
	require.Equal(t, "0.0.0.0:8081", cfg.HealthCheck.Addr)
}

func TestConfig_Validate_Valid(t *testing.T) {
	cfg := validTestConfig()
	err := cfg.Validate()
	require.NoError(t, err)
}

func TestConfig_Validate_MissingListenAddr(t *testing.T) {
	cfg := validTestConfig()
	cfg.ListenAddr = ""

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "listen_addr is required")
}

func TestConfig_Validate_MissingRedisURL(t *testing.T) {
	cfg := validTestConfig()
	cfg.Redis.URL = ""

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "redis.url is required")
}

func TestConfig_Validate_InvalidRedisURL(t *testing.T) {
	cfg := validTestConfig()
	cfg.Redis.URL = "://invalid"

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid redis.url")
}

func TestConfig_Validate_MissingPocketNodeRPC(t *testing.T) {
	cfg := validTestConfig()
	cfg.PocketNode.QueryNodeRPCUrl = ""

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "query_node_rpc_url is required")
}

func TestConfig_Validate_MissingPocketNodeGRPC(t *testing.T) {
	cfg := validTestConfig()
	cfg.PocketNode.QueryNodeGRPCUrl = ""

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "query_node_grpc_url is required")
}

func TestConfig_Validate_NoServices(t *testing.T) {
	cfg := validTestConfig()
	cfg.Services = nil

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one service must be configured")
}

func TestConfig_Validate_EmptyServices(t *testing.T) {
	cfg := validTestConfig()
	cfg.Services = map[string]ServiceConfig{}

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one service must be configured")
}

func TestConfig_Validate_ServiceMissingBackendURL(t *testing.T) {
	cfg := validTestConfig()
	cfg.Services["ethereum"] = ServiceConfig{
		BackendURL: "",
	}

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "service[ethereum].backend_url is required")
}

func TestConfig_Validate_ServiceInvalidBackendURL(t *testing.T) {
	cfg := validTestConfig()
	cfg.Services["ethereum"] = ServiceConfig{
		BackendURL: "://invalid",
	}

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "service[ethereum].backend_url is invalid")
}

func TestConfig_Validate_ServiceInvalidValidationMode(t *testing.T) {
	cfg := validTestConfig()
	cfg.Services["ethereum"] = ServiceConfig{
		BackendURL:     "http://localhost:8545",
		ValidationMode: "invalid",
	}

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "service[ethereum].validation_mode is invalid")
}

func TestConfig_Validate_InvalidDefaultValidationMode(t *testing.T) {
	cfg := validTestConfig()
	cfg.DefaultValidationMode = "invalid"

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid default_validation_mode")
}

func TestConfig_Validate_RPCTypeBackendMissingURL(t *testing.T) {
	cfg := validTestConfig()
	cfg.Services["ethereum"] = ServiceConfig{
		BackendURL: "http://localhost:8545",
		RPCTypeBackends: map[string]RPCTypeBackendConfig{
			"json-rpc": {BackendURL: ""},
		},
	}

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "rpc_type_backends[json-rpc].backend_url is required")
}

func TestConfig_Validate_RPCTypeBackendInvalidURL(t *testing.T) {
	cfg := validTestConfig()
	cfg.Services["ethereum"] = ServiceConfig{
		BackendURL: "http://localhost:8545",
		RPCTypeBackends: map[string]RPCTypeBackendConfig{
			"json-rpc": {BackendURL: "://invalid"},
		},
	}

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "rpc_type_backends[json-rpc].backend_url is invalid")
}

func TestConfig_Validate_HealthCheckMissingEndpoint(t *testing.T) {
	cfg := validTestConfig()
	cfg.Services["ethereum"] = ServiceConfig{
		BackendURL: "http://localhost:8545",
		HealthCheck: &BackendHealthCheckConfig{
			Enabled:         true,
			Endpoint:        "",
			IntervalSeconds: 10,
		},
	}

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "health_check.endpoint is required")
}

func TestConfig_Validate_HealthCheckInvalidInterval(t *testing.T) {
	cfg := validTestConfig()
	cfg.Services["ethereum"] = ServiceConfig{
		BackendURL: "http://localhost:8545",
		HealthCheck: &BackendHealthCheckConfig{
			Enabled:         true,
			Endpoint:        "/health",
			IntervalSeconds: 0,
		},
	}

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "health_check.interval_seconds must be positive")
}

func TestConfig_Validate_ValidWithAllOptions(t *testing.T) {
	cfg := Config{
		ListenAddr: "0.0.0.0:8080",
		Redis: RedisConfig{
			URL:          "redis://localhost:6379",
			StreamPrefix: "test:relays",
			MaxStreamLen: 50000,
		},
		PocketNode: PocketNodeConfig{
			QueryNodeRPCUrl:  "http://localhost:26657",
			QueryNodeGRPCUrl: "localhost:9090",
		},
		DefaultValidationMode:        ValidationModeEager,
		DefaultRequestTimeoutSeconds: 60,
		DefaultMaxBodySizeBytes:      5 * 1024 * 1024,
		GracePeriodExtraBlocks:       3,
		Services: map[string]ServiceConfig{
			"ethereum": {
				BackendURL:            "http://localhost:8545",
				ValidationMode:        ValidationModeOptimistic,
				RequestTimeoutSeconds: 30,
				MaxBodySizeBytes:      1024 * 1024,
				Headers: map[string]string{
					"X-Custom-Header": "value",
				},
				Authentication: &AuthenticationConfig{
					Username: "user",
					Password: "pass",
				},
				HealthCheck: &BackendHealthCheckConfig{
					Enabled:            true,
					Endpoint:           "/health",
					IntervalSeconds:    10,
					TimeoutSeconds:     5,
					UnhealthyThreshold: 3,
					HealthyThreshold:   2,
				},
				RPCTypeBackends: map[string]RPCTypeBackendConfig{
					"rest": {
						BackendURL: "http://localhost:8546",
						Headers: map[string]string{
							"X-REST-Header": "rest-value",
						},
					},
				},
			},
		},
		Metrics: MetricsConfig{
			Enabled: true,
			Addr:    "0.0.0.0:9090",
		},
		HealthCheck: HealthCheckConfig{
			Enabled: true,
			Addr:    "0.0.0.0:8081",
		},
	}

	err := cfg.Validate()
	require.NoError(t, err)
}

func TestConfig_GetServiceValidationMode(t *testing.T) {
	cfg := validTestConfig()
	cfg.DefaultValidationMode = ValidationModeOptimistic
	cfg.Services["ethereum"] = ServiceConfig{
		BackendURL:     "http://localhost:8545",
		ValidationMode: ValidationModeEager,
	}
	cfg.Services["anvil"] = ServiceConfig{
		BackendURL: "http://localhost:8546",
		// No validation mode set
	}

	// Service with explicit mode
	require.Equal(t, ValidationModeEager, cfg.GetServiceValidationMode("ethereum"))

	// Service without explicit mode (uses default)
	require.Equal(t, ValidationModeOptimistic, cfg.GetServiceValidationMode("anvil"))

	// Unknown service (uses default)
	require.Equal(t, ValidationModeOptimistic, cfg.GetServiceValidationMode("unknown"))
}

func TestConfig_GetServiceTimeout(t *testing.T) {
	cfg := validTestConfig()
	cfg.DefaultRequestTimeoutSeconds = 30
	cfg.Services["ethereum"] = ServiceConfig{
		BackendURL:            "http://localhost:8545",
		RequestTimeoutSeconds: 60,
	}
	cfg.Services["anvil"] = ServiceConfig{
		BackendURL: "http://localhost:8546",
		// No timeout set
	}

	// Service with explicit timeout
	require.Equal(t, 60*time.Second, cfg.GetServiceTimeout("ethereum"))

	// Service without explicit timeout (uses default)
	require.Equal(t, 30*time.Second, cfg.GetServiceTimeout("anvil"))

	// Unknown service (uses default)
	require.Equal(t, 30*time.Second, cfg.GetServiceTimeout("unknown"))
}

func TestConfig_GetServiceMaxBodySize(t *testing.T) {
	cfg := validTestConfig()
	cfg.DefaultMaxBodySizeBytes = 10 * 1024 * 1024
	cfg.Services["ethereum"] = ServiceConfig{
		BackendURL:       "http://localhost:8545",
		MaxBodySizeBytes: 5 * 1024 * 1024,
	}
	cfg.Services["anvil"] = ServiceConfig{
		BackendURL: "http://localhost:8546",
		// No max body size set
	}

	// Service with explicit max body size
	require.Equal(t, int64(5*1024*1024), cfg.GetServiceMaxBodySize("ethereum"))

	// Service without explicit max body size (uses default)
	require.Equal(t, int64(10*1024*1024), cfg.GetServiceMaxBodySize("anvil"))

	// Unknown service (uses default)
	require.Equal(t, int64(10*1024*1024), cfg.GetServiceMaxBodySize("unknown"))
}

func TestValidationMode_Constants(t *testing.T) {
	require.Equal(t, ValidationMode("eager"), ValidationModeEager)
	require.Equal(t, ValidationMode("optimistic"), ValidationModeOptimistic)
}

// validTestConfig returns a minimal valid configuration for testing.
// The service ID is the map key (e.g., "ethereum").
func validTestConfig() Config {
	return Config{
		ListenAddr: "0.0.0.0:8080",
		Redis: RedisConfig{
			URL:          "redis://localhost:6379",
			StreamPrefix: "test:relays",
		},
		PocketNode: PocketNodeConfig{
			QueryNodeRPCUrl:  "http://localhost:26657",
			QueryNodeGRPCUrl: "localhost:9090",
		},
		DefaultValidationMode:        ValidationModeOptimistic,
		DefaultRequestTimeoutSeconds: 30,
		DefaultMaxBodySizeBytes:      10 * 1024 * 1024,
		Services: map[string]ServiceConfig{
			"ethereum": {
				BackendURL: "http://localhost:8545",
			},
		},
	}
}
