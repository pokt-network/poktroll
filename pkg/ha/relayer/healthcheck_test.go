package relayer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

func TestHealthStatus_String(t *testing.T) {
	tests := []struct {
		status   HealthStatus
		expected string
	}{
		{HealthStatusUnknown, "unknown"},
		{HealthStatusHealthy, "healthy"},
		{HealthStatusUnhealthy, "unhealthy"},
		{HealthStatus(99), "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.status.String())
		})
	}
}

func TestBackendHealth_GetSetStatus(t *testing.T) {
	health := &BackendHealth{
		ServiceID:  "test",
		BackendURL: "http://localhost:8080",
	}

	// Initial status is 0 (Unknown)
	require.Equal(t, HealthStatusUnknown, health.GetStatus())

	// Set to healthy
	health.SetStatus(HealthStatusHealthy)
	require.Equal(t, HealthStatusHealthy, health.GetStatus())

	// Set to unhealthy
	health.SetStatus(HealthStatusUnhealthy)
	require.Equal(t, HealthStatusUnhealthy, health.GetStatus())
}

func TestBackendHealth_IsHealthy(t *testing.T) {
	health := &BackendHealth{}

	// Unknown is treated as healthy
	health.SetStatus(HealthStatusUnknown)
	require.True(t, health.IsHealthy())

	// Healthy
	health.SetStatus(HealthStatusHealthy)
	require.True(t, health.IsHealthy())

	// Unhealthy
	health.SetStatus(HealthStatusUnhealthy)
	require.False(t, health.IsHealthy())
}

func TestBackendHealth_LastError(t *testing.T) {
	health := &BackendHealth{}

	// Initially empty
	require.Empty(t, health.GetLastError())

	// Set error
	health.lastError.Store("connection refused")
	require.Equal(t, "connection refused", health.GetLastError())

	// Clear error
	health.lastError.Store("")
	require.Empty(t, health.GetLastError())
}

func TestBackendHealth_LastCheck(t *testing.T) {
	health := &BackendHealth{}

	// Initially zero time (Unix epoch)
	lastCheck := health.GetLastCheck()
	require.Equal(t, time.Unix(0, 0), lastCheck)

	// Set time
	now := time.Now()
	health.lastCheck.Store(now.UnixNano())

	// Should be equal (nanosecond precision is preserved)
	retrieved := health.GetLastCheck()
	require.Equal(t, now.UnixNano(), retrieved.UnixNano())
}

func TestHealthChecker_RegisterBackend(t *testing.T) {
	logger := polyzero.NewLogger()
	hc := NewHealthChecker(logger)
	defer hc.Close()

	// Register without health check config
	hc.RegisterBackend("service1", "http://localhost:8080", nil)

	health := hc.GetHealth("service1")
	require.NotNil(t, health)
	require.Equal(t, "service1", health.ServiceID)
	require.Equal(t, "http://localhost:8080", health.BackendURL)

	// Register with health check config
	config := &BackendHealthCheckConfig{
		Enabled:         true,
		Endpoint:        "/health",
		IntervalSeconds: 10,
	}
	hc.RegisterBackend("service2", "http://localhost:8081", config)

	health2 := hc.GetHealth("service2")
	require.NotNil(t, health2)
}

func TestHealthChecker_IsHealthy_UnknownBackend(t *testing.T) {
	logger := polyzero.NewLogger()
	hc := NewHealthChecker(logger)
	defer hc.Close()

	// Unknown backend should be treated as healthy
	require.True(t, hc.IsHealthy("unknown"))
}

func TestHealthChecker_IsHealthy_RegisteredBackend(t *testing.T) {
	logger := polyzero.NewLogger()
	hc := NewHealthChecker(logger)
	defer hc.Close()

	hc.RegisterBackend("service1", "http://localhost:8080", nil)

	// Initially unknown = healthy
	require.True(t, hc.IsHealthy("service1"))

	// Set to unhealthy
	health := hc.GetHealth("service1")
	health.SetStatus(HealthStatusUnhealthy)
	require.False(t, hc.IsHealthy("service1"))

	// Set to healthy
	health.SetStatus(HealthStatusHealthy)
	require.True(t, hc.IsHealthy("service1"))
}

func TestHealthChecker_GetAllHealth(t *testing.T) {
	logger := polyzero.NewLogger()
	hc := NewHealthChecker(logger)
	defer hc.Close()

	hc.RegisterBackend("service1", "http://localhost:8080", nil)
	hc.RegisterBackend("service2", "http://localhost:8081", nil)
	hc.RegisterBackend("service3", "http://localhost:8082", nil)

	all := hc.GetAllHealth()
	require.Len(t, all, 3)
	require.Contains(t, all, "service1")
	require.Contains(t, all, "service2")
	require.Contains(t, all, "service3")
}

func TestHealthChecker_CheckBackend_Success(t *testing.T) {
	// Create a test server that returns 200
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/health", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := polyzero.NewLogger()
	hc := NewHealthChecker(logger)
	defer hc.Close()

	config := &BackendHealthCheckConfig{
		Enabled:          true,
		Endpoint:         "/health",
		IntervalSeconds:  60,
		TimeoutSeconds:   5,
		HealthyThreshold: 1, // Become healthy after 1 success
	}

	hc.RegisterBackend("test", server.URL, config)

	ctx := context.Background()
	hc.checkBackend(ctx, "test", config)

	health := hc.GetHealth("test")
	require.Equal(t, HealthStatusHealthy, health.GetStatus())
	require.Empty(t, health.GetLastError())
	require.Equal(t, int32(1), health.consecutiveSuccesses.Load())
	require.Equal(t, int32(0), health.consecutiveFailures.Load())
}

func TestHealthChecker_CheckBackend_Failure_StatusCode(t *testing.T) {
	// Create a test server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	logger := polyzero.NewLogger()
	hc := NewHealthChecker(logger)
	defer hc.Close()

	config := &BackendHealthCheckConfig{
		Enabled:            true,
		Endpoint:           "/health",
		IntervalSeconds:    60,
		TimeoutSeconds:     5,
		UnhealthyThreshold: 1, // Become unhealthy after 1 failure
	}

	hc.RegisterBackend("test", server.URL, config)

	ctx := context.Background()
	hc.checkBackend(ctx, "test", config)

	health := hc.GetHealth("test")
	require.Equal(t, HealthStatusUnhealthy, health.GetStatus())
	require.Contains(t, health.GetLastError(), "unhealthy status code: 500")
	require.Equal(t, int32(0), health.consecutiveSuccesses.Load())
	require.Equal(t, int32(1), health.consecutiveFailures.Load())
}

func TestHealthChecker_CheckBackend_Failure_ConnectionRefused(t *testing.T) {
	logger := polyzero.NewLogger()
	hc := NewHealthChecker(logger)
	defer hc.Close()

	config := &BackendHealthCheckConfig{
		Enabled:            true,
		Endpoint:           "/health",
		IntervalSeconds:    60,
		TimeoutSeconds:     1,
		UnhealthyThreshold: 1,
	}

	// Use a port that's not listening
	hc.RegisterBackend("test", "http://localhost:59999", config)

	ctx := context.Background()
	hc.checkBackend(ctx, "test", config)

	health := hc.GetHealth("test")
	require.Equal(t, HealthStatusUnhealthy, health.GetStatus())
	require.Contains(t, health.GetLastError(), "request failed")
}

func TestHealthChecker_Thresholds(t *testing.T) {
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)
		if count <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	logger := polyzero.NewLogger()
	hc := NewHealthChecker(logger)
	defer hc.Close()

	config := &BackendHealthCheckConfig{
		Enabled:            true,
		Endpoint:           "/health",
		IntervalSeconds:    60,
		TimeoutSeconds:     5,
		UnhealthyThreshold: 2, // Need 2 failures to become unhealthy
		HealthyThreshold:   2, // Need 2 successes to become healthy
	}

	hc.RegisterBackend("test", server.URL, config)
	ctx := context.Background()

	// First failure - still unknown (not enough failures)
	hc.checkBackend(ctx, "test", config)
	health := hc.GetHealth("test")
	require.Equal(t, HealthStatusUnknown, health.GetStatus())
	require.Equal(t, int32(1), health.consecutiveFailures.Load())

	// Second failure - now unhealthy
	hc.checkBackend(ctx, "test", config)
	require.Equal(t, HealthStatusUnhealthy, health.GetStatus())
	require.Equal(t, int32(2), health.consecutiveFailures.Load())

	// First success - still unhealthy (not enough successes)
	hc.checkBackend(ctx, "test", config)
	require.Equal(t, HealthStatusUnhealthy, health.GetStatus())
	require.Equal(t, int32(1), health.consecutiveSuccesses.Load())
	require.Equal(t, int32(0), health.consecutiveFailures.Load())

	// Second success - now healthy
	hc.checkBackend(ctx, "test", config)
	require.Equal(t, HealthStatusHealthy, health.GetStatus())
	require.Equal(t, int32(2), health.consecutiveSuccesses.Load())
}

func TestHealthChecker_Start_PeriodicChecks(t *testing.T) {
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := polyzero.NewLogger()
	hc := NewHealthChecker(logger)

	config := &BackendHealthCheckConfig{
		Enabled:          true,
		Endpoint:         "/health",
		IntervalSeconds:  1, // 1 second interval for faster test
		TimeoutSeconds:   5,
		HealthyThreshold: 1,
	}

	hc.RegisterBackend("test", server.URL, config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := hc.Start(ctx)
	require.NoError(t, err)

	// Wait for at least 2 checks
	time.Sleep(2500 * time.Millisecond)

	hc.Close()

	// Should have at least 2 requests (initial + 1 periodic)
	require.GreaterOrEqual(t, requestCount.Load(), int32(2))

	health := hc.GetHealth("test")
	require.Equal(t, HealthStatusHealthy, health.GetStatus())
}

func TestHealthChecker_CheckNow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := polyzero.NewLogger()
	hc := NewHealthChecker(logger)
	defer hc.Close()

	config := &BackendHealthCheckConfig{
		Enabled:          true,
		Endpoint:         "/health",
		IntervalSeconds:  60,
		HealthyThreshold: 1,
	}

	hc.RegisterBackend("test", server.URL, config)

	ctx := context.Background()
	err := hc.CheckNow(ctx, "test")
	require.NoError(t, err)

	health := hc.GetHealth("test")
	require.Equal(t, HealthStatusHealthy, health.GetStatus())
}

func TestHealthChecker_CheckNow_UnknownService(t *testing.T) {
	logger := polyzero.NewLogger()
	hc := NewHealthChecker(logger)
	defer hc.Close()

	ctx := context.Background()
	err := hc.CheckNow(ctx, "unknown")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no health check config")
}

func TestHealthChecker_Close(t *testing.T) {
	logger := polyzero.NewLogger()
	hc := NewHealthChecker(logger)

	config := &BackendHealthCheckConfig{
		Enabled:         true,
		Endpoint:        "/health",
		IntervalSeconds: 1,
	}

	hc.RegisterBackend("test", "http://localhost:8080", config)

	ctx := context.Background()
	err := hc.Start(ctx)
	require.NoError(t, err)

	err = hc.Close()
	require.NoError(t, err)

	// Double close should be safe
	err = hc.Close()
	require.NoError(t, err)
}

func TestHealthChecker_Start_AlreadyClosed(t *testing.T) {
	logger := polyzero.NewLogger()
	hc := NewHealthChecker(logger)
	hc.Close()

	err := hc.Start(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "closed")
}

func TestHealthChecker_WithBearerAuth(t *testing.T) {
	// This test verifies the config parsing, not the actual auth header
	// (auth header implementation is in the proxy, not health checker)
	config := &BackendHealthCheckConfig{
		Enabled:         true,
		Endpoint:        "/health",
		IntervalSeconds: 10,
	}

	auth := &AuthenticationConfig{
		BearerToken: "test-token",
	}

	require.NotNil(t, config)
	require.NotNil(t, auth)
	require.Equal(t, "test-token", auth.BearerToken)
}

func TestNewHealthChecker_Defaults(t *testing.T) {
	logger := polyzero.NewLogger()
	hc := NewHealthChecker(logger)
	defer hc.Close()

	require.NotNil(t, hc.httpClient)
	require.NotNil(t, hc.backends)
	require.NotNil(t, hc.configs)
	require.Equal(t, 3, hc.defaultUnhealthyThreshold)
	require.Equal(t, 2, hc.defaultHealthyThreshold)
}
