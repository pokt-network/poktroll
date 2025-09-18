// Buffer Pool for High-Concurrency HTTP Processing
// ================================================
//
// This buffer pool manages reusable byte buffers to optimize memory allocation
// for high-throughput HTTP response processing. When handling thousands of
// concurrent HTTP requests with large response bodies (blockchain data often
// exceeds 1MB), naive allocation patterns create significant performance issues.
//
// Memory Allocation Patterns:
//   - Without pooling: Each request allocates new []byte buffers
//   - With pooling: Buffers are reused across requests via sync.Pool
//
// Benefits:
//   - Reduces garbage collection pressure
//   - Provides predictable memory usage under load
//   - Maintains consistent performance during traffic spikes
//   - Size limits prevent memory bloat
//
// The pool automatically grows buffer capacity as needed while preventing
// oversized buffers from being returned to avoid memory waste.
package concurrency

import (
	"bytes"
	"io"
	"sync"
)

const (
	// DefaultInitialBufferSize is the initial size of the buffer pool.
	// Start with 256KB buffers - can grow as needed
	DefaultInitialBufferSize = 256 * 1024

	// TODO_IMPROVE: Make this configurable via YAML settings
	// DefaultMaxBufferSize is the maximum size of the buffer pool.
	// Set the max buffer size to 4MB to avoid memory bloat.
	DefaultMaxBufferSize = 4 * 1024 * 1024
)

// BufferPool manages reusable byte buffers to reduce GC pressure.
// Uses sync.Pool for efficient buffer recycling with size limits.
type BufferPool struct {
	pool          sync.Pool
	maxReaderSize int64
}

func NewBufferPool(maxReaderSize int64) *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 0, DefaultInitialBufferSize))
			},
		},
		maxReaderSize: maxReaderSize,
	}
}

// getBuffer retrieves a buffer from the pool.
func (bp *BufferPool) getBuffer() *bytes.Buffer {
	buf := bp.pool.Get().(*bytes.Buffer)
	buf.Reset() // Always reset to ensure clean state
	return buf
}

// putBuffer returns a buffer to the pool.
// Buffers larger than maxBufferSize are not returned to avoid memory bloat.
func (bp *BufferPool) putBuffer(buf *bytes.Buffer) {
	// Skip pooling oversized buffers to prevent memory bloat
	if buf.Cap() > DefaultMaxBufferSize {
		return
	}
	bp.pool.Put(buf)
}

// ReadWithBuffer reads from an io.Reader using a pooled buffer.
func (bp *BufferPool) ReadWithBuffer(r io.Reader) ([]byte, error) {
	buf := bp.getBuffer()
	defer bp.putBuffer(buf)

	limitedReader := io.LimitReader(r, bp.maxReaderSize)
	_, err := buf.ReadFrom(limitedReader)
	if err != nil {
		return nil, err
	}

	// Return independent copy to avoid data races
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}
