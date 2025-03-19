package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pokt-network/poktroll/tools/relay-spam/config"
)

// WorkItem represents a single relay request to be processed
type WorkItem struct {
	AppAddress string
	GatewayURL string
	ServiceID  string
}

// RequestResult represents the result of a relay request
type RequestResult struct {
	Success      bool
	Error        string
	AppAddress   string
	GatewayURL   string
	ResponseTime time.Duration
}

// Metrics tracks the results of the relay spam run
type Metrics struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	StartTime          time.Time
	EndTime            time.Time
	// Enhanced metrics
	AppMetrics         map[string]*AppMetrics
	GatewayMetrics     map[string]*GatewayMetrics
	ResponseTimeMin    time.Duration
	ResponseTimeMax    time.Duration
	ResponseTimeAvg    time.Duration
	ResponseTimeTotals time.Duration
}

// AppMetrics tracks metrics for a specific application
type AppMetrics struct {
	Address            string
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	ResponseTimeAvg    time.Duration
	ResponseTimeTotals time.Duration
}

// GatewayMetrics tracks metrics for a specific gateway
type GatewayMetrics struct {
	URL                string
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	ResponseTimeAvg    time.Duration
	ResponseTimeTotals time.Duration
}

// SpamMode defines how the spammer distributes requests
type SpamMode string

const (
	// FixedRequestsMode sends a fixed number of requests per app-gateway pair
	FixedRequestsMode SpamMode = "fixed"
	// TimeBasedMode sends as many requests as possible within a time period
	TimeBasedMode SpamMode = "time"
	// InfiniteMode sends requests continuously until stopped
	InfiniteMode SpamMode = "infinite"
)

// Spammer is responsible for sending relay requests
type Spammer struct {
	config               *config.Config
	numRequests          int
	concurrency          int
	rateLimit            float64
	mode                 SpamMode
	duration             time.Duration
	client               *http.Client
	distributionStrategy string // "even", "weighted", "random"
	requestTimeout       time.Duration
}

// SpammerOption is a functional option for configuring the Spammer
type SpammerOption func(*Spammer)

// WithMode sets the spam mode
func WithMode(mode SpamMode) SpammerOption {
	return func(s *Spammer) {
		s.mode = mode
	}
}

// WithDuration sets the duration for time-based mode
func WithDuration(duration time.Duration) SpammerOption {
	return func(s *Spammer) {
		s.duration = duration
	}
}

// WithDistributionStrategy sets how requests are distributed across applications
func WithDistributionStrategy(strategy string) SpammerOption {
	return func(s *Spammer) {
		s.distributionStrategy = strategy
	}
}

// WithRequestTimeout sets the timeout for individual requests
func WithRequestTimeout(timeout time.Duration) SpammerOption {
	return func(s *Spammer) {
		s.requestTimeout = timeout
		s.client.Timeout = timeout
	}
}

// NewSpammer creates a new relay spammer
func NewSpammer(cfg *config.Config, numRequests, concurrency int, rateLimit float64, options ...SpammerOption) *Spammer {
	s := &Spammer{
		config:               cfg,
		numRequests:          numRequests,
		concurrency:          concurrency,
		rateLimit:            rateLimit,
		mode:                 FixedRequestsMode,
		duration:             10 * time.Minute, // Default duration for time-based mode
		distributionStrategy: "even",
		requestTimeout:       10 * time.Second,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	// Apply options
	for _, option := range options {
		option(s)
	}

	return s
}

// Run executes the relay spam process
func (s *Spammer) Run(ctx context.Context) (*Metrics, error) {
	metrics := &Metrics{
		TotalRequests:      0,
		SuccessfulRequests: 0,
		FailedRequests:     0,
		StartTime:          time.Now(),
		AppMetrics:         make(map[string]*AppMetrics),
		GatewayMetrics:     make(map[string]*GatewayMetrics),
		ResponseTimeMin:    time.Hour, // Start with a large value
	}

	// Initialize app metrics
	for _, app := range s.config.Applications {
		metrics.AppMetrics[app.Address] = &AppMetrics{
			Address: app.Address,
		}
	}

	// Initialize gateway metrics
	for _, url := range s.config.GatewayURLs {
		metrics.GatewayMetrics[url] = &GatewayMetrics{
			URL: url,
		}
	}

	// Create work generator based on mode
	var workGenerator func(chan<- WorkItem)
	var workItems []WorkItem
	var stopGenerator context.CancelFunc

	switch s.mode {
	case FixedRequestsMode:
		// Create fixed work items
		workItems = s.createFixedWorkItems()
		metrics.TotalRequests = int64(len(workItems))

		workGenerator = func(workCh chan<- WorkItem) {
			for _, work := range workItems {
				select {
				case workCh <- work:
					// Work sent
				case <-ctx.Done():
					return
				}
			}
			close(workCh)
		}

	case TimeBasedMode, InfiniteMode:
		// Create a context with timeout for time-based mode
		var genCtx context.Context
		if s.mode == TimeBasedMode {
			genCtx, stopGenerator = context.WithTimeout(ctx, s.duration)
		} else {
			genCtx, stopGenerator = context.WithCancel(ctx)
		}

		// Generate work items continuously
		workGenerator = func(workCh chan<- WorkItem) {
			defer close(workCh)

			// Create a distribution of app-gateway pairs based on strategy
			var workDistribution []WorkItem
			switch s.distributionStrategy {
			case "even":
				workDistribution = s.createEvenDistribution()
			case "weighted":
				workDistribution = s.createWeightedDistribution()
			case "random":
				// For random, we'll just pick randomly each time
			default:
				workDistribution = s.createEvenDistribution()
			}

			// Keep track of how many requests we've sent
			var requestCounter int64

			// Generate work items until context is done
			for {
				// Check if context is done
				select {
				case <-genCtx.Done():
					atomic.StoreInt64(&metrics.TotalRequests, requestCounter)
					return
				default:
					// Continue
				}

				// Get next work item based on distribution strategy
				var work WorkItem
				if s.distributionStrategy == "random" {
					work = s.getRandomWorkItem()
				} else {
					// Cycle through the distribution
					idx := int(requestCounter) % len(workDistribution)
					work = workDistribution[idx]
				}

				// Send work item
				select {
				case workCh <- work:
					atomic.AddInt64(&requestCounter, 1)
				case <-genCtx.Done():
					atomic.StoreInt64(&metrics.TotalRequests, requestCounter)
					return
				}
			}
		}
	}

	// Create result channel and worker pool
	resultCh := make(chan RequestResult, s.concurrency*2)
	workCh := make(chan WorkItem, s.concurrency*2)

	// Start result collector
	var wgResults sync.WaitGroup
	wgResults.Add(1)
	go func() {
		defer wgResults.Done()
		for result := range resultCh {
			// Update global metrics
			if result.Success {
				atomic.AddInt64(&metrics.SuccessfulRequests, 1)
			} else {
				atomic.AddInt64(&metrics.FailedRequests, 1)
			}

			// Update app metrics
			appMetrics, ok := metrics.AppMetrics[result.AppAddress]
			if ok {
				if result.Success {
					atomic.AddInt64(&appMetrics.SuccessfulRequests, 1)
				} else {
					atomic.AddInt64(&appMetrics.FailedRequests, 1)
				}
				atomic.AddInt64(&appMetrics.TotalRequests, 1)
				atomic.AddInt64((*int64)(&appMetrics.ResponseTimeTotals), int64(result.ResponseTime))
			}

			// Update gateway metrics
			gatewayMetrics, ok := metrics.GatewayMetrics[result.GatewayURL]
			if ok {
				if result.Success {
					atomic.AddInt64(&gatewayMetrics.SuccessfulRequests, 1)
				} else {
					atomic.AddInt64(&gatewayMetrics.FailedRequests, 1)
				}
				atomic.AddInt64(&gatewayMetrics.TotalRequests, 1)
				atomic.AddInt64((*int64)(&gatewayMetrics.ResponseTimeTotals), int64(result.ResponseTime))
			}

			// Update response time metrics
			atomic.AddInt64((*int64)(&metrics.ResponseTimeTotals), int64(result.ResponseTime))

			// Update min/max response times (with atomic operations to avoid race conditions)
			for {
				currentMin := metrics.ResponseTimeMin
				if result.ResponseTime < currentMin {
					if atomic.CompareAndSwapInt64((*int64)(&metrics.ResponseTimeMin), int64(currentMin), int64(result.ResponseTime)) {
						break
					}
				} else {
					break
				}
			}

			for {
				currentMax := metrics.ResponseTimeMax
				if result.ResponseTime > currentMax {
					if atomic.CompareAndSwapInt64((*int64)(&metrics.ResponseTimeMax), int64(currentMax), int64(result.ResponseTime)) {
						break
					}
				} else {
					break
				}
			}
		}
	}()

	// Start work generator
	var wgGenerator sync.WaitGroup
	wgGenerator.Add(1)
	go func() {
		defer wgGenerator.Done()
		workGenerator(workCh)
	}()

	// Start workers
	var wgWorkers sync.WaitGroup
	for i := 0; i < s.concurrency; i++ {
		wgWorkers.Add(1)
		go func() {
			defer wgWorkers.Done()
			s.worker(ctx, workCh, resultCh)
		}()
	}

	// Wait for all work to be generated
	wgGenerator.Wait()

	// Wait for all workers to finish
	wgWorkers.Wait()

	// Close result channel and wait for result collector
	close(resultCh)
	wgResults.Wait()

	// Clean up
	if stopGenerator != nil {
		stopGenerator()
	}

	// Calculate final metrics
	metrics.EndTime = time.Now()

	// Calculate average response time
	if metrics.TotalRequests > 0 {
		metrics.ResponseTimeAvg = time.Duration(int64(metrics.ResponseTimeTotals) / metrics.TotalRequests)
	}

	// Calculate per-app and per-gateway averages
	for _, appMetrics := range metrics.AppMetrics {
		if appMetrics.TotalRequests > 0 {
			appMetrics.ResponseTimeAvg = time.Duration(int64(appMetrics.ResponseTimeTotals) / appMetrics.TotalRequests)
		}
	}

	for _, gatewayMetrics := range metrics.GatewayMetrics {
		if gatewayMetrics.TotalRequests > 0 {
			gatewayMetrics.ResponseTimeAvg = time.Duration(int64(gatewayMetrics.ResponseTimeTotals) / gatewayMetrics.TotalRequests)
		}
	}

	return metrics, nil
}

// worker processes work items from the work channel
func (s *Spammer) worker(ctx context.Context, workCh <-chan WorkItem, resultCh chan<- RequestResult) {
	for {
		select {
		case work, ok := <-workCh:
			if !ok {
				return
			}

			// Apply rate limiting if needed
			if s.rateLimit > 0 {
				time.Sleep(time.Duration(1000/s.rateLimit) * time.Millisecond)
			}

			// Process the work item
			startTime := time.Now()
			success, errMsg := s.makeRequest(work.GatewayURL, work.AppAddress, work.ServiceID)
			responseTime := time.Since(startTime)

			// Send result
			resultCh <- RequestResult{
				Success:      success,
				Error:        errMsg,
				AppAddress:   work.AppAddress,
				GatewayURL:   work.GatewayURL,
				ResponseTime: responseTime,
			}

		case <-ctx.Done():
			return
		}
	}
}

// createFixedWorkItems creates work items for fixed mode
func (s *Spammer) createFixedWorkItems() []WorkItem {
	var workItems []WorkItem
	for _, app := range s.config.Applications {
		// For each gateway ID in DelegateesGoal, look up the corresponding URL
		for _, gatewayID := range app.DelegateesGoal {
			// Get the gateway URL from the mapping
			gatewayURL, exists := s.config.GatewayURLs[gatewayID]
			if !exists {
				fmt.Printf("Warning: No URL found for gateway ID %s, skipping\n", gatewayID)
				continue
			}

			for i := 0; i < s.numRequests; i++ {
				workItems = append(workItems, WorkItem{
					AppAddress: app.Address,
					GatewayURL: gatewayURL,
					ServiceID:  app.ServiceIdGoal,
				})
			}
		}
	}
	return workItems
}

// createEvenDistribution creates an even distribution of app-gateway pairs
func (s *Spammer) createEvenDistribution() []WorkItem {
	var distribution []WorkItem
	for _, app := range s.config.Applications {
		for _, gatewayID := range app.DelegateesGoal {
			gatewayURL, exists := s.config.GatewayURLs[gatewayID]
			if !exists {
				continue
			}

			distribution = append(distribution, WorkItem{
				AppAddress: app.Address,
				GatewayURL: gatewayURL,
				ServiceID:  app.ServiceIdGoal,
			})
		}
	}
	return distribution
}

// createWeightedDistribution creates a weighted distribution based on some criteria
func (s *Spammer) createWeightedDistribution() []WorkItem {
	// This is a simple implementation that gives more weight to certain apps
	// In a real implementation, you might want to use more sophisticated weighting
	var distribution []WorkItem

	for _, app := range s.config.Applications {
		for _, gatewayID := range app.DelegateesGoal {
			gatewayURL, exists := s.config.GatewayURLs[gatewayID]
			if !exists {
				continue
			}

			// Add this pair multiple times based on some weighting factor
			// For now, just add each pair once (same as even distribution)
			distribution = append(distribution, WorkItem{
				AppAddress: app.Address,
				GatewayURL: gatewayURL,
				ServiceID:  app.ServiceIdGoal,
			})
		}
	}

	return distribution
}

// getRandomWorkItem returns a random work item
func (s *Spammer) getRandomWorkItem() WorkItem {
	// Simple implementation that picks a random app and gateway
	// In a real implementation, you might want to use a more sophisticated approach

	// Get a random app
	apps := s.config.Applications
	if len(apps) == 0 {
		return WorkItem{}
	}

	app := apps[time.Now().UnixNano()%int64(len(apps))]

	// Get a random gateway for this app
	if len(app.DelegateesGoal) == 0 {
		return WorkItem{}
	}

	gatewayID := app.DelegateesGoal[time.Now().UnixNano()%int64(len(app.DelegateesGoal))]
	gatewayURL, exists := s.config.GatewayURLs[gatewayID]
	if !exists {
		return WorkItem{}
	}

	return WorkItem{
		AppAddress: app.Address,
		GatewayURL: gatewayURL,
		ServiceID:  app.ServiceIdGoal,
	}
}

// makeRequest sends a single relay request
func (s *Spammer) makeRequest(gatewayURL, appAddress, serviceID string) (bool, string) {
	// Create a new request
	req, err := http.NewRequest("GET", gatewayURL, nil)
	if err != nil {
		return false, err.Error()
	}

	// Set headers based on the curl example
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("App-Address", appAddress)
	req.Header.Set("Target-Service-ID", serviceID)

	// Send the request
	resp, err := s.client.Do(req)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != 200 {
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return true, ""
}

// makeJSONRPCRequest sends a JSON-RPC relay request (alternative implementation)
func (s *Spammer) makeJSONRPCRequest(gatewayURL, appAddress, serviceID string) (bool, string) {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_blockNumber",
		"params":  []interface{}{},
		"id":      1,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return false, err.Error()
	}

	req, err := http.NewRequest("POST", gatewayURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return false, err.Error()
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("App-Address", appAddress)
	req.Header.Set("Target-Service-ID", serviceID)

	resp, err := s.client.Do(req)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return true, ""
}
