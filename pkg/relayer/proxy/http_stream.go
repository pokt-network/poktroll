package proxy

import (
	"bufio"
	"context"
	"fmt"
	"mime"
	"net/http"
	"slices"
	"strings"
	"time"

	sdktypes "github.com/pokt-network/shannon-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/x/service/types"
)

// A custom delimiter used to separate chunks in a streaming response.
const streamDelimiter = "||POKT_STREAM||"

// Batch streaming configuration
const (
	// batchTimeThreshold is the maximum time to wait before flushing a batch (100ms)
	batchTimeThreshold = 100 * time.Millisecond
	// batchSizeThreshold is the maximum payload size before flushing a batch (100KB)
	batchSizeThreshold = 100 * 1024 // 100KB in bytes
	// batchChunksThreshold is the maximum number of chunks received efore flushing a batch (100)
	batchChunksThreshold = 100
)

// Target streaming types
var httpStreamingTypes = []string{
	"text/event-stream",
	"application/x-ndjson",
}

// chunkBatch accumulates multiple chunks for batch signing and writing
type chunkBatch struct {
	chunks      [][]byte  // individual chunk bodies
	totalSize   int64     // cumulative size of all chunks
	totalChunks int64     // number of chunks
	startTime   time.Time // when the batch was created
}

// isStreamingResponse determines if an HTTP response should be handled as a stream.
//
// Checks the "Content-Type" HTTP header against supported streaming types:
//   - text/event-stream (Server-Sent Events)
//   - application/x-ndjson (Newline-Delimited JSON)
//
// Returns true if the response should be streamed based on the content type.
//
// DEV_NOTE: While we could handle any chunked stream, we limit support to these
// content types to ensure predictable behavior and proper testing coverage.
func isStreamingResponse(response *http.Response) bool {
	// Extract the content type from the response header
	ct := response.Header.Get("Content-Type")
	if ct == "" {
		return false
	}

	// Parse the media type to strip parameters (e.g., "; charset=utf-8")
	// and compare the canonical type/subtype only. This avoids substring
	// false-positives and handles case-insensitivity per RFC.
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return false
	}

	return slices.Contains(httpStreamingTypes, strings.ToLower(mediaType))
}

// shouldFlushBatch determines if the current batch should be flushed based on
// time and size thresholds:
//   - Time threshold: 100ms has elapsed since batch creation
//   - Size threshold: batch size >= 100KB
//   - Chunk threshold: chunks number >= 100
//   - Force flag: set to true to flush regardless of thresholds
func shouldFlushBatch(batch *chunkBatch, forceFlush bool) bool {
	if forceFlush {
		return forceFlush && batch.totalChunks > 0
	}

	// Check time threshold
	elapsed := time.Since(batch.startTime)
	if elapsed >= batchTimeThreshold {
		return true
	}

	// Check size threshold
	if batch.totalSize >= batchSizeThreshold {
		return true
	}

	// Check chunk number
	if batch.totalChunks >= batchChunksThreshold {
		return true
	}

	return false
}

// flushBatch signs and writes the accumulated batch to the client
func (server *relayMinerHTTPServer) flushBatch(
	ctx context.Context,
	logger polylog.Logger,
	batch *chunkBatch,
	meta *types.RelayRequestMetadata,
	writer http.ResponseWriter,
	flusher http.Flusher,
) (*types.RelayResponse, error) {
	if batch.totalChunks == 0 {
		return nil, nil
	}

	// Combine all chunks into a single payload
	combinedPayload := make([]byte, 0, batch.totalSize)
	for i := int64(0); i < batch.totalChunks; i++ {
		combinedPayload = append(combinedPayload, batch.chunks[i]...)
	}

	// Wrap combined chunks in POKT HTTP response structure
	poktHTTPResponse := &sdktypes.POKTHTTPResponse{
		StatusCode: uint32(http.StatusOK),
		Header:     make(map[string]*sdktypes.Header, 0),
		BodyBz:     combinedPayload,
	}

	// Marshal with deterministic ordering for signature consistency
	marshalOpts := proto.MarshalOptions{Deterministic: true}
	poktHTTPResponseBz, err := marshalOpts.Marshal(poktHTTPResponse)
	if err != nil {
		return nil, fmt.Errorf("❌ failed to marshal POKT HTTP response batch: %w", err)
	}

	// Sign the batch once
	relayResponse, err := server.newRelayResponse(poktHTTPResponseBz, meta.SessionHeader, meta.SupplierOperatorAddress)
	if err != nil {
		return nil, fmt.Errorf("❌ failed to sign relay response batch: %w", err)
	}

	// Serialize signed response
	signedBatch, err := relayResponse.Marshal()
	if err != nil {
		return nil, fmt.Errorf("❌ failed to marshal signed relay response batch: %w", err)
	}

	// Append POKT stream delimiter (allows client-side batch detection)
	signedBatch = append(signedBatch, []byte(streamDelimiter)...)

	// Write signed batch to client
	if _, err = writer.Write(signedBatch); err != nil {
		return nil, fmt.Errorf("❌ failed to write stream batch to client: %w", err)
	}

	// Flush to ensure data reaches client with low latency
	flusher.Flush()

	return relayResponse, nil
}

// handleHttpStream processes streaming HTTP responses from backend services with batching.
//
// Streaming flow with batching:
//  1. Accumulate newline-delimited chunks in a batch buffer
//  2. Flush batch when either:
//     - 100ms has elapsed since batch creation AND at least one chunk exists
//     - Total accumulated payload reaches 100KB
//  3. When flushing:
//     - Combine all buffered chunks into single POKTHTTPResponse
//     - Sign batch once (not per-chunk)
//     - Write signed batch with delimiter to client
//     - Flush to ensure low-latency delivery
//  4. Final batch automatically flushes when stream ends
//
// This batching strategy reduces signing overhead and improves throughput while
// maintaining low-latency streaming (max 100ms delay) for SSE and NDJSON responses.
//
// TODO_IMPROVE: Consider adding configurable buffer size for scanner to handle
// large streaming chunks (default is 64KB).
// Some LLM responses may exceed this.
//
// Returns:
//   - Final relay response (contains last batch's signature)
//   - Total response size across all batches (for metrics)
//   - Error if streaming fails (network errors, signature failures, etc.)
func (server *relayMinerHTTPServer) handleHttpStream(
	ctx context.Context,
	logger polylog.Logger,
	_ *relayer.InstructionTimer,
	relayRequest *types.RelayRequest,
	response *http.Response,
	writer http.ResponseWriter,
) (*types.RelayResponse, float64, error) {
	// Close the response body early to free up connection pool resources.
	defer CloseBody(logger, response.Body)

	// Extract the metadata from the relay request
	meta := relayRequest.Meta

	// Copy all backend headers to client response
	for k, v := range response.Header {
		writer.Header()[k] = v
	}
	// Force connection close to prevent client reuse issues with streaming
	writer.Header().Set("Connection", "close")
	writer.WriteHeader(response.StatusCode)

	// Verify writer supports flushing (required for streaming)
	flusher, ok := writer.(http.Flusher)
	if !ok {
		logger.Error().Msg("❌ Streaming not supported - ResponseWriter does not implement http.Flusher")
		return nil, 0, fmt.Errorf("❌ failed to open stream request: flusher unavailable")
	}

	// Create scanner with default newline delimiter
	scanner := bufio.NewScanner(response.Body)

	// Initialize batch buffer
	batch := &chunkBatch{
		chunks:      make([][]byte, batchChunksThreshold),
		totalSize:   0,
		totalChunks: 0,
		startTime:   time.Now(),
	}

	// Initialize the return values
	var relayResponse *types.RelayResponse
	responseSize := float64(0)

	// Create a ticker for time-based flushing
	ticker := time.NewTicker(batchTimeThreshold)
	defer ticker.Stop()

	// Process chunks from backend stream with time-based and size-based batching
	for {
		select {
		case <-ctx.Done():
			logger.Debug().Msg("📦 Stream Context canceled: flushing.")
			// Context cancelled - flush any remaining batch before exiting
			if batch.totalChunks > 0 {
				resp, err := server.flushBatch(ctx, logger, batch, &meta, writer, flusher)
				if err != nil {
					logger.Error().Err(err).Msg("❌ failed to flush final batch on context cancellation")
				} else if resp != nil {
					relayResponse = resp
					responseSize += float64(resp.Size())
				}
			}
			return nil, responseSize, ctx.Err()

		case <-ticker.C:
			logger.Debug().Msg("📦 Stream Ticker Time up: Streaming content.")
			// Time threshold (100ms) reached - flush if batch has chunks
			if shouldFlushBatch(batch, false) {
				resp, err := server.flushBatch(ctx, logger, batch, &meta, writer, flusher)
				if err != nil {
					return nil, responseSize, fmt.Errorf("❌ failed to flush batch on time threshold: %w", err)
				}
				if resp != nil {
					relayResponse = resp
					responseSize += float64(resp.Size())
				}

				// Reset batch for next round
				batch.totalSize = 0
				batch.totalChunks = 0
				batch.startTime = time.Now()
			}

		default:
			// Try to read next chunk from scanner (non-blocking via default case)
			if !scanner.Scan() {
				// Stream ended - flush final batch if it has chunks
				logger.Debug().Msg("📦 Stream ended: flusshing last.")
				if batch.totalChunks > 0 {
					resp, err := server.flushBatch(ctx, logger, batch, &meta, writer, flusher)
					if err != nil {
						return nil, responseSize, fmt.Errorf("❌ failed to flush final batch: %w", err)
					}
					if resp != nil {
						relayResponse = resp
						responseSize += float64(resp.Size())
					}
				}

				// Check for scanner errors (network issues, buffer overflows, etc.)
				if err := scanner.Err(); err != nil {
					return nil, responseSize, fmt.Errorf("❌ stream scanning error: %w", err)
				}

				// Stream ended successfully
				return relayResponse, responseSize, nil
			}

			logger.Debug().Msg("📦 Stream Received: Adding to buffer.")

			// Restore newline stripped by scanner (needed for protocol compatibility)
			lineBz := scanner.Bytes()
			line := make([]byte, len(lineBz)+1)
			copy(line, lineBz)
			line[len(lineBz)] = '\n'

			// Add chunk to batch
			batch.chunks[batch.totalChunks] = line
			batch.totalSize += int64(len(line))
			batch.totalChunks += 1

			// Check if batch size threshold (100KB) exceeded
			if shouldFlushBatch(batch, false) {
				logger.Debug().Msg("📦 Stream Buffer full: flushing.")
				resp, err := server.flushBatch(ctx, logger, batch, &meta, writer, flusher)
				if err != nil {
					return nil, responseSize, fmt.Errorf("❌ failed to flush batch on size threshold: %w", err)
				}
				if resp != nil {
					relayResponse = resp
					responseSize += float64(resp.Size())
				}

				// Reset timer
				ticker.Reset(batchTimeThreshold)
				// Reset batch for next round
				batch.totalSize = 0
				batch.totalChunks = 0
				batch.startTime = time.Now()

			}
		}
	}
}

// ScanEvents is a bufio.SplitFunc that splits streaming data by the POKT stream delimiter.
// This is used by clients to parse the signed relay response chunks from the stream.
func ScanEvents(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// Look for the POKT_STREAM delimiter
	if i := strings.Index(string(data), streamDelimiter); i >= 0 {
		// Return chunk without the delimiter
		return i + len(streamDelimiter), data[0:i], nil
	}

	// If we're at EOF, return whatever we have
	if atEOF {
		return len(data), data, nil
	}

	// Request more data
	return 0, nil, nil
}
