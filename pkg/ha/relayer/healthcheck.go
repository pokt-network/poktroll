package relayer

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

// HealthStatus represents the health status of a backend.
type HealthStatus int32

const (
	// HealthStatusUnknown means health has not been checked yet.
	HealthStatusUnknown HealthStatus = iota
	// HealthStatusHealthy means the backend is responding correctly.
	HealthStatusHealthy
	// HealthStatusUnhealthy means the backend is not responding correctly.
	HealthStatusUnhealthy
)

func (s HealthStatus) String() string {
	switch s {
	case HealthStatusUnknown:
		return "unknown"
	case HealthStatusHealthy:
		return "healthy"
	case HealthStatusUnhealthy:
		return "unhealthy"
	default:
		return "invalid"
	}
}

// BackendHealth tracks the health of a single backend.
type BackendHealth struct {
	// ServiceID is the service this backend belongs to.
	ServiceID string

	// BackendURL is the URL of the backend.
	BackendURL string

	// Status is the current health status.
	status atomic.Int32

	// LastCheck is when the last health check was performed.
	lastCheck atomic.Int64

	// LastError is the last error encountered (if unhealthy).
	lastError atomic.Value // stores string

	// ConsecutiveFailures tracks failures for threshold calculation.
	consecutiveFailures atomic.Int32

	// ConsecutiveSuccesses tracks successes for threshold calculation.
	consecutiveSuccesses atomic.Int32
}

// GetStatus returns the current health status.
func (h *BackendHealth) GetStatus() HealthStatus {
	return HealthStatus(h.status.Load())
}

// SetStatus sets the health status.
func (h *BackendHealth) SetStatus(status HealthStatus) {
	h.status.Store(int32(status))
}

// GetLastCheck returns when the last health check was performed.
func (h *BackendHealth) GetLastCheck() time.Time {
	return time.Unix(0, h.lastCheck.Load())
}

// GetLastError returns the last error message (empty if healthy).
func (h *BackendHealth) GetLastError() string {
	if err := h.lastError.Load(); err != nil {
		return err.(string)
	}
	return ""
}

// IsHealthy returns true if the backend is healthy.
func (h *BackendHealth) IsHealthy() bool {
	status := h.GetStatus()
	// Unknown is treated as healthy to avoid blocking on startup
	return status == HealthStatusHealthy || status == HealthStatusUnknown
}

// HealthChecker manages health checks for all backends.
type HealthChecker struct {
	logger     polylog.Logger
	httpClient *http.Client

	// backends maps serviceID -> BackendHealth
	backends   map[string]*BackendHealth
	backendsMu sync.RWMutex

	// Configuration per backend
	configs   map[string]*BackendHealthCheckConfig
	configsMu sync.RWMutex

	// Default thresholds
	defaultUnhealthyThreshold int
	defaultHealthyThreshold   int

	// Lifecycle
	mu       sync.Mutex
	closed   bool
	cancelFn context.CancelFunc
	wg       sync.WaitGroup
}

// NewHealthChecker creates a new health checker.
func NewHealthChecker(logger polylog.Logger) *HealthChecker {
	return &HealthChecker{
		logger: logging.ForComponent(logger, logging.ComponentHealthChecker),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		backends:                  make(map[string]*BackendHealth),
		configs:                   make(map[string]*BackendHealthCheckConfig),
		defaultUnhealthyThreshold: 3,
		defaultHealthyThreshold:   2,
	}
}

// RegisterBackend registers a backend for health checking.
func (hc *HealthChecker) RegisterBackend(serviceID, backendURL string, config *BackendHealthCheckConfig) {
	hc.backendsMu.Lock()
	hc.backends[serviceID] = &BackendHealth{
		ServiceID:  serviceID,
		BackendURL: backendURL,
	}
	hc.backendsMu.Unlock()

	if config != nil {
		hc.configsMu.Lock()
		hc.configs[serviceID] = config
		hc.configsMu.Unlock()
	}

	hc.logger.Info().
		Str(logging.FieldServiceID, serviceID).
		Str("backend_url", backendURL).
		Bool("health_check_enabled", config != nil && config.Enabled).
		Msg("registered backend")
}

// GetHealth returns the health status for a service.
func (hc *HealthChecker) GetHealth(serviceID string) *BackendHealth {
	hc.backendsMu.RLock()
	defer hc.backendsMu.RUnlock()
	return hc.backends[serviceID]
}

// IsHealthy returns true if the backend for the given service is healthy.
func (hc *HealthChecker) IsHealthy(serviceID string) bool {
	health := hc.GetHealth(serviceID)
	if health == nil {
		// Unknown backend - assume healthy
		return true
	}
	return health.IsHealthy()
}

// GetAllHealth returns health status for all backends.
func (hc *HealthChecker) GetAllHealth() map[string]*BackendHealth {
	hc.backendsMu.RLock()
	defer hc.backendsMu.RUnlock()

	result := make(map[string]*BackendHealth, len(hc.backends))
	for k, v := range hc.backends {
		result[k] = v
	}
	return result
}

// Start begins health checking for all registered backends.
func (hc *HealthChecker) Start(ctx context.Context) error {
	hc.mu.Lock()
	if hc.closed {
		hc.mu.Unlock()
		return fmt.Errorf("health checker is closed")
	}

	ctx, hc.cancelFn = context.WithCancel(ctx)
	hc.mu.Unlock()

	// Start health check loops for each configured backend
	hc.configsMu.RLock()
	for serviceID, config := range hc.configs {
		if config.Enabled {
			hc.wg.Add(1)
			go hc.healthCheckLoop(ctx, serviceID, config)
		}
	}
	hc.configsMu.RUnlock()

	hc.logger.Info().Msg("health checker started")
	return nil
}

// healthCheckLoop runs periodic health checks for a single backend.
func (hc *HealthChecker) healthCheckLoop(ctx context.Context, serviceID string, config *BackendHealthCheckConfig) {
	defer hc.wg.Done()

	interval := time.Duration(config.IntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run initial check immediately
	hc.checkBackend(ctx, serviceID, config)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hc.checkBackend(ctx, serviceID, config)
		}
	}
}

// checkBackend performs a single health check for a backend.
func (hc *HealthChecker) checkBackend(ctx context.Context, serviceID string, config *BackendHealthCheckConfig) {
	hc.backendsMu.RLock()
	backend, ok := hc.backends[serviceID]
	hc.backendsMu.RUnlock()

	if !ok {
		return
	}

	// Build health check URL with proper path joining
	healthURL, err := joinURLPath(backend.BackendURL, config.Endpoint)
	if err != nil {
		hc.recordFailure(backend, config, fmt.Sprintf("invalid backend URL: %v", err))
		return
	}

	// Create request with timeout
	timeout := time.Duration(config.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, healthURL, nil)
	if err != nil {
		hc.recordFailure(backend, config, fmt.Sprintf("failed to create request: %v", err))
		return
	}

	// Perform health check
	resp, err := hc.httpClient.Do(req)
	if err != nil {
		hc.recordFailure(backend, config, fmt.Sprintf("request failed: %v", err))
		return
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		hc.recordFailure(backend, config, fmt.Sprintf("unhealthy status code: %d", resp.StatusCode))
		return
	}

	hc.recordSuccess(backend, config)
}

// recordFailure records a health check failure.
func (hc *HealthChecker) recordFailure(backend *BackendHealth, config *BackendHealthCheckConfig, errMsg string) {
	backend.lastCheck.Store(time.Now().UnixNano())
	backend.lastError.Store(errMsg)
	backend.consecutiveSuccesses.Store(0)

	failures := backend.consecutiveFailures.Add(1)

	unhealthyThreshold := hc.defaultUnhealthyThreshold
	if config.UnhealthyThreshold > 0 {
		unhealthyThreshold = config.UnhealthyThreshold
	}

	if int(failures) >= unhealthyThreshold {
		oldStatus := backend.GetStatus()
		backend.SetStatus(HealthStatusUnhealthy)

		if oldStatus != HealthStatusUnhealthy {
			hc.logger.Warn().
				Str(logging.FieldServiceID, backend.ServiceID).
				Str("backend_url", backend.BackendURL).
				Str("error", errMsg).
				Int32("consecutive_failures", failures).
				Msg("backend became unhealthy")

			backendHealthStatus.WithLabelValues(backend.ServiceID).Set(0)
		}
	}

	healthCheckFailures.WithLabelValues(backend.ServiceID).Inc()
}

// recordSuccess records a health check success.
func (hc *HealthChecker) recordSuccess(backend *BackendHealth, config *BackendHealthCheckConfig) {
	backend.lastCheck.Store(time.Now().UnixNano())
	backend.lastError.Store("")
	backend.consecutiveFailures.Store(0)

	successes := backend.consecutiveSuccesses.Add(1)

	healthyThreshold := hc.defaultHealthyThreshold
	if config.HealthyThreshold > 0 {
		healthyThreshold = config.HealthyThreshold
	}

	if int(successes) >= healthyThreshold {
		oldStatus := backend.GetStatus()
		backend.SetStatus(HealthStatusHealthy)

		if oldStatus != HealthStatusHealthy {
			hc.logger.Info().
				Str(logging.FieldServiceID, backend.ServiceID).
				Str("backend_url", backend.BackendURL).
				Int32("consecutive_successes", successes).
				Msg("backend became healthy")

			backendHealthStatus.WithLabelValues(backend.ServiceID).Set(1)
		}
	}

	healthCheckSuccesses.WithLabelValues(backend.ServiceID).Inc()
}

// CheckNow performs an immediate health check for a service.
func (hc *HealthChecker) CheckNow(ctx context.Context, serviceID string) error {
	hc.configsMu.RLock()
	config, ok := hc.configs[serviceID]
	hc.configsMu.RUnlock()

	if !ok {
		return fmt.Errorf("no health check config for service %s", serviceID)
	}

	hc.checkBackend(ctx, serviceID, config)
	return nil
}

// Close gracefully shuts down the health checker.
func (hc *HealthChecker) Close() error {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if hc.closed {
		return nil
	}

	hc.closed = true

	if hc.cancelFn != nil {
		hc.cancelFn()
	}

	hc.wg.Wait()

	hc.logger.Info().Msg("health checker closed")
	return nil
}

// joinURLPath properly joins a base URL with a path, handling edge cases like
// trailing slashes and leading slashes to avoid double slashes.
func joinURLPath(baseURL, pathToJoin string) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	// Clean up paths to avoid double slashes
	basePath := strings.TrimSuffix(parsed.Path, "/")
	joinPath := pathToJoin
	if !strings.HasPrefix(joinPath, "/") {
		joinPath = "/" + joinPath
	}

	parsed.Path = basePath + joinPath
	return parsed.String(), nil
}
