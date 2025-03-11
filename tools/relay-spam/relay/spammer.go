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
}

// RequestResult represents the result of a relay request
type RequestResult struct {
	Success bool
	Error   string
}

// Metrics tracks the results of the relay spam run
type Metrics struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	StartTime          time.Time
	EndTime            time.Time
}

// Spammer is responsible for sending relay requests
type Spammer struct {
	config      *config.Config
	numRequests int
	concurrency int
	rateLimit   float64
}

// NewSpammer creates a new relay spammer
func NewSpammer(cfg *config.Config, numRequests, concurrency int, rateLimit float64) *Spammer {
	return &Spammer{
		config:      cfg,
		numRequests: numRequests,
		concurrency: concurrency,
		rateLimit:   rateLimit,
	}
}

// Run executes the relay spam process
func (s *Spammer) Run(ctx context.Context) (*Metrics, error) {
	metrics := &Metrics{
		TotalRequests:      0,
		SuccessfulRequests: 0,
		FailedRequests:     0,
		StartTime:          time.Now(),
	}

	// Create work items
	var workItems []WorkItem
	for _, app := range s.config.Applications {
		for _, gatewayURL := range app.Gateways {
			for i := 0; i < s.numRequests; i++ {
				workItems = append(workItems, WorkItem{
					AppAddress: app.Address,
					GatewayURL: gatewayURL,
				})
			}
		}
	}

	metrics.TotalRequests = int64(len(workItems))
	fmt.Printf("Created %d work items\n", metrics.TotalRequests)

	// Create worker pool
	var wg sync.WaitGroup
	workCh := make(chan WorkItem)

	// Start workers
	for i := 0; i < s.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for work := range workCh {
				result := s.makeRequest(work.GatewayURL, work.AppAddress)
				if result.Success {
					atomic.AddInt64(&metrics.SuccessfulRequests, 1)
				} else {
					atomic.AddInt64(&metrics.FailedRequests, 1)
					fmt.Printf("Request failed: %s\n", result.Error)
				}
			}
		}()
	}

	// Rate limiting
	startTime := time.Now()
	for i, work := range workItems {
		if s.rateLimit > 0 {
			expectedTime := float64(i) / s.rateLimit
			elapsed := time.Since(startTime).Seconds()
			if elapsed < expectedTime {
				time.Sleep(time.Duration((expectedTime - elapsed) * float64(time.Second)))
			}
		}

		select {
		case workCh <- work:
			// Work sent
		case <-ctx.Done():
			close(workCh)
			return metrics, ctx.Err()
		}
	}

	close(workCh)
	wg.Wait()

	metrics.EndTime = time.Now()
	return metrics, nil
}

// makeRequest sends a single relay request
func (s *Spammer) makeRequest(gatewayURL, appAddress string) RequestResult {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_blockNumber",
		"params":  []interface{}{},
		"id":      1,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return RequestResult{Success: false, Error: err.Error()}
	}

	req, err := http.NewRequest("POST", gatewayURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return RequestResult{Success: false, Error: err.Error()}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-App-Address", appAddress)
	req.Header.Set("Target-Service-ID", s.config.ApplicationDefaults.ServiceID)

	resp, err := client.Do(req)
	if err != nil {
		return RequestResult{Success: false, Error: err.Error()}
	}
	defer resp.Body.Close()

	var errorMsg string
	if resp.StatusCode != 200 {
		errorMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return RequestResult{
		Success: resp.StatusCode == 200,
		Error:   errorMsg,
	}
}
