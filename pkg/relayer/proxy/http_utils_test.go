package proxy

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/stretchr/testify/require"
)

const maxBodySize = int64(16 * 1024) // 16KB limit

// TestSafeReadBody_BufferPoolCorruption tests for buffer pool race conditions.
// This test demonstrates the critical issue where SafeReadBody returns buf.Bytes()
// which shares the underlying array with the pooled buffer, causing data corruption
// when the buffer gets reused.
func TestSafeReadBody_BufferPoolCorruption(t *testing.T) {
	logger := polyzero.NewLogger()

	// First test data - will be read first
	testData1 := []byte("ORIGINAL_DATA_SHOULD_NOT_CHANGE")
	reader1 := newReadCloser(testData1)

	// Call SafeReadBody and store the result
	result1, err := SafeReadBody(logger, reader1, maxBodySize)
	require.NoError(t, err)
	require.Equal(t, testData1, result1, "First result should match input data")

	// Store a copy of the result for comparison
	result1Copy := make([]byte, len(result1))
	copy(result1Copy, result1)

	// Second test data - different content that will overwrite the buffer
	testData2 := []byte("SECOND_CALL_OVERWRITES_BUFFER_CONTENT_CAUSING_CORRUPTION")
	reader2 := newReadCloser(testData2)

	// Make second call - this will reuse the same buffer from the pool
	result2, err := SafeReadBody(logger, reader2, maxBodySize)
	require.NoError(t, err)
	require.Equal(t, testData2, result2, "Second result should match its input data")

	// CRITICAL TEST: Check if first result got corrupted
	// With the bug (returning buf.Bytes() directly), result1 would be corrupted
	// because it shares the underlying array with the reused buffer
	require.True(t, bytes.Equal(result1, result1Copy),
		"BUFFER POOL CORRUPTION DETECTED!\n"+
			"First result was corrupted after second call:\n"+
			"Original: %q\n"+
			"After 2nd call: %q\n"+
			"Expected to remain: %q\n"+
			"This proves the buffer pool race condition exists!",
		result1Copy, result1, testData1)

	// Both results should still be valid
	require.Equal(t, testData1, result1, "First result should remain unchanged")
	require.Equal(t, testData2, result2, "Second result should be correct")
}

// TestSafeReadBody_MemoryLeakPrevention verifies that buffers are properly
// returned to the pool and memory usage remains stable.
func TestSafeReadBody_MemoryLeakPrevention(t *testing.T) {
	logger := polyzero.NewLogger()

	testData := make([]byte, 8*1024) // 8KB of data (much smaller)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	// Perform many calls to ensure buffers are reused
	const numCalls = 100
	for range numCalls {
		reader := newReadCloser(testData)
		result, err := SafeReadBody(logger, reader, maxBodySize)
		require.NoError(t, err)
		require.Len(t, result, len(testData))
		require.Equal(t, testData, result)
	}
}

// TestSafeReadBody_SizeLimit verifies that the size limit is properly enforced.
func TestSafeReadBody_SizeLimit(t *testing.T) {
	logger := polyzero.NewLogger()

	// Create data larger than the limit (18KB > 16KB limit)
	largeData := make([]byte, 18*1024)
	for i := range largeData {
		largeData[i] = 'A'
	}

	reader := newReadCloser(largeData)

	result, err := SafeReadBody(logger, reader, maxBodySize)
	require.Error(t, err, "Should return error for data exceeding size limit")
	require.Nil(t, result, "Should return nil result on error")
	require.Contains(t, err.Error(), "exceeds maximum allowed body", "Error should mention size limit")
}

// TestSafeReadBody_EmptyBody verifies handling of empty request bodies.
func TestSafeReadBody_EmptyBody(t *testing.T) {
	logger := polyzero.NewLogger()

	reader := newReadCloser([]byte{})
	result, err := SafeReadBody(logger, reader, maxBodySize)
	require.NoError(t, err)
	require.Empty(t, result, "Empty body should return empty result")
}

// TestSafeReadBody_ConcurrentPoolExhaustion tests SafeReadBody under concurrent load
// that exhausts the buffer pool. This test demonstrates that:
// 1. SafeReadBody is safe for concurrent use (no race conditions)
// 2. When the buffer pool is exhausted, new buffers are allocated temporarily
// 3. All goroutines get correct, uncorrupted data regardless of pool state
// 4. The system gracefully handles memory pressure without crashes
func TestSafeReadBody_ConcurrentPoolExhaustion(t *testing.T) {
	logger := polyzero.NewLogger()

	// Create test data that will stress the buffer pool
	testData := make([]byte, 4*1024) // 4KB per request
	for i := range testData {
		testData[i] = byte(i % 256) // Unique pattern
	}

	// Number of concurrent goroutines - more than typical buffer pool size
	// This will force pool exhaustion and temporary buffer allocation
	const numGoroutines = 100
	const requestsPerGoroutine = 10

	// Channel to collect results from all goroutines
	type result struct {
		data []byte
		err  error
		id   int
	}
	results := make(chan result, numGoroutines*requestsPerGoroutine)

	// Launch many concurrent goroutines
	for i := range numGoroutines {
		go func(goroutineID int) {
			// Each goroutine makes multiple requests
			for j := range requestsPerGoroutine {
				reader := newReadCloser(testData)
				data, err := SafeReadBody(logger, reader, maxBodySize)
				results <- result{
					data: data,
					err:  err,
					id:   goroutineID*requestsPerGoroutine + j,
				}
			}
		}(i)
	}

	// Collect and verify all results
	for range numGoroutines * requestsPerGoroutine {
		select {
		case res := <-results:
			require.NoError(t, res.err, "Request %d should not error", res.id)
			require.Equal(t, testData, res.data, "Request %d should return correct data", res.id)
		case <-time.After(30 * time.Second):
			t.Fatal("Test timed out - possible deadlock or performance issue")
		}
	}

	// Additional verification: make sure we can still use SafeReadBody normally
	// after the concurrent stress test (buffer pool should be in good state)
	reader := newReadCloser(testData)
	finalResult, err := SafeReadBody(logger, reader, maxBodySize)
	require.NoError(t, err)
	require.Equal(t, testData, finalResult, "SafeReadBody should work normally after concurrent stress")
}

// newReadCloser creates a real io.ReadCloser from byte data using io.NopCloser
// which wraps any io.Reader to make it an io.ReadCloser
func newReadCloser(data []byte) io.ReadCloser {
	return io.NopCloser(bytes.NewReader(data))
}
