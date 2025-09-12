package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httptrace"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/pokt-network/poktroll/pkg/network/concurrency"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

// TODO_IMPROVE: Make these configurable.
const (
	// Maximum length of an HTTP response's body.
	maxResponseSize = 100 * 1024 * 1024 // 100MB limit

	// Maximum number of concurrent HTTP requests.
	concurrencyLimiterMax = 10_000
)

// HTTPClientWithDebugMetrics provides HTTP client functionality with embedded tracking of debug metrics.
// It includes things like:
// - Built-in request debugging
// - Metrics collection
// - Detailed logging
// - Timeout debugging
// - Connection issue visibility
type HTTPClientWithDebugMetrics struct {
	httpClient *http.Client
	limiter    *concurrency.ConcurrencyLimiter
	bufferPool *concurrency.BufferPool

	// Atomic counters for monitoring
	activeRequests   atomic.Uint64
	totalRequests    atomic.Uint64
	timeoutErrors    atomic.Uint64
	connectionErrors atomic.Uint64
}

// httpRequestMetrics holds detailed timing and status information for a single HTTP request
type httpRequestMetrics struct {
	startTime      time.Time
	url            string
	contextTimeout time.Duration
	goroutineCount int

	// DNS Resolution
	dnsLookupTime time.Duration

	// Connection Establishment
	connectTime      time.Duration
	connectionReused bool
	remoteAddr       string
	localAddr        string

	// TLS Handshake
	tlsTime time.Duration

	// Connection Acquisition (from pool or new)
	getConnTime time.Duration

	// Request Writing
	wroteHeadersTime time.Duration
	wroteRequestTime time.Duration

	// Response Waiting
	firstByteTime time.Duration

	// Overall
	totalTime  time.Duration
	statusCode int
	error      error
}

// NewDefaultHTTPClientWithDebugMetrics creates a new HTTP client with:
// - Transport settings configured for high-concurrency usage
// - Built in request debugging capabilities and metrics tracking
// TODO_TECHDEBT(@adshmh): Make HTTP client settings configurable
func NewDefaultHTTPClientWithDebugMetrics() *HTTPClientWithDebugMetrics {
	// Configure transport with optimized settings for high-concurrency usage
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,  // Quick connection establishment to fail fast on unreachable hosts
			KeepAlive: 30 * time.Second, // Keep-alive probe interval to maintain connection health
		}).DialContext,

		// Connection pool settings scaled with concurrency limit
		MaxIdleConns:        concurrencyLimiterMax / 5,  // Scale total pool: 20% of max concurrency
		MaxIdleConnsPerHost: concurrencyLimiterMax / 20, // Scale per-host pool: 5% of max concurrency
		MaxConnsPerHost:     concurrencyLimiterMax / 10, // Scale max connections: 10% of max concurrency
		IdleConnTimeout:     90 * time.Second,           // Reduced from 300s - shorter idle to free resources

		// Timeout settings optimized for quick failure detection
		TLSHandshakeTimeout:   5 * time.Second, // Fast TLS timeout since handshakes typically complete in ~100ms
		ResponseHeaderTimeout: 5 * time.Second, // Header timeout to allow for server processing time

		// Performance optimizations
		DisableKeepAlives:  false, // Enable connection reuse to reduce connection overhead
		DisableCompression: false, // Enable gzip compression to reduce bandwidth
		ForceAttemptHTTP2:  true,  // Prefer HTTP/2 for connection multiplexing benefits

		// Buffer sizes optimized for throughput
		WriteBufferSize: 32 * 1024, // 32KB write buffer to reduce syscalls for large requests
		ReadBufferSize:  32 * 1024, // 32KB read buffer to reduce syscalls for large responses
	}

	// Create HTTP client with large timeout as fallback
	// Individual requests will use context deadlines for actual timeout control
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   80 * time.Second, // Large fallback timeout (80 seconds)
	}

	return &HTTPClientWithDebugMetrics{
		httpClient: httpClient,
		limiter:    concurrency.NewConcurrencyLimiter(concurrencyLimiterMax),
		bufferPool: concurrency.NewBufferPool(maxResponseSize),
	}
}

// TODO_TECHDEBT(@adshmh): Switch to buffered reading of the HTTP response.
//
// Do executes an HTTP request using the provided context and logger.
// It acquires a concurrency slot before proceeding and sets up a debugging context
// to track detailed metrics of the request lifecycle. In case of an error, it logs
// the metrics for debugging purposes.
//
// Returns:
// - *http.Response: the HTTP response received from the server.
// - error: an error if the request fails, or nil if successful.
func (h *HTTPClientWithDebugMetrics) Do(
	ctx context.Context,
	logger polylog.Logger,
	req *http.Request,
) (*http.Response, error) {
	// Acquire concurrency slot before proceeding
	// TODO: use a predefined error, so we can build a proper observation indicating request failure due to reaching max concurrency.
	if !h.limiter.Acquire(ctx) {
		return &http.Response{}, fmt.Errorf("failed to acquire concurrency slot: context canceled")
	}
	defer h.limiter.Release()

	// Set up debugging context and logging function
	// Wraps around the existing context (to preserve deadline, etc.)
	debugCtx, requestRecorder := h.setupRequestDebugging(ctx, logger, req.URL.String())

	// update the request with the debug context
	reqWithDebugCtx := req.WithContext(debugCtx)

	// Execute HTTP request
	httpResp, err := h.httpClient.Do(reqWithDebugCtx)

	defer func() {
		requestRecorder(err)
	}()

	return httpResp, err
}

// setupRequestDebugging initializes request metrics, HTTP debugging context, and atomic counters.
// Returns the debug context and a cleanup function that accepts an error parameter.
func (h *HTTPClientWithDebugMetrics) setupRequestDebugging(
	ctx context.Context,
	logger polylog.Logger,
	endpointURL string,
) (context.Context, func(error)) {
	// Update atomic counters
	h.activeRequests.Add(1)
	h.totalRequests.Add(1)

	startTime := time.Now()

	// Initialize metrics collection
	metrics := &httpRequestMetrics{
		startTime:      startTime,
		goroutineCount: runtime.NumGoroutine(),
		url:            endpointURL,
	}

	// Capture context timeout for logging
	if deadline, ok := ctx.Deadline(); ok {
		metrics.contextTimeout = time.Until(deadline)
	}

	// Create HTTP trace and add to context
	trace := createDetailedHTTPTrace(metrics)
	debugCtx := httptrace.WithClientTrace(ctx, trace)

	// Return recorder function that logs request details.
	requestRecorder := func(err error) {
		h.activeRequests.Add(^uint64(0)) // Atomic decrement
		metrics.totalTime = time.Since(metrics.startTime)
		metrics.error = err
		// Log detailed metrics on error for debugging
		if err != nil {
			h.logRequestMetrics(logger, *metrics)
		}
	}

	return debugCtx, requestRecorder
}

// categorizeError categorizes HTTP client errors and updates counters for monitoring
func (h *HTTPClientWithDebugMetrics) categorizeError(ctx context.Context, err error) error {
	if ctx.Err() == context.DeadlineExceeded {
		h.timeoutErrors.Add(1)
		return fmt.Errorf("request timeout: %w", err)
	} else {
		h.connectionErrors.Add(1)
		return fmt.Errorf("connection error: %w", err)
	}
}

// readAndValidateResponse reads the response body and validates the HTTP status code
func (h *HTTPClientWithDebugMetrics) readAndValidateResponse(resp *http.Response) ([]byte, error) {
	// Read response body with size protection using buffer pool
	responseBody, err := h.bufferPool.ReadWithBuffer(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// TODO_TECHDEBT(@adshmh): Do we need to verify the HTTP status code is 200?
	return responseBody, nil
}

// createDetailedHTTPTrace creates comprehensive HTTP tracing using the httptrace library:
// https://pkg.go.dev/net/http/httptrace
// Captures granular timing for every phase of the HTTP request lifecycle to identify bottlenecks.
func createDetailedHTTPTrace(metrics *httpRequestMetrics) *httptrace.ClientTrace {
	var (
		dnsStart, connectStart, tlsStart time.Time
		getConnStart, gotConnTime        time.Time
		wroteRequestStart                time.Time
		waitingForResponseStart          time.Time
	)

	return &httptrace.ClientTrace{
		// DNS Resolution Phase
		DNSStart: func(info httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			if !dnsStart.IsZero() {
				metrics.dnsLookupTime = time.Since(dnsStart)
			}
		},

		// Connection Establishment Phase
		ConnectStart: func(network, addr string) {
			connectStart = time.Now()
			metrics.remoteAddr = addr
		},
		ConnectDone: func(network, addr string, err error) {
			if !connectStart.IsZero() {
				metrics.connectTime = time.Since(connectStart)
			}
		},

		// TLS Handshake Phase
		TLSHandshakeStart: func() {
			tlsStart = time.Now()
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			if !tlsStart.IsZero() {
				metrics.tlsTime = time.Since(tlsStart)
			}
		},

		// Connection Acquisition Phase
		// Tracks potential connection pool exhaustions.
		GetConn: func(hostPort string) {
			getConnStart = time.Now()
		},
		GotConn: func(info httptrace.GotConnInfo) {
			gotConnTime = time.Now() // Record when we actually got the connection
			if !getConnStart.IsZero() {
				metrics.getConnTime = time.Since(getConnStart) // This is pure connection acquisition time
			}
			metrics.connectionReused = info.Reused
			if info.Conn != nil {
				metrics.localAddr = info.Conn.LocalAddr().String()
			}
		},

		// Request Writing Phase
		// Tracks potential write delays.
		WroteHeaders: func() {
			// Time from getting connection to headers completion (not from connection start)
			if !gotConnTime.IsZero() {
				metrics.wroteHeadersTime = time.Since(gotConnTime)
			}
			// Start timing request body writing
			wroteRequestStart = time.Now()
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			// Entire request (headers + body) written successfully
			if !wroteRequestStart.IsZero() {
				metrics.wroteRequestTime = time.Since(wroteRequestStart)
			}
			// Now waiting for server response
			waitingForResponseStart = time.Now()
		},

		// Response Reading Phase
		GotFirstResponseByte: func() {
			if !waitingForResponseStart.IsZero() {
				metrics.firstByteTime = time.Since(waitingForResponseStart)
			}
		},
	}
}

// logRequestMetrics logs comprehensive request metrics for debugging failed requests.
// Only called when a request fails to avoid verbose logging on successful requests.
func (h *HTTPClientWithDebugMetrics) logRequestMetrics(logger polylog.Logger, metrics httpRequestMetrics) {
	// Calculate derived timings for easier analysis
	connectionEstablishmentTime := metrics.dnsLookupTime + metrics.connectTime + metrics.tlsTime
	requestTransmissionTime := metrics.wroteHeadersTime + metrics.wroteRequestTime

	// Log detailed failure metrics using the provided structured logger
	logger.With(
		// Request identification
		"http_client_debug_url", metrics.url,
		"http_client_debug_total_ms", metrics.totalTime.Milliseconds(),
		"http_client_debug_timeout_ms", metrics.contextTimeout.Milliseconds(),
		"http_client_debug_status_code", metrics.statusCode,

		// Phase 1: DNS Resolution
		"http_client_debug_dns_lookup_ms", metrics.dnsLookupTime.Milliseconds(),

		// Phase 2: Connection Management
		"http_client_debug_get_conn_ms", metrics.getConnTime.Milliseconds(), // Time to get connection from pool
		"http_client_debug_connection_reused", metrics.connectionReused, // Was connection reused?
		"http_client_debug_connect_ms", metrics.connectTime.Milliseconds(), // TCP connection time (if new)
		"http_client_debug_tls_ms", metrics.tlsTime.Milliseconds(), // TLS handshake time (if new)
		"http_client_debug_connection_establishment_ms", connectionEstablishmentTime.Milliseconds(), // Total setup time

		// Phase 3: Request Transmission
		"http_client_debug_wrote_headers_ms", metrics.wroteHeadersTime.Milliseconds(), // Time to write headers
		"http_client_debug_wrote_request_ms", metrics.wroteRequestTime.Milliseconds(), // Time to write body
		"http_client_debug_request_transmission_ms", requestTransmissionTime.Milliseconds(), // Total write time

		// Phase 4: Response Waiting
		"http_client_debug_first_byte_ms", metrics.firstByteTime.Milliseconds(), // Time waiting for server response

		// Connection details
		"http_client_debug_remote_addr", metrics.remoteAddr,
		"http_client_debug_local_addr", metrics.localAddr,

		// System state
		"http_client_debug_goroutines", metrics.goroutineCount,
		"http_client_debug_active_requests", h.activeRequests.Load(),
		"http_client_debug_total_requests", h.totalRequests.Load(),
		"http_client_debug_timeout_errors", h.timeoutErrors.Load(),
		"http_client_debug_connection_errors", h.connectionErrors.Load(),
	).Error().Err(metrics.error).Msg("HTTP request failed - detailed phase breakdown for timeout debugging")
}
