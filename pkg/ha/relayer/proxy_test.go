package relayer

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/ha/transport"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

// mockPublisher implements transport.MinedRelayPublisher for testing.
type mockPublisher struct {
	messages      []*transport.MinedRelayMessage
	publishCalled atomic.Int32
	publishErr    error
}

func (m *mockPublisher) Publish(ctx context.Context, msg *transport.MinedRelayMessage) error {
	m.publishCalled.Add(1)
	if m.publishErr != nil {
		return m.publishErr
	}
	m.messages = append(m.messages, msg)
	return nil
}

func (m *mockPublisher) PublishBatch(ctx context.Context, msgs []*transport.MinedRelayMessage) error {
	for _, msg := range msgs {
		if err := m.Publish(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockPublisher) Close() error {
	return nil
}

func TestNewProxyServer(t *testing.T) {
	logger := polyzero.NewLogger()
	config := &Config{
		ListenAddr:                   "0.0.0.0:8080",
		DefaultRequestTimeoutSeconds: 30,
		DefaultMaxBodySizeBytes:      1024 * 1024,
		Services: map[string]ServiceConfig{
			"ethereum": {
				BackendURL: "http://localhost:8545",
			},
		},
	}

	hc := NewHealthChecker(logger)
	defer hc.Close()

	publisher := &mockPublisher{}

	proxy, err := NewProxyServer(logger, config, hc, publisher)
	require.NoError(t, err)
	require.NotNil(t, proxy)
	defer proxy.Close()

	// Verify backend URL was parsed
	require.Contains(t, proxy.backendURLs, "ethereum")
	require.Equal(t, "localhost:8545", proxy.backendURLs["ethereum"].Host)
}

func TestNewProxyServer_InvalidBackendURL(t *testing.T) {
	logger := polyzero.NewLogger()
	config := &Config{
		ListenAddr: "0.0.0.0:8080",
		Services: map[string]ServiceConfig{
			"bad": {
				BackendURL: "://invalid",
			},
		},
	}

	hc := NewHealthChecker(logger)
	defer hc.Close()

	_, err := NewProxyServer(logger, config, hc, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid backend URL")
}

func TestProxyServer_HandleRelay_Success(t *testing.T) {
	// Create backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		require.Equal(t, "test request", string(body))
		w.Header().Set("X-Backend-Response", "true")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("backend response"))
	}))
	defer backend.Close()

	logger := polyzero.NewLogger()
	config := &Config{
		ListenAddr:                   "0.0.0.0:0",
		DefaultRequestTimeoutSeconds: 30,
		DefaultMaxBodySizeBytes:      1024 * 1024,
		DefaultValidationMode:        ValidationModeOptimistic,
		Services: map[string]ServiceConfig{
			"ethereum": {
				BackendURL: backend.URL,
			},
		},
	}

	hc := NewHealthChecker(logger)
	hc.RegisterBackend("ethereum", backend.URL, nil)
	hc.GetHealth("ethereum").SetStatus(HealthStatusHealthy)
	defer hc.Close()

	publisher := &mockPublisher{}

	proxy, err := NewProxyServer(logger, config, hc, publisher)
	require.NoError(t, err)
	defer proxy.Close()

	// Create test request
	req := httptest.NewRequest(http.MethodPost, "/ethereum", bytes.NewReader([]byte("test request")))
	req.Header.Set("Pocket-Service-Id", "ethereum")
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	w := httptest.NewRecorder()
	proxy.handleRelay(w, req)

	// Verify response
	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	require.Equal(t, "backend response", string(body))
	require.Equal(t, "true", resp.Header.Get("X-Backend-Response"))
}

func TestProxyServer_HandleRelay_MissingServiceID(t *testing.T) {
	logger := polyzero.NewLogger()
	config := &Config{
		ListenAddr: "0.0.0.0:0",
		Services: map[string]ServiceConfig{
			"ethereum": {
				BackendURL: "http://localhost:8545",
			},
		},
	}

	hc := NewHealthChecker(logger)
	defer hc.Close()

	proxy, _ := NewProxyServer(logger, config, hc, nil)
	defer proxy.Close()

	// Request without service ID header and path
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()

	proxy.handleRelay(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestProxyServer_HandleRelay_UnknownService(t *testing.T) {
	logger := polyzero.NewLogger()
	config := &Config{
		ListenAddr: "0.0.0.0:0",
		Services: map[string]ServiceConfig{
			"ethereum": {
				BackendURL: "http://localhost:8545",
			},
		},
	}

	hc := NewHealthChecker(logger)
	defer hc.Close()

	proxy, _ := NewProxyServer(logger, config, hc, nil)
	defer proxy.Close()

	req := httptest.NewRequest(http.MethodPost, "/unknown", nil)
	req.Header.Set("Pocket-Service-Id", "unknown")
	w := httptest.NewRecorder()

	proxy.handleRelay(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestProxyServer_HandleRelay_UnhealthyBackend(t *testing.T) {
	logger := polyzero.NewLogger()
	config := &Config{
		ListenAddr: "0.0.0.0:0",
		Services: map[string]ServiceConfig{
			"ethereum": {
				BackendURL: "http://localhost:8545",
			},
		},
	}

	hc := NewHealthChecker(logger)
	hc.RegisterBackend("ethereum", "http://localhost:8545", nil)
	hc.GetHealth("ethereum").SetStatus(HealthStatusUnhealthy)
	defer hc.Close()

	proxy, _ := NewProxyServer(logger, config, hc, nil)
	defer proxy.Close()

	req := httptest.NewRequest(http.MethodPost, "/ethereum", nil)
	req.Header.Set("Pocket-Service-Id", "ethereum")
	w := httptest.NewRecorder()

	proxy.handleRelay(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestProxyServer_HandleRelay_BodyTooLarge(t *testing.T) {
	logger := polyzero.NewLogger()
	config := &Config{
		ListenAddr:              "0.0.0.0:0",
		DefaultMaxBodySizeBytes: 100, // Very small limit
		Services: map[string]ServiceConfig{
			"ethereum": {
				BackendURL: "http://localhost:8545",
			},
		},
	}

	hc := NewHealthChecker(logger)
	hc.RegisterBackend("ethereum", "http://localhost:8545", nil)
	hc.GetHealth("ethereum").SetStatus(HealthStatusHealthy)
	defer hc.Close()

	proxy, _ := NewProxyServer(logger, config, hc, nil)
	defer proxy.Close()

	// Create request with body larger than limit
	largeBody := make([]byte, 200)
	req := httptest.NewRequest(http.MethodPost, "/ethereum", bytes.NewReader(largeBody))
	req.Header.Set("Pocket-Service-Id", "ethereum")
	w := httptest.NewRecorder()

	proxy.handleRelay(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)
}

func TestProxyServer_HandleRelay_BackendError(t *testing.T) {
	// Backend that refuses connections
	logger := polyzero.NewLogger()
	config := &Config{
		ListenAddr:                   "0.0.0.0:0",
		DefaultRequestTimeoutSeconds: 1, // Short timeout
		DefaultMaxBodySizeBytes:      1024 * 1024,
		Services: map[string]ServiceConfig{
			"ethereum": {
				BackendURL: "http://localhost:59999", // Non-existent
			},
		},
	}

	hc := NewHealthChecker(logger)
	hc.RegisterBackend("ethereum", "http://localhost:59999", nil)
	hc.GetHealth("ethereum").SetStatus(HealthStatusHealthy)
	defer hc.Close()

	proxy, _ := NewProxyServer(logger, config, hc, nil)
	defer proxy.Close()

	req := httptest.NewRequest(http.MethodPost, "/ethereum", bytes.NewReader([]byte("test")))
	req.Header.Set("Pocket-Service-Id", "ethereum")
	w := httptest.NewRecorder()

	proxy.handleRelay(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusBadGateway, resp.StatusCode)
}

func TestProxyServer_HandleRelay_WithHeaders(t *testing.T) {
	var receivedHeaders http.Header
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	logger := polyzero.NewLogger()
	config := &Config{
		ListenAddr:              "0.0.0.0:0",
		DefaultMaxBodySizeBytes: 1024 * 1024,
		Services: map[string]ServiceConfig{
			"ethereum": {
				BackendURL: backend.URL,
				Headers: map[string]string{
					"X-Custom-Header": "custom-value",
				},
			},
		},
	}

	hc := NewHealthChecker(logger)
	hc.RegisterBackend("ethereum", backend.URL, nil)
	hc.GetHealth("ethereum").SetStatus(HealthStatusHealthy)
	defer hc.Close()

	proxy, _ := NewProxyServer(logger, config, hc, nil)
	defer proxy.Close()

	req := httptest.NewRequest(http.MethodPost, "/ethereum", nil)
	req.Header.Set("Pocket-Service-Id", "ethereum")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Pocket-Session-Id", "session123")
	w := httptest.NewRecorder()

	proxy.handleRelay(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify custom header was set
	require.Equal(t, "custom-value", receivedHeaders.Get("X-Custom-Header"))

	// Verify Content-Type was copied
	require.Equal(t, "application/json", receivedHeaders.Get("Content-Type"))

	// Verify Pocket-* headers were copied
	require.Equal(t, "session123", receivedHeaders.Get("Pocket-Session-Id"))
}

func TestProxyServer_HandleRelay_WithBasicAuth(t *testing.T) {
	var authHeader string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	logger := polyzero.NewLogger()
	config := &Config{
		ListenAddr:              "0.0.0.0:0",
		DefaultMaxBodySizeBytes: 1024 * 1024,
		Services: map[string]ServiceConfig{
			"ethereum": {
				BackendURL: backend.URL,
				Authentication: &AuthenticationConfig{
					Username: "user",
					Password: "pass",
				},
			},
		},
	}

	hc := NewHealthChecker(logger)
	hc.RegisterBackend("ethereum", backend.URL, nil)
	hc.GetHealth("ethereum").SetStatus(HealthStatusHealthy)
	defer hc.Close()

	proxy, _ := NewProxyServer(logger, config, hc, nil)
	defer proxy.Close()

	req := httptest.NewRequest(http.MethodPost, "/ethereum", nil)
	req.Header.Set("Pocket-Service-Id", "ethereum")
	w := httptest.NewRecorder()

	proxy.handleRelay(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify Basic auth was set
	require.Contains(t, authHeader, "Basic ")
}

func TestProxyServer_HandleRelay_WithBearerAuth(t *testing.T) {
	var authHeader string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	logger := polyzero.NewLogger()
	config := &Config{
		ListenAddr:              "0.0.0.0:0",
		DefaultMaxBodySizeBytes: 1024 * 1024,
		Services: map[string]ServiceConfig{
			"ethereum": {
				BackendURL: backend.URL,
				Authentication: &AuthenticationConfig{
					BearerToken: "my-secret-token",
				},
			},
		},
	}

	hc := NewHealthChecker(logger)
	hc.RegisterBackend("ethereum", backend.URL, nil)
	hc.GetHealth("ethereum").SetStatus(HealthStatusHealthy)
	defer hc.Close()

	proxy, _ := NewProxyServer(logger, config, hc, nil)
	defer proxy.Close()

	req := httptest.NewRequest(http.MethodPost, "/ethereum", nil)
	req.Header.Set("Pocket-Service-Id", "ethereum")
	w := httptest.NewRecorder()

	proxy.handleRelay(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "Bearer my-secret-token", authHeader)
}

func TestProxyServer_HandleRelay_RPCTypeBackend(t *testing.T) {
	var hitBackend string

	jsonRpcBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitBackend = "json-rpc"
		w.WriteHeader(http.StatusOK)
	}))
	defer jsonRpcBackend.Close()

	restBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitBackend = "rest"
		w.WriteHeader(http.StatusOK)
	}))
	defer restBackend.Close()

	logger := polyzero.NewLogger()
	config := &Config{
		ListenAddr:              "0.0.0.0:0",
		DefaultMaxBodySizeBytes: 1024 * 1024,
		Services: map[string]ServiceConfig{
			"ethereum": {
				BackendURL: jsonRpcBackend.URL, // Default
				RPCTypeBackends: map[string]RPCTypeBackendConfig{
					"rest": {
						BackendURL: restBackend.URL,
					},
				},
			},
		},
	}

	hc := NewHealthChecker(logger)
	hc.RegisterBackend("ethereum", jsonRpcBackend.URL, nil)
	hc.GetHealth("ethereum").SetStatus(HealthStatusHealthy)
	defer hc.Close()

	proxy, _ := NewProxyServer(logger, config, hc, nil)
	defer proxy.Close()

	// Test JSON-RPC (default)
	req := httptest.NewRequest(http.MethodPost, "/ethereum", nil)
	req.Header.Set("Pocket-Service-Id", "ethereum")
	w := httptest.NewRecorder()
	proxy.handleRelay(w, req)
	require.Equal(t, "json-rpc", hitBackend)

	// Test REST
	req2 := httptest.NewRequest(http.MethodPost, "/ethereum", nil)
	req2.Header.Set("Pocket-Service-Id", "ethereum")
	req2.Header.Set("Rpc-Type", "rest")
	w2 := httptest.NewRecorder()
	proxy.handleRelay(w2, req2)
	require.Equal(t, "rest", hitBackend)
}

func TestProxyServer_ExtractServiceID(t *testing.T) {
	logger := polyzero.NewLogger()
	config := &Config{
		ListenAddr: "0.0.0.0:0",
		Services:   map[string]ServiceConfig{},
	}

	hc := NewHealthChecker(logger)
	defer hc.Close()

	proxy, _ := NewProxyServer(logger, config, hc, nil)
	defer proxy.Close()

	tests := []struct {
		name     string
		setup    func(*http.Request)
		expected string
	}{
		{
			name: "from header",
			setup: func(r *http.Request) {
				r.Header.Set("Pocket-Service-Id", "ethereum")
			},
			expected: "ethereum",
		},
		{
			name: "from path single segment",
			setup: func(r *http.Request) {
				r.URL.Path = "/anvil"
			},
			expected: "anvil",
		},
		{
			name: "from path multiple segments",
			setup: func(r *http.Request) {
				r.URL.Path = "/polygon/v1/query"
			},
			expected: "polygon",
		},
		{
			name: "header takes precedence",
			setup: func(r *http.Request) {
				r.Header.Set("Pocket-Service-Id", "header-service")
				r.URL.Path = "/path-service"
			},
			expected: "header-service",
		},
		{
			name:     "empty",
			setup:    func(r *http.Request) {},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			tt.setup(req)
			serviceID := proxy.extractServiceID(req)
			require.Equal(t, tt.expected, serviceID)
		})
	}
}

func TestProxyServer_SetBlockHeight(t *testing.T) {
	logger := polyzero.NewLogger()
	config := &Config{
		ListenAddr: "0.0.0.0:0",
		Services:   map[string]ServiceConfig{},
	}

	hc := NewHealthChecker(logger)
	defer hc.Close()

	proxy, _ := NewProxyServer(logger, config, hc, nil)
	defer proxy.Close()

	require.Equal(t, int64(0), proxy.currentBlockHeight.Load())

	proxy.SetBlockHeight(100)
	require.Equal(t, int64(100), proxy.currentBlockHeight.Load())

	proxy.SetBlockHeight(200)
	require.Equal(t, int64(200), proxy.currentBlockHeight.Load())
}

func TestProxyServer_PublishMinedRelay(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response"))
	}))
	defer backend.Close()

	logger := polyzero.NewLogger()
	config := &Config{
		ListenAddr:                   "0.0.0.0:0",
		DefaultMaxBodySizeBytes:      1024 * 1024,
		DefaultValidationMode:        ValidationModeEager,
		DefaultRequestTimeoutSeconds: 30,
		Services: map[string]ServiceConfig{
			"ethereum": {
				BackendURL: backend.URL,
			},
		},
	}

	hc := NewHealthChecker(logger)
	hc.RegisterBackend("ethereum", backend.URL, nil)
	hc.GetHealth("ethereum").SetStatus(HealthStatusHealthy)
	defer hc.Close()

	publisher := &mockPublisher{}

	proxy, _ := NewProxyServer(logger, config, hc, publisher)
	proxy.SetBlockHeight(1000)
	defer proxy.Close()

	req := httptest.NewRequest(http.MethodPost, "/ethereum", bytes.NewReader([]byte("test")))
	req.Header.Set("Pocket-Service-Id", "ethereum")
	req.Header.Set("Pocket-Supplier-Address", "pokt1supplier123")
	req.Header.Set("Pocket-Session-Id", "session123")
	req.Header.Set("Pocket-Application-Address", "pokt1app123")
	w := httptest.NewRecorder()

	proxy.handleRelay(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Give background goroutine time to publish
	time.Sleep(100 * time.Millisecond)

	require.Equal(t, int32(1), publisher.publishCalled.Load())
	require.Len(t, publisher.messages, 1)

	msg := publisher.messages[0]
	require.Equal(t, "ethereum", msg.ServiceId)
	require.Equal(t, "pokt1supplier123", msg.SupplierOperatorAddress)
	require.Equal(t, "session123", msg.SessionId)
	require.Equal(t, "pokt1app123", msg.ApplicationAddress)
	require.Equal(t, int64(1000), msg.ArrivalBlockHeight)
}

func TestProxyServer_SendError(t *testing.T) {
	logger := polyzero.NewLogger()
	config := &Config{
		ListenAddr: "0.0.0.0:0",
		Services:   map[string]ServiceConfig{},
	}

	hc := NewHealthChecker(logger)
	defer hc.Close()

	proxy, _ := NewProxyServer(logger, config, hc, nil)
	defer proxy.Close()

	w := httptest.NewRecorder()
	proxy.sendError(w, http.StatusBadRequest, "test error")

	resp := w.Result()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	require.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	body, _ := io.ReadAll(resp.Body)
	var errResp map[string]string
	err := json.Unmarshal(body, &errResp)
	require.NoError(t, err)
	require.Equal(t, "test error", errResp["error"])
}

func TestProxyServer_Close(t *testing.T) {
	logger := polyzero.NewLogger()
	config := &Config{
		ListenAddr: "0.0.0.0:0",
		Services:   map[string]ServiceConfig{},
	}

	hc := NewHealthChecker(logger)
	defer hc.Close()

	proxy, _ := NewProxyServer(logger, config, hc, nil)

	err := proxy.Close()
	require.NoError(t, err)

	// Double close should be safe
	err = proxy.Close()
	require.NoError(t, err)
}

func TestProxyServer_Start_AlreadyClosed(t *testing.T) {
	logger := polyzero.NewLogger()
	config := &Config{
		ListenAddr: "0.0.0.0:0",
		Services:   map[string]ServiceConfig{},
	}

	hc := NewHealthChecker(logger)
	defer hc.Close()

	proxy, _ := NewProxyServer(logger, config, hc, nil)
	proxy.Close()

	err := proxy.Start(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "closed")
}

func TestProxyServer_Start_AlreadyStarted(t *testing.T) {
	logger := polyzero.NewLogger()
	config := &Config{
		ListenAddr:                   "127.0.0.1:0",
		DefaultRequestTimeoutSeconds: 30,
		Services:                     map[string]ServiceConfig{},
	}

	hc := NewHealthChecker(logger)
	defer hc.Close()

	proxy, _ := NewProxyServer(logger, config, hc, nil)
	defer proxy.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := proxy.Start(ctx)
	require.NoError(t, err)

	err = proxy.Start(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already started")
}

// TestIsStreamingResponse tests the streaming response detection.
func TestIsStreamingResponse(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expected    bool
	}{
		{
			name:        "SSE content type",
			contentType: "text/event-stream",
			expected:    true,
		},
		{
			name:        "SSE with charset",
			contentType: "text/event-stream; charset=utf-8",
			expected:    true,
		},
		{
			name:        "NDJSON content type",
			contentType: "application/x-ndjson",
			expected:    true,
		},
		{
			name:        "NDJSON with charset",
			contentType: "application/x-ndjson; charset=utf-8",
			expected:    true,
		},
		{
			name:        "JSON is not streaming",
			contentType: "application/json",
			expected:    false,
		},
		{
			name:        "HTML is not streaming",
			contentType: "text/html",
			expected:    false,
		},
		{
			name:        "Empty content type",
			contentType: "",
			expected:    false,
		},
		{
			name:        "Case insensitive SSE",
			contentType: "TEXT/EVENT-STREAM",
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				Header: http.Header{},
			}
			if tt.contentType != "" {
				resp.Header.Set("Content-Type", tt.contentType)
			}
			result := isStreamingResponse(resp)
			require.Equal(t, tt.expected, result)
		})
	}
}

// TestProxyServer_HandleRelay_Streaming_SSE tests SSE streaming responses.
func TestProxyServer_HandleRelay_Streaming_SSE(t *testing.T) {
	// Create a backend that streams SSE events
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		require.True(t, ok, "ResponseWriter should support Flusher")

		// Send SSE events
		events := []string{
			"data: {\"message\": \"Hello\"}\n",
			"data: {\"message\": \"World\"}\n",
			"data: {\"message\": \"Done\"}\n",
		}

		for _, event := range events {
			w.Write([]byte(event))
			flusher.Flush()
		}
	}))
	defer backend.Close()

	logger := polyzero.NewLogger()
	config := &Config{
		ListenAddr:                   "0.0.0.0:0",
		DefaultRequestTimeoutSeconds: 30,
		DefaultMaxBodySizeBytes:      1024 * 1024,
		DefaultValidationMode:        ValidationModeOptimistic,
		Services: map[string]ServiceConfig{
			"llm": {
				BackendURL: backend.URL,
			},
		},
	}

	hc := NewHealthChecker(logger)
	hc.RegisterBackend("llm", backend.URL, nil)
	hc.GetHealth("llm").SetStatus(HealthStatusHealthy)
	defer hc.Close()

	publisher := &mockPublisher{}

	proxy, err := NewProxyServer(logger, config, hc, publisher)
	require.NoError(t, err)
	defer proxy.Close()

	req := httptest.NewRequest(http.MethodPost, "/llm", bytes.NewReader([]byte("test request")))
	req.Header.Set("Pocket-Service-Id", "llm")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	proxy.handleRelay(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Verify all events are in the response
	require.Contains(t, bodyStr, "Hello")
	require.Contains(t, bodyStr, "World")
	require.Contains(t, bodyStr, "Done")
}

// TestProxyServer_HandleRelay_Streaming_NDJSON tests NDJSON streaming responses.
func TestProxyServer_HandleRelay_Streaming_NDJSON(t *testing.T) {
	// Create a backend that streams NDJSON
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		require.True(t, ok, "ResponseWriter should support Flusher")

		// Send NDJSON lines
		lines := []string{
			`{"id": 1, "token": "The"}`,
			`{"id": 2, "token": " quick"}`,
			`{"id": 3, "token": " fox"}`,
		}

		for _, line := range lines {
			w.Write([]byte(line + "\n"))
			flusher.Flush()
		}
	}))
	defer backend.Close()

	logger := polyzero.NewLogger()
	config := &Config{
		ListenAddr:                   "0.0.0.0:0",
		DefaultRequestTimeoutSeconds: 30,
		DefaultMaxBodySizeBytes:      1024 * 1024,
		DefaultValidationMode:        ValidationModeOptimistic,
		Services: map[string]ServiceConfig{
			"llm": {
				BackendURL: backend.URL,
			},
		},
	}

	hc := NewHealthChecker(logger)
	hc.RegisterBackend("llm", backend.URL, nil)
	hc.GetHealth("llm").SetStatus(HealthStatusHealthy)
	defer hc.Close()

	publisher := &mockPublisher{}

	proxy, err := NewProxyServer(logger, config, hc, publisher)
	require.NoError(t, err)
	defer proxy.Close()

	req := httptest.NewRequest(http.MethodPost, "/llm", bytes.NewReader([]byte(`{"prompt": "test"}`)))
	req.Header.Set("Pocket-Service-Id", "llm")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	proxy.handleRelay(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "application/x-ndjson", resp.Header.Get("Content-Type"))

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Verify all tokens are in the response
	require.Contains(t, bodyStr, "The")
	require.Contains(t, bodyStr, "quick")
	require.Contains(t, bodyStr, "fox")

	// Verify it's proper NDJSON (each line is valid JSON)
	lines := bytes.Split(body, []byte("\n"))
	jsonLineCount := 0
	for _, line := range lines {
		if len(line) > 0 {
			var obj map[string]interface{}
			err := json.Unmarshal(line, &obj)
			require.NoError(t, err, "Each line should be valid JSON")
			jsonLineCount++
		}
	}
	require.Equal(t, 3, jsonLineCount)
}

// TestProxyServer_HandleRelay_NonStreaming_JSON tests that JSON responses are not streamed.
func TestProxyServer_HandleRelay_NonStreaming_JSON(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "success"}`))
	}))
	defer backend.Close()

	logger := polyzero.NewLogger()
	config := &Config{
		ListenAddr:                   "0.0.0.0:0",
		DefaultRequestTimeoutSeconds: 30,
		DefaultMaxBodySizeBytes:      1024 * 1024,
		DefaultValidationMode:        ValidationModeOptimistic,
		Services: map[string]ServiceConfig{
			"ethereum": {
				BackendURL: backend.URL,
			},
		},
	}

	hc := NewHealthChecker(logger)
	hc.RegisterBackend("ethereum", backend.URL, nil)
	hc.GetHealth("ethereum").SetStatus(HealthStatusHealthy)
	defer hc.Close()

	proxy, err := NewProxyServer(logger, config, hc, nil)
	require.NoError(t, err)
	defer proxy.Close()

	req := httptest.NewRequest(http.MethodPost, "/ethereum", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Pocket-Service-Id", "ethereum")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	proxy.handleRelay(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	body, _ := io.ReadAll(resp.Body)
	require.Equal(t, `{"result": "success"}`, string(body))
}

// TestProxyServer_HandleRelay_Streaming_PublishesRelay tests that streaming relays are published.
func TestProxyServer_HandleRelay_Streaming_PublishesRelay(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher, _ := w.(http.Flusher)
		w.Write([]byte("data: chunk1\n"))
		flusher.Flush()
		w.Write([]byte("data: chunk2\n"))
		flusher.Flush()
	}))
	defer backend.Close()

	logger := polyzero.NewLogger()
	config := &Config{
		ListenAddr:                   "0.0.0.0:0",
		DefaultRequestTimeoutSeconds: 30,
		DefaultMaxBodySizeBytes:      1024 * 1024,
		DefaultValidationMode:        ValidationModeEager,
		Services: map[string]ServiceConfig{
			"llm": {
				BackendURL: backend.URL,
			},
		},
	}

	hc := NewHealthChecker(logger)
	hc.RegisterBackend("llm", backend.URL, nil)
	hc.GetHealth("llm").SetStatus(HealthStatusHealthy)
	defer hc.Close()

	publisher := &mockPublisher{}

	proxy, err := NewProxyServer(logger, config, hc, publisher)
	require.NoError(t, err)
	proxy.SetBlockHeight(500)
	defer proxy.Close()

	req := httptest.NewRequest(http.MethodPost, "/llm", bytes.NewReader([]byte("test")))
	req.Header.Set("Pocket-Service-Id", "llm")
	req.Header.Set("Pocket-Supplier-Address", "pokt1supplier")
	req.Header.Set("Pocket-Session-Id", "session-streaming")

	w := httptest.NewRecorder()
	proxy.handleRelay(w, req)

	require.Equal(t, http.StatusOK, w.Result().StatusCode)

	// Give background goroutine time to publish
	time.Sleep(100 * time.Millisecond)

	require.Equal(t, int32(1), publisher.publishCalled.Load())
	require.Len(t, publisher.messages, 1)

	msg := publisher.messages[0]
	require.Equal(t, "llm", msg.ServiceId)
	require.Equal(t, "session-streaming", msg.SessionId)
	require.Equal(t, int64(500), msg.ArrivalBlockHeight)
}
