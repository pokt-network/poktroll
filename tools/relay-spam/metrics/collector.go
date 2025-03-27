package metrics

import (
	"fmt"
	"time"
)

// Metrics tracks the results of the relay spam run
type Metrics struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	StartTime          time.Time
	EndTime            time.Time
}

// Print outputs the metrics in a human-readable format
func (m *Metrics) Print() {
	duration := m.EndTime.Sub(m.StartTime)
	successRate := float64(m.SuccessfulRequests) / float64(m.TotalRequests) * 100
	requestsPerSecond := float64(m.TotalRequests) / duration.Seconds()

	fmt.Println("=== Relay Spam Results ===")
	fmt.Printf("Total Requests:      %d\n", m.TotalRequests)
	fmt.Printf("Successful Requests: %d (%.2f%%)\n", m.SuccessfulRequests, successRate)
	fmt.Printf("Failed Requests:     %d (%.2f%%)\n", m.FailedRequests, 100-successRate)
	fmt.Printf("Duration:            %.2f seconds\n", duration.Seconds())
	fmt.Printf("Requests Per Second: %.2f\n", requestsPerSecond)
}
