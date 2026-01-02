package relayer

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	sdktypes "github.com/pokt-network/shannon-sdk/types"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"

	"github.com/pokt-network/poktroll/pkg/ha/cache"
	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/ha/transport"
	"github.com/pokt-network/poktroll/pkg/polylog"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
)

// httpStreamingTypes contains Content-Type values that indicate streaming responses.
// These are used to detect when a backend response should be streamed to the client
// rather than buffered entirely.
var httpStreamingTypes = []string{
	"text/event-stream",    // Server-Sent Events (SSE)
	"application/x-ndjson", // Newline-Delimited JSON (common for LLM APIs)
}

// publishTask holds the data needed for publishing a mined relay.
type publishTask struct {
	reqBody            []byte
	respBody           []byte
	arrivalBlockHeight int64
	serviceID          string
	supplierAddr       string
	sessionID          string
	applicationAddr    string
}

// ProxyServer handles incoming relay requests and forwards them to backends.
type ProxyServer struct {
	logger         polylog.Logger
	config         *Config
	healthChecker  *HealthChecker
	publisher      transport.MinedRelayPublisher
	validator      RelayValidator
	relayProcessor RelayProcessor
	responseSigner *ResponseSigner
	supplierCache  *cache.SupplierCache

	// HTTP client for backend requests
	httpClient *http.Client

	// HTTP server
	server *http.Server

	// Parsed backend URLs
	backendURLs map[string]*url.URL

	// Current block height (from block subscriber)
	currentBlockHeight atomic.Int64

	// Supplier address for this proxy instance
	supplierAddress string

	// Publish worker pool - uses server context, not request context
	publishCh   chan publishTask
	publishCtx  context.Context
	numWorkers  int

	// gRPC proxy handler (legacy transparent proxy - deprecated)
	grpcHandler    *GRPCProxyHandler
	grpcWebWrapper *GRPCWebWrapper

	// gRPC relay service (proper relay protocol over gRPC)
	grpcRelayService *RelayGRPCService
	grpcRelayServer  *grpc.Server // gRPC server for the relay service

	// Lifecycle
	mu       sync.Mutex
	started  bool
	closed   bool
	cancelFn context.CancelFunc
	wg       sync.WaitGroup
}

// NewProxyServer creates a new HTTP proxy server.
func NewProxyServer(
	logger polylog.Logger,
	config *Config,
	healthChecker *HealthChecker,
	publisher transport.MinedRelayPublisher,
) (*ProxyServer, error) {
	// Parse backend URLs - use the first available backend for each service
	backendURLs := make(map[string]*url.URL)
	for id, svc := range config.Services {
		// Find the first available backend (prefer "rest" if available)
		var backendURL string
		if backend, ok := svc.Backends["rest"]; ok {
			backendURL = backend.URL
		} else {
			// Use the first backend found
			for _, backend := range svc.Backends {
				backendURL = backend.URL
				break
			}
		}
		if backendURL == "" {
			return nil, fmt.Errorf("no backend configured for service %s", id)
		}
		parsed, err := url.Parse(backendURL)
		if err != nil {
			return nil, fmt.Errorf("invalid backend URL for service %s: %w", id, err)
		}
		backendURLs[id] = parsed
	}

	// Default to 4 publish workers with a buffer of 10000 tasks
	numWorkers := 4
	bufferSize := 10000

	proxy := &ProxyServer{
		logger:        logging.ForComponent(logger, logging.ComponentProxyServer),
		config:        config,
		healthChecker: healthChecker,
		publisher:     publisher,
		backendURLs:   backendURLs,
		publishCh:     make(chan publishTask, bufferSize),
		numWorkers:    numWorkers,
		httpClient: &http.Client{
			// Don't follow redirects - pass them through
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  true, // Don't modify content encoding
			},
		},
	}

	return proxy, nil
}

// Start starts the HTTP proxy server.
func (p *ProxyServer) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return fmt.Errorf("proxy server is closed")
	}
	if p.started {
		p.mu.Unlock()
		return fmt.Errorf("proxy server already started")
	}

	p.started = true
	ctx, p.cancelFn = context.WithCancel(ctx)
	p.publishCtx = ctx // Store the server context for publish workers
	p.mu.Unlock()

	// Start publish workers - these use the server context, not request contexts
	for i := 0; i < p.numWorkers; i++ {
		p.wg.Add(1)
		go p.publishWorker(ctx, i)
	}
	p.logger.Info().Int("workers", p.numWorkers).Msg("started publish workers")

	// Create HTTP server with h2c (HTTP/2 cleartext) support for native gRPC
	mux := http.NewServeMux()
	mux.HandleFunc("/", p.handleRelay)

	// Configure HTTP/2 server for h2c (HTTP/2 without TLS)
	// This is required for native gRPC clients connecting without TLS
	h2s := &http2.Server{
		MaxConcurrentStreams: 250, // Allow up to 250 concurrent streams per connection
	}

	// Wrap the handler with h2c to support both HTTP/1.1 and HTTP/2 cleartext
	h2cHandler := h2c.NewHandler(mux, h2s)

	p.server = &http.Server{
		Addr:         p.config.ListenAddr,
		Handler:      h2cHandler,
		ReadTimeout:  time.Duration(p.config.DefaultRequestTimeoutSeconds+5) * time.Second,
		WriteTimeout: time.Duration(p.config.DefaultRequestTimeoutSeconds+10) * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.logger.Info().Str(logging.FieldListenAddr, p.config.ListenAddr).Msg("starting HTTP proxy server")

		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			p.logger.Error().Err(err).Msg("HTTP server error")
		}
	}()

	// Wait for shutdown signal
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := p.server.Shutdown(shutdownCtx); err != nil {
			p.logger.Error().Err(err).Msg("error during server shutdown")
		}
	}()

	return nil
}

// publishWorker consumes tasks from the publish channel and publishes them to Redis.
// This runs with the server context, not request contexts, avoiding context cancellation issues.
func (p *ProxyServer) publishWorker(ctx context.Context, workerID int) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-p.publishCh:
			if !ok {
				return
			}
			p.executePublish(ctx, task)
		}
	}
}

// handleRelay handles incoming relay requests.
func (p *ProxyServer) handleRelay(w http.ResponseWriter, r *http.Request) {
	// Health check endpoint - bypasses relay validation for load balancers
	if r.URL.Path == "/health" || r.URL.Path == "/healthz" || r.URL.Path == "/ready" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","block_height":%d}`, p.currentBlockHeight.Load())
		return
	}

	startTime := time.Now()
	activeConnections.Inc()
	defer activeConnections.Dec()

	// Check for WebSocket upgrade request
	if IsWebSocketUpgrade(r) {
		p.WebSocketHandler()(w, r)
		return
	}

	// Check for gRPC-Web requests (HTTP/1.1 browser clients)
	if p.grpcWebWrapper != nil && p.grpcWebWrapper.IsGRPCWebRequest(r) {
		p.grpcWebWrapper.ServeHTTP(w, r)
		return
	}

	// Check for native gRPC requests (HTTP/2 with application/grpc content type)
	// Uses the new relay service that properly handles RelayRequest/RelayResponse protocol
	if IsGRPCRequest(r) && p.grpcRelayServer != nil {
		p.grpcRelayServer.ServeHTTP(w, r)
		return
	}

	// Read request body first (we need it to extract service ID from relay request)
	maxBodySize := p.config.DefaultMaxBodySizeBytes
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize+1))
	if err != nil {
		p.sendError(w, http.StatusBadRequest, "failed to read request body")
		relaysRejected.WithLabelValues("unknown", "read_body_error").Inc()
		return
	}

	if int64(len(body)) > maxBodySize {
		p.sendError(w, http.StatusRequestEntityTooLarge, "request body too large")
		relaysRejected.WithLabelValues("unknown", "body_too_large").Inc()
		return
	}

	// Parse the relay request protobuf to extract service ID and payload
	// SECURITY: Only valid RelayRequest protobufs are accepted - raw HTTP requests are rejected
	relayRequest, serviceID, poktHTTPRequest, parseErr := p.parseRelayRequest(body)
	if parseErr != nil {
		// SECURITY FIX: Reject all non-relay traffic with proper error
		// This prevents unsigned/raw HTTP requests from being proxied
		p.logger.Debug().
			Err(parseErr).
			Msg("rejected request: not a valid RelayRequest protobuf")
		p.sendError(w, http.StatusBadRequest, "invalid relay request: body must be a valid RelayRequest protobuf")
		relaysRejected.WithLabelValues("unknown", "invalid_relay_request").Inc()
		return
	}
	if serviceID == "" {
		p.sendError(w, http.StatusBadRequest, "missing service ID in relay request")
		relaysRejected.WithLabelValues("unknown", "missing_service_id").Inc()
		return
	}

	relaysReceived.WithLabelValues(serviceID).Inc()

	// Check if service exists
	svcConfig, ok := p.config.Services[serviceID]
	if !ok {
		p.sendError(w, http.StatusNotFound, fmt.Sprintf("unknown service: %s", serviceID))
		relaysRejected.WithLabelValues(serviceID, "unknown_service").Inc()
		return
	}

	// Check supplier state if we have a valid relay request and supplier cache
	if relayRequest != nil && p.supplierCache != nil {
		supplierOperatorAddr := relayRequest.Meta.SupplierOperatorAddress
		if supplierOperatorAddr != "" {
			supplierState, cacheErr := p.supplierCache.GetSupplierState(r.Context(), supplierOperatorAddr)
			if cacheErr != nil {
				// Cache error - fail-open behavior depends on cache configuration
				p.logger.Warn().
					Err(cacheErr).
					Str(logging.FieldSupplier, supplierOperatorAddr).
					Str(logging.FieldServiceID, serviceID).
					Msg("failed to check supplier state in cache")
				// Continue processing - fail-open behavior is handled by the cache
			} else if supplierState == nil {
				// Supplier not found in cache
				p.logger.Info().
					Str(logging.FieldSupplier, supplierOperatorAddr).
					Str(logging.FieldServiceID, serviceID).
					Msg("supplier not found in cache")
				p.sendError(w, http.StatusServiceUnavailable, fmt.Sprintf("supplier %s not registered with any miner", supplierOperatorAddr))
				relaysRejected.WithLabelValues(serviceID, "supplier_not_found").Inc()
				return
			} else if !supplierState.IsActive() {
				// Supplier exists but not active (e.g., unstaking)
				p.logger.Info().
					Str(logging.FieldSupplier, supplierOperatorAddr).
					Str(logging.FieldServiceID, serviceID).
					Str("status", supplierState.Status).
					Msg("supplier not active")
				p.sendError(w, http.StatusServiceUnavailable, fmt.Sprintf("supplier %s is %s", supplierOperatorAddr, supplierState.Status))
				relaysRejected.WithLabelValues(serviceID, "supplier_inactive").Inc()
				return
			} else if len(supplierState.Services) == 0 {
				// Supplier active but has no services registered
				p.logger.Info().
					Str(logging.FieldSupplier, supplierOperatorAddr).
					Str(logging.FieldServiceID, serviceID).
					Msg("supplier has no services registered")
				p.sendError(w, http.StatusServiceUnavailable, fmt.Sprintf("supplier %s has no services registered", supplierOperatorAddr))
				relaysRejected.WithLabelValues(serviceID, "no_services").Inc()
				return
			} else if !supplierState.IsActiveForService(serviceID) {
				// Supplier active but not for this service
				p.logger.Info().
					Str(logging.FieldSupplier, supplierOperatorAddr).
					Str(logging.FieldServiceID, serviceID).
					Str("registered_services", fmt.Sprintf("%v", supplierState.Services)).
					Msg("supplier not staked for service")
				p.sendError(w, http.StatusServiceUnavailable, fmt.Sprintf("supplier %s not staked for service %s (staked for: %v)", supplierOperatorAddr, serviceID, supplierState.Services))
				relaysRejected.WithLabelValues(serviceID, "wrong_service").Inc()
				return
			}
			p.logger.Debug().
				Str(logging.FieldSupplier, supplierOperatorAddr).
				Str(logging.FieldServiceID, serviceID).
				Msg("supplier is active for service")
		}
	}

	// Check backend health
	if !p.healthChecker.IsHealthy(serviceID) {
		p.sendError(w, http.StatusServiceUnavailable, "backend unhealthy")
		relaysRejected.WithLabelValues(serviceID, "backend_unhealthy").Inc()
		return
	}

	// Check service-specific body size limit
	serviceMaxBodySize := p.config.GetServiceMaxBodySize(serviceID)
	if int64(len(body)) > serviceMaxBodySize {
		p.sendError(w, http.StatusRequestEntityTooLarge, "request body too large for service")
		relaysRejected.WithLabelValues(serviceID, "body_too_large").Inc()
		return
	}

	requestBodySize.WithLabelValues(serviceID).Observe(float64(len(body)))

	// Pin block height at arrival time (for grace period calculation)
	arrivalBlockHeight := p.currentBlockHeight.Load()

	// Get validation mode
	validationMode := p.config.GetServiceValidationMode(serviceID)

	// For eager validation, validate before forwarding
	if validationMode == ValidationModeEager {
		if validationErr := p.validateRelayRequest(r.Context(), r, body, arrivalBlockHeight); validationErr != nil {
			p.sendError(w, http.StatusForbidden, validationErr.Error())
			relaysRejected.WithLabelValues(serviceID, "validation_failed").Inc()
			validationFailures.WithLabelValues(serviceID, "signature").Inc()
			return
		}
		validationLatency.WithLabelValues(serviceID, "eager").Observe(time.Since(startTime).Seconds())
	}

	// Forward request to backend (handles both streaming and non-streaming)
	// Use the parsed POKTHTTPRequest if available, otherwise fall back to raw body
	backendStart := time.Now()
	respBody, respHeaders, respStatus, isStreaming, err := p.forwardToBackendWithStreaming(r.Context(), r, body, serviceID, &svcConfig, poktHTTPRequest, w)
	backendLatency.WithLabelValues(serviceID).Observe(time.Since(backendStart).Seconds())

	if err != nil {
		// Only send error response if we haven't started streaming yet
		if !isStreaming {
			p.sendError(w, http.StatusBadGateway, "backend error")
		}
		relaysRejected.WithLabelValues(serviceID, "backend_error").Inc()
		return
	}

	// For non-streaming responses, build and return signed RelayResponse
	if !isStreaming {
		responseBodySize.WithLabelValues(serviceID).Observe(float64(len(respBody)))

		// Check if this is a valid relay request that requires a signed response
		if relayRequest != nil && p.responseSigner != nil {
			// Build and sign the RelayResponse
			_, signedResponseBz, signErr := p.responseSigner.BuildAndSignRelayResponseFromBody(
				relayRequest,
				respBody,
				respHeaders,
				respStatus,
			)
			if signErr != nil {
				p.logger.Error().
					Err(signErr).
					Str(logging.FieldServiceID, serviceID).
					Msg("failed to sign relay response")
				p.sendError(w, http.StatusInternalServerError, "failed to sign response")
				relaysRejected.WithLabelValues(serviceID, "signing_error").Inc()
				return
			}

			// Send the signed RelayResponse protobuf
			w.Header().Set("Content-Type", "application/x-protobuf")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write(signedResponseBz); err != nil {
				p.logger.Debug().Err(err).Msg("failed to write signed response body")
			}

			p.logger.Debug().
				Str(logging.FieldServiceID, serviceID).
				Int("response_size", len(signedResponseBz)).
				Msg("sent signed relay response")
		} else {
			// Fallback: no response signer or not a relay request - send raw response
			// This path should only be used for non-relay traffic (health checks, etc.)
			if relayRequest != nil && p.responseSigner == nil {
				p.logger.Warn().
					Str(logging.FieldServiceID, serviceID).
					Msg("no response signer configured - sending unsigned response (NOT valid for relay protocol)")
			}

			// Copy response headers
			for key, values := range respHeaders {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}

			// Write response
			w.WriteHeader(respStatus)
			if _, err := w.Write(respBody); err != nil {
				p.logger.Debug().Err(err).Msg("failed to write response body")
			}
		}
	}

	relaysServed.WithLabelValues(serviceID).Inc()
	relayLatency.WithLabelValues(serviceID).Observe(time.Since(startTime).Seconds())

	// Track streaming metrics
	if isStreaming {
		streamingRelaysServed.WithLabelValues(serviceID).Inc()
	}

	// For optimistic validation, validate after serving (in background)
	if validationMode == ValidationModeOptimistic {
		go func() {
			validationStart := time.Now()
			if err := p.validateRelayRequest(context.Background(), r, body, arrivalBlockHeight); err != nil {
				validationFailures.WithLabelValues(serviceID, "signature").Inc()
				p.logger.Debug().
					Err(err).
					Str(logging.FieldServiceID, serviceID).
					Msg("optimistic validation failed")
				return
			}
			validationLatency.WithLabelValues(serviceID, "optimistic").Observe(time.Since(validationStart).Seconds())

			// Submit publish task to worker pool (after successful validation)
			p.submitPublishTask(relayRequest, r, body, respBody, arrivalBlockHeight, serviceID)
		}()
	} else {
		// For eager validation, submit publish task to worker pool
		p.submitPublishTask(relayRequest, r, body, respBody, arrivalBlockHeight, serviceID)
	}
}

// submitPublishTask submits a relay for publication via the worker pool.
// This is non-blocking and uses the server context, not the request context.
func (p *ProxyServer) submitPublishTask(
	relayRequest *servicetypes.RelayRequest,
	r *http.Request,
	reqBody, respBody []byte,
	arrivalBlockHeight int64,
	serviceID string,
) {
	// Get supplier address from relay request if available
	var supplierAddr string
	var sessionID string
	var applicationAddr string

	if relayRequest != nil {
		supplierAddr = relayRequest.Meta.SupplierOperatorAddress
		if relayRequest.Meta.SessionHeader != nil {
			sessionID = relayRequest.Meta.SessionHeader.SessionId
			applicationAddr = relayRequest.Meta.SessionHeader.ApplicationAddress
		}
	}

	// Fallback to headers if not in relay request
	if supplierAddr == "" {
		supplierAddr = p.supplierAddress
	}
	if supplierAddr == "" {
		supplierAddr = r.Header.Get("Pocket-Supplier-Address")
	}
	if supplierAddr == "" {
		p.logger.Warn().
			Str(logging.FieldServiceID, serviceID).
			Msg("no supplier address available, skipping relay publication")
		return
	}
	if sessionID == "" {
		sessionID = r.Header.Get("Pocket-Session-Id")
	}
	if applicationAddr == "" {
		applicationAddr = r.Header.Get("Pocket-Application-Address")
	}

	task := publishTask{
		reqBody:            reqBody,
		respBody:           respBody,
		arrivalBlockHeight: arrivalBlockHeight,
		serviceID:          serviceID,
		supplierAddr:       supplierAddr,
		sessionID:          sessionID,
		applicationAddr:    applicationAddr,
	}

	// Non-blocking send to avoid latency on hot path
	select {
	case p.publishCh <- task:
		// Successfully queued
	default:
		// Channel full - log and drop (better than blocking hot path)
		p.logger.Warn().
			Str(logging.FieldServiceID, serviceID).
			Str(logging.FieldSupplier, supplierAddr).
			Msg("publish channel full, relay dropped")
		relaysDropped.WithLabelValues(serviceID, "channel_full").Inc()
	}
}

// parseRelayRequest parses the relay request protobuf body and extracts the service ID
// and the POKTHTTPRequest payload. Returns nil values if the body is not a valid relay request.
func (p *ProxyServer) parseRelayRequest(body []byte) (*servicetypes.RelayRequest, string, *sdktypes.POKTHTTPRequest, error) {
	if len(body) == 0 {
		return nil, "", nil, fmt.Errorf("empty body")
	}

	// Try to unmarshal as a RelayRequest protobuf
	relayRequest := &servicetypes.RelayRequest{}
	if err := relayRequest.Unmarshal(body); err != nil {
		// Not a valid relay request - this is expected for non-relay traffic
		p.logger.Debug().
			Err(err).
			Msg("request body is not a valid RelayRequest protobuf")
		return nil, "", nil, err
	}

	// Extract service ID from the session header
	var serviceID string
	if relayRequest.Meta.SessionHeader != nil {
		serviceID = relayRequest.Meta.SessionHeader.ServiceId
	}

	if serviceID == "" {
		return relayRequest, "", nil, fmt.Errorf("missing service ID in relay request")
	}

	p.logger.Debug().
		Str(logging.FieldServiceID, serviceID).
		Msg("extracted service ID from relay request")

	// Deserialize the POKTHTTPRequest from the payload
	poktHTTPRequest, err := sdktypes.DeserializeHTTPRequest(relayRequest.Payload)
	if err != nil {
		p.logger.Debug().
			Err(err).
			Str(logging.FieldServiceID, serviceID).
			Msg("failed to deserialize POKTHTTPRequest from payload")
		return relayRequest, serviceID, nil, err
	}

	p.logger.Debug().
		Str(logging.FieldServiceID, serviceID).
		Str("method", poktHTTPRequest.Method).
		Str("url", poktHTTPRequest.Url).
		Msg("deserialized POKTHTTPRequest from relay payload")

	return relayRequest, serviceID, poktHTTPRequest, nil
}

// extractServiceID extracts the service ID from request headers or path.
// This is a fallback method for non-relay traffic or when the body cannot be parsed.
func (p *ProxyServer) extractServiceID(r *http.Request) string {
	// Try header first
	if serviceID := r.Header.Get("Pocket-Service-Id"); serviceID != "" {
		return serviceID
	}

	// Try X-Forwarded-Host header (for path-based routing)
	if host := r.Header.Get("X-Forwarded-Host"); host != "" {
		// Could parse host to extract service ID
		return host
	}

	// Try path-based extraction (e.g., /v1/ethereum/...)
	// This is a simplified version - real implementation would be more robust
	if len(r.URL.Path) > 1 {
		// Extract first path segment
		path := r.URL.Path[1:] // Remove leading /
		for i, c := range path {
			if c == '/' {
				return path[:i]
			}
		}
		return path
	}

	return ""
}

// forwardToBackendWithStreaming forwards the request to the backend service,
// handling both streaming and non-streaming responses.
// Returns the response body, headers, status, whether it was streaming, and any error.
// For streaming responses, the body is written directly to the ResponseWriter.
// If poktHTTPRequest is provided (valid relay request), it uses the deserialized request data.
// Otherwise, it falls back to forwarding the raw body (for non-relay traffic).
func (p *ProxyServer) forwardToBackendWithStreaming(
	ctx context.Context,
	originalReq *http.Request,
	body []byte,
	serviceID string,
	svcConfig *ServiceConfig,
	poktHTTPRequest *sdktypes.POKTHTTPRequest,
	w http.ResponseWriter,
) ([]byte, http.Header, int, bool, error) {
	// Get backend URL based on RPC type (default to "rest")
	rpcType := originalReq.Header.Get("Rpc-Type")
	if rpcType == "" {
		rpcType = "rest"
	}

	// Find the backend configuration
	var backendURL string
	var configHeaders map[string]string
	var auth *AuthenticationConfig

	if backend, ok := svcConfig.Backends[rpcType]; ok {
		backendURL = backend.URL
		configHeaders = backend.Headers
		auth = backend.Authentication
	} else if backend, ok := svcConfig.Backends["rest"]; ok {
		// Fallback to rest backend
		backendURL = backend.URL
		configHeaders = backend.Headers
		auth = backend.Authentication
	} else {
		// Use any available backend
		for _, backend := range svcConfig.Backends {
			backendURL = backend.URL
			configHeaders = backend.Headers
			auth = backend.Authentication
			break
		}
	}

	if backendURL == "" {
		return nil, nil, 0, false, fmt.Errorf("no backend configured for service %s and RPC type %s", serviceID, rpcType)
	}

	// Create backend request
	timeout := p.config.GetServiceTimeout(serviceID)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Parse the backend URL
	parsedBackendURL, err := url.Parse(backendURL)
	if err != nil {
		return nil, nil, 0, false, fmt.Errorf("failed to parse backend URL: %w", err)
	}

	var req *http.Request

	// If we have a valid POKTHTTPRequest from the relay payload, use it to build the backend request
	if poktHTTPRequest != nil {
		// Parse the URL from the POKTHTTPRequest
		requestURL, err := url.Parse(poktHTTPRequest.Url)
		if err != nil {
			return nil, nil, 0, false, fmt.Errorf("failed to parse request URL: %w", err)
		}

		// Replace host and scheme with backend URL
		requestURL.Host = parsedBackendURL.Host
		requestURL.Scheme = parsedBackendURL.Scheme

		// Prepend the backend path to the request path
		if parsedBackendURL.Path != "" && parsedBackendURL.Path != "/" {
			requestURL.Path = strings.TrimSuffix(parsedBackendURL.Path, "/") + requestURL.Path
		}

		// Merge query parameters from backend URL
		query := requestURL.Query()
		for key, values := range parsedBackendURL.Query() {
			for _, value := range values {
				query.Add(key, value)
			}
		}
		requestURL.RawQuery = query.Encode()

		// Create the HTTP request with the payload body
		req, err = http.NewRequestWithContext(ctx, poktHTTPRequest.Method, requestURL.String(), bytes.NewReader(poktHTTPRequest.BodyBz))
		if err != nil {
			return nil, nil, 0, false, fmt.Errorf("failed to create request: %w", err)
		}

		// Copy headers from POKTHTTPRequest
		poktHTTPRequest.CopyToHTTPHeader(req.Header)

		p.logger.Debug().
			Str("method", poktHTTPRequest.Method).
			Str("url", requestURL.String()).
			Int("body_size", len(poktHTTPRequest.BodyBz)).
			Msg("built backend request from POKTHTTPRequest")
	} else {
		// Fallback: forward the raw body for non-relay traffic
		fullBackendURL := backendURL
		if originalReq.URL.Path != "" && originalReq.URL.Path != "/" {
			if parsedBackendURL.Path == "" || parsedBackendURL.Path == "/" {
				parsedBackendURL.Path = originalReq.URL.Path
			} else {
				parsedBackendURL.Path = strings.TrimSuffix(parsedBackendURL.Path, "/") + originalReq.URL.Path
			}
			fullBackendURL = parsedBackendURL.String()
		}

		req, err = http.NewRequestWithContext(ctx, originalReq.Method, fullBackendURL, bytes.NewReader(body))
		if err != nil {
			return nil, nil, 0, false, fmt.Errorf("failed to create request: %w", err)
		}

		// Copy relevant headers from original request
		p.copyHeaders(req, originalReq)
	}

	// Apply service-specific configuration headers (override any matching headers)
	for key, value := range configHeaders {
		req.Header.Set(key, value)
	}

	// Apply authentication
	if auth != nil {
		if auth.Username != "" && auth.Password != "" {
			req.SetBasicAuth(auth.Username, auth.Password)
		} else if auth.BearerToken != "" {
			req.Header.Set("Authorization", "Bearer "+auth.BearerToken)
		}
	}

	// Execute request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, nil, 0, false, fmt.Errorf("backend request failed: %w", err)
	}

	// Check if this is a streaming response
	if isStreamingResponse(resp) {
		// Handle streaming response
		respBody, streamErr := p.handleStreamingResponse(resp, w)
		return respBody, resp.Header, resp.StatusCode, true, streamErr
	}

	// Non-streaming: read entire response
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, 0, false, fmt.Errorf("failed to read response: %w", err)
	}

	return respBody, resp.Header, resp.StatusCode, false, nil
}

// isStreamingResponse checks if the HTTP response should be handled as a stream.
// Detects SSE (text/event-stream) and NDJSON (application/x-ndjson) content types.
func isStreamingResponse(resp *http.Response) bool {
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		return false
	}

	// Parse media type to strip parameters (e.g., "; charset=utf-8")
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return false
	}

	return slices.Contains(httpStreamingTypes, strings.ToLower(mediaType))
}

// handleStreamingResponse handles streaming responses (SSE, NDJSON).
// It forwards chunks in real-time to the client while collecting the full body
// for relay publishing.
func (p *ProxyServer) handleStreamingResponse(
	resp *http.Response,
	w http.ResponseWriter,
) ([]byte, error) {
	defer resp.Body.Close()

	// Copy headers to response
	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	// Set connection close to prevent client reuse issues with streaming
	w.Header().Set("Connection", "close")
	w.WriteHeader(resp.StatusCode)

	// Check if writer supports flushing (optional but recommended for streaming)
	flusher, canFlush := w.(http.Flusher)

	// Buffer to collect full response for relay publishing
	var fullResponse bytes.Buffer

	// Stream chunks to client
	scanner := bufio.NewScanner(resp.Body)

	// Increase buffer size for large chunks (LLM responses can be large)
	const maxScanTokenSize = 256 * 1024 // 256KB per chunk
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	for scanner.Scan() {
		line := scanner.Bytes()
		lineWithNewline := append(line, '\n')

		// Collect for full response
		fullResponse.Write(lineWithNewline)

		// Forward to client
		if _, err := w.Write(lineWithNewline); err != nil {
			return fullResponse.Bytes(), fmt.Errorf("failed to write stream chunk: %w", err)
		}

		// Flush immediately for low latency if supported
		if canFlush {
			flusher.Flush()
		}

		// Track streaming metrics
		streamingChunksForwarded.Inc()
	}

	if err := scanner.Err(); err != nil {
		return fullResponse.Bytes(), fmt.Errorf("stream scanning error: %w", err)
	}

	streamingBytesForwarded.Add(float64(fullResponse.Len()))
	return fullResponse.Bytes(), nil
}

// copyHeaders copies relevant headers from original request to backend request.
func (p *ProxyServer) copyHeaders(dst, src *http.Request) {
	// Headers to copy
	headersToCopy := []string{
		"Content-Type",
		"Accept",
		"Accept-Encoding",
		"User-Agent",
	}

	for _, header := range headersToCopy {
		if value := src.Header.Get(header); value != "" {
			dst.Header.Set(header, value)
		}
	}

	// Copy Pocket-* headers if forward_pocket_headers is enabled
	// (This would be checked per-service in real implementation)
	// HTTP headers in Go are canonicalized, but we use case-insensitive matching
	// to handle any edge cases with header casing from different clients.
	for key := range src.Header {
		if strings.HasPrefix(strings.ToLower(key), "pocket-") {
			dst.Header.Set(key, src.Header.Get(key))
		}
	}
}

// SetValidator sets the relay validator for the proxy server.
// This is optional - if not set, validation is skipped (useful for testing).
func (p *ProxyServer) SetValidator(validator RelayValidator) {
	p.validator = validator
}

// SetRelayProcessor sets the relay processor for proper relay mining.
// This is required for proper relay handling - without it, mined relays will be skipped.
func (p *ProxyServer) SetRelayProcessor(processor RelayProcessor) {
	p.relayProcessor = processor
}

// SetResponseSigner sets the response signer for signing relay responses.
// This is REQUIRED for proper relay handling - clients expect signed RelayResponse protobufs.
func (p *ProxyServer) SetResponseSigner(signer *ResponseSigner) {
	p.responseSigner = signer
}

// SetSupplierCache sets the supplier cache for checking supplier state.
// This allows the relayer to check if suppliers are active before processing relays.
func (p *ProxyServer) SetSupplierCache(cache *cache.SupplierCache) {
	p.supplierCache = cache
}

// SetSupplierAddress sets the supplier address for this proxy instance.
func (p *ProxyServer) SetSupplierAddress(addr string) {
	p.supplierAddress = addr
}

// InitGRPCHandler initializes the gRPC proxy handler for handling gRPC and gRPC-Web requests.
// This should be called after SetRelayProcessor and SetResponseSigner.
func (p *ProxyServer) InitGRPCHandler() {
	// Initialize the new relay service (proper relay protocol over gRPC)
	p.grpcRelayService = NewRelayGRPCService(
		p.logger,
		RelayGRPCServiceConfig{
			ServiceConfigs:     p.config.Services,
			ResponseSigner:     p.responseSigner,
			Publisher:          p.publisher,
			RelayProcessor:     p.relayProcessor,
			CurrentBlockHeight: &p.currentBlockHeight,
			MaxBodySize:        p.config.DefaultMaxBodySizeBytes,
			HTTPClient:         p.httpClient,
		},
	)

	// Create gRPC server for the relay service
	// This properly handles RelayRequest/RelayResponse protocol
	p.grpcRelayServer = NewGRPCServerForRelayService(p.grpcRelayService)
	p.logger.Info().Msg("gRPC relay service server initialized")

	// Legacy: Initialize the transparent proxy handler (deprecated - kept for gRPC-Web compatibility)
	p.grpcHandler = NewGRPCProxyHandler(
		p.logger,
		p.config.Services,
		p.supplierAddress,
		p.relayProcessor,
		p.publisher,
		p.responseSigner,
		&p.currentBlockHeight,
	)

	// Initialize gRPC-Web wrapper using the relay server (not legacy handler)
	// gRPC-Web clients should send proper RelayRequest messages
	p.grpcWebWrapper = NewGRPCWebWrapper(
		p.logger,
		p.grpcRelayServer,
	)

	p.logger.Info().Msg("gRPC relay service and handlers initialized")
}

// validateRelayRequest validates the relay request.
// If no validator is configured, validation is skipped (but body must still be valid RelayRequest).
func (p *ProxyServer) validateRelayRequest(
	ctx context.Context,
	r *http.Request,
	body []byte,
	arrivalBlockHeight int64,
) error {
	// Deserialize RelayRequest from body
	// SECURITY: This should always succeed since we already validated in handleRelay
	relayRequest := &servicetypes.RelayRequest{}
	if err := relayRequest.Unmarshal(body); err != nil {
		// SECURITY FIX: Reject non-relay traffic - don't allow unsigned requests
		return fmt.Errorf("invalid relay request: %w", err)
	}

	// If no validator is configured, skip signature/session validation
	// (The request is still a valid RelayRequest protobuf, just not cryptographically verified)
	if p.validator == nil {
		p.logger.Debug().Msg("no validator configured, skipping signature validation")
		return nil
	}

	// Set the block height for the validator
	p.validator.SetCurrentBlockHeight(arrivalBlockHeight)

	// Validate the relay request
	if err := p.validator.ValidateRelayRequest(ctx, relayRequest); err != nil {
		return fmt.Errorf("relay validation failed: %w", err)
	}

	// Check reward eligibility (for eager validation, we do this now)
	if err := p.validator.CheckRewardEligibility(ctx, relayRequest); err != nil {
		p.logger.Warn().
			Err(err).
			Msg("relay not eligible for rewards (continuing to serve)")
		// Don't return error - we still serve the relay, just won't get rewards
	}

	return nil
}

// executePublish processes a publish task and publishes the relay to Redis.
// This is called by worker goroutines with the server context.
func (p *ProxyServer) executePublish(ctx context.Context, task publishTask) {
	if p.publisher == nil {
		p.logger.Debug().
			Str(logging.FieldServiceID, task.serviceID).
			Msg("no publisher configured, skipping relay publication")
		return
	}

	// Use RelayProcessor if available for proper relay construction
	if p.relayProcessor != nil {
		msg, err := p.relayProcessor.ProcessRelay(
			ctx,
			task.reqBody,
			task.respBody,
			task.supplierAddr,
			task.serviceID,
			task.arrivalBlockHeight,
		)
		if err != nil {
			p.logger.Warn().
				Err(err).
				Str(logging.FieldServiceID, task.serviceID).
				Msg("failed to process relay")
			return
		}

		// msg is nil if relay doesn't meet mining difficulty
		if msg == nil {
			p.logger.Debug().
				Str(logging.FieldServiceID, task.serviceID).
				Msg("relay skipped (not mined)")
			return
		}

		// Publish the mined relay
		if err := p.publisher.Publish(ctx, msg); err != nil {
			p.logger.Warn().
				Err(err).
				Str(logging.FieldServiceID, task.serviceID).
				Msg("failed to publish mined relay")
			return
		}

		relaysPublished.WithLabelValues(task.serviceID, task.supplierAddr).Inc()
		relaysMinedSuccessfully.WithLabelValues(task.serviceID).Inc()
		return
	}

	// Fallback: create a basic message without proper relay construction
	// This path should only be used in testing or when RelayProcessor is not configured
	p.logger.Warn().
		Str(logging.FieldServiceID, task.serviceID).
		Msg("no relay processor configured, using fallback message construction")

	msg := &transport.MinedRelayMessage{
		RelayHash:               nil, // Not calculated - fallback mode
		RelayBytes:              task.reqBody,
		ComputeUnitsPerRelay:    1,
		SessionId:               task.sessionID,
		SessionEndHeight:        0,
		SupplierOperatorAddress: task.supplierAddr,
		ServiceId:               task.serviceID,
		ApplicationAddress:      task.applicationAddr,
		ArrivalBlockHeight:      task.arrivalBlockHeight,
	}
	msg.SetPublishedAt()

	if err := p.publisher.Publish(ctx, msg); err != nil {
		p.logger.Warn().
			Err(err).
			Str(logging.FieldServiceID, task.serviceID).
			Msg("failed to publish mined relay")
		return
	}

	relaysPublished.WithLabelValues(task.serviceID, task.supplierAddr).Inc()
}

// sendError sends an error response.
func (p *ProxyServer) sendError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":"%s"}`, message)
}

// SetBlockHeight updates the current block height.
func (p *ProxyServer) SetBlockHeight(height int64) {
	p.currentBlockHeight.Store(height)
	currentBlockHeight.Set(float64(height))
}

// Close gracefully shuts down the proxy server.
func (p *ProxyServer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	if p.cancelFn != nil {
		p.cancelFn()
	}

	p.wg.Wait()

	p.logger.Info().Msg("proxy server closed")
	return nil
}
