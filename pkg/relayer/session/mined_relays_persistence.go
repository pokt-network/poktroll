// Context:
// - The Session Merkle Trie (SMT/SMST) used by the relayer is backed by an in-memory map (SimpleMap).
// - This is fast, but volatile: a crash or restart will drop any unclaimed, in-flight relays from RAM.
//
// This adds a backup path to avoid losing mined relays during crashes or restarts.:
// - Buffer serialized relays in memory (for performance)
// - Periodically and conditionally flush them to an append-only log on disk (a WAL)
// - On restart, deterministically replay the WAL to reconstruct the in-memory SMST
package session

import (
	"encoding/binary"
	"io"
	"os"
	"sync"
	"time"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore"
)

const (
	// maxBufferedMinedRelaysBytesBeforeFlush defines when to proactively flush the
	// in-memory buffer to disk. It is a simple safety and responsiveness threshold:
	// - Keeps memory usage bounded
	// - Ensures timely persistence even if traffic is bursty
	//
	// TODO_TECHDEBT: Make this value configurable via RelayMinerConfig.
	// High-throughput suppliers may need lower thresholds to prevent memory pressure.
	// Low-resource environments may need lower thresholds to prevent OOM.
	maxBufferedMinedRelaysBytesBeforeFlush = 10_000_000 // 10 MB

	// minedRelaysLogFlushInterval is the periodic cadence for background buffer flushes.
	// Even if the threshold is not hit, this ensures regular persistence:
	// - Reduces potential loss window in case of abrupt termination
	// - Smooths out I/O instead of writing every mined relay
	//
	// TODO_TECHDEBT: Make this value configurable via RelayMinerConfig.
	// Testing environments may want faster flushes (e.g., 1s) for rapid iteration.
	// Production may want longer intervals (e.g., 30s) to reduce I/O overhead.
	minedRelaysLogFlushInterval = 10 * time.Second

	// Encoding constants for the on-disk WAL format

	// minedRelaysLogRelayPayloadLengthPrefixSizeBytes indicates the size of the
	// little-endian uint32 prefix that encodes the mined relay payload length.
	// This allows us to read frames efficiently during replay.
	minedRelaysLogRelayPayloadLengthPrefixSizeBytes = 4

	// minedRelaysLogRelayHashSizeBytes is the fixed number of bytes for a relay hash entry.
	// It matches the protocol's RelayHasherSize so each frame has a predictable layout.
	minedRelaysLogRelayHashSizeBytes = protocol.RelayHasherSize

	// minedRelaysLogComputeUnitsFieldSizeBytes is the size of the little-endian uint64
	// field that captures the mined relay's weight (compute units). This participates in the
	// SMST sum and is critical for accurate accounting.
	minedRelaysLogComputeUnitsFieldSizeBytes = 8
)

// On-disk record layout (per mined relay), in little-endian order:
// - [4 bytes]  PayloadLength: uint32, number of bytes in RelayPayload
// - [8 bytes]  ComputeUnits:  uint64, weight of this relay
// - [N bytes]  RelayHash:     fixed-size hash (protocol.RelayHasherSize)
// - [L bytes]  RelayPayload:  opaque bytes, length = PayloadLength
//
// Example replay loop during recovery:
// - Read 4 bytes -> L
// - Read 8 bytes -> CU
// - Read N bytes -> H
// - Read L bytes -> P
// - trie.Update(H, P, CU)

// minedRelaysWriteAheadLog provides an append-only, length-prefixed, write-ahead log (WAL) for mined relays.
//
// Why it exists:
// - The SMST lives in memory for performance and would be lost on crash or restart
// - WAL ensures crash recovery by persisting relay evidence to disk
//
// How it works:
// - Buffer entries in memory to reduce I/O overhead
// - Flush to disk either:
//   - Periodically via timer (every minedRelaysLogFlushInterval)
//   - Immediately when buffer exceeds memory threshold (maxBufferedMinedRelaysBytesBeforeFlush)
// - On shutdown: flush remaining entries
// - On restart: replay WAL to rebuild SMST state deterministically
//
// Lifecycle:
// 1. NewMinedRelaysWriteAheadLog() - Opens/creates WAL file, starts flush timer
// 2. AppendMinedRelay() - Adds relay to buffer (called per mined relay)
// 3. Close() - Flushes buffer, closes file (keeps WAL for potential replay)
// 4. CloseAndRemove() - Flushes, closes, and deletes WAL (after successful settlement)
//
// File location: <smtStoresBasePath>/mined_relays/<supplierAddr>/<sessionId>.wal
type minedRelaysWriteAheadLog struct {
	logger polylog.Logger

	// bufferedLogBytes holds serialized log entries until they are flushed to disk.
	bufferedLogBytes []byte

	// bufferMu guards bufferedLogBytes from concurrent access.
	bufferMu sync.Mutex

	// logFile is the append-only file where buffered entries are persisted.
	logFile *os.File

	// flushTicker triggers periodic flushes of the in-memory buffer to disk.
	flushTicker *time.Ticker
}

// NewMinedRelaysWriteAheadLog constructs a new minedRelaysWriteAheadLog, opening (or creating)
// the underlying append-only file at the given path and starting the periodic flush loop.
//
// Paired with the SessionTree's in-memory SMST:
// - Call this once per newly created SessionTree to start capturing mined relays
// - For each mined relay, append to the WAL and also update the in-memory SMST
// - Keep the returned WAL around for the duration of the session
// - On clean shutdown, flush; on crash, replay to rebuild the in-memory SMST
//
// Errors indicate we could not open the on-disk log (permissions, full disk, etc.).
func NewMinedRelaysWriteAheadLog(logFilePath string, logger polylog.Logger) (*minedRelaysWriteAheadLog, error) {
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		logger.Error().Err(err).Msg("❌️ Failed to open mined relays WAL file for appending. ❗Check disk space and permissions. ❗Relay evidence may be lost on restart.")
		return nil, err
	}

	wal := &minedRelaysWriteAheadLog{
		bufferedLogBytes: []byte{},
		logFile:          logFile,
		logger:           logger,
		flushTicker:      time.NewTicker(minedRelaysLogFlushInterval),
	}

	// Start the background periodic flush loop
	go wal.runPeriodicFlushLoop()

	return wal, nil
}

// AppendMinedRelay serializes a mined relay entry and appends it to the in-memory buffer.
// If the buffer exceeds the configured threshold, it is flushed to disk immediately.
//
// What happens (and why it matters for backup):
//   - Encode a single mined relay (length || compute units || hash || payload)
//   - Append to the in-memory buffer for batching (fast path)
//   - If the buffer is getting large, reset the periodic timer and force a flush
//     so recent relays are durably recorded on disk in case the process dies
//
func (wal *minedRelaysWriteAheadLog) AppendMinedRelay(relayHash []byte, relayPayload []byte, computeUnitsPerRelay uint64) {
	serializedEntry := encodeMinedRelaysLogEntry(relayHash, relayPayload, computeUnitsPerRelay)

	wal.bufferMu.Lock()
	wal.bufferedLogBytes = append(wal.bufferedLogBytes, serializedEntry...)
	bufferSize := len(wal.bufferedLogBytes)
	wal.bufferMu.Unlock()

	if bufferSize > maxBufferedMinedRelaysBytesBeforeFlush {
		wal.flushTicker.Reset(minedRelaysLogFlushInterval)
		if err := wal.flushBufferToDisk(); err != nil {
			wal.logger.Error().Err(err).Msg("❌️ Failed to flush mined relays WAL buffer to disk after exceeding size threshold")
		}
	}
}

// Close flushes any buffered entries, closes the underlying file.
//
// This is called when the relay miner is shutting down but the session may still be active.
// - Flush persists any pending entries
// - Close releases the OS file handle
func (wal *minedRelaysWriteAheadLog) Close() error {
	wal.flushTicker.Stop()
	if err := wal.flushBufferToDisk(); err != nil {
		return err
	}

	return wal.logFile.Close()
}

// CloseAndRemove flushes any buffered entries, closes the underlying file, and removes it from disk.
//
// This is called when the session is fully settled and the WAL is no longer needed.
// - Flush persists any pending entries
// - Close releases the OS file handle
// - Remove deletes the WAL file once the on-chain claim/proof process has made the backup redundant
func (wal *minedRelaysWriteAheadLog) CloseAndRemove() error {
	if err := wal.Close(); err != nil {
		return err
	}

	return os.Remove(wal.logFile.Name())
}

// flushBufferToDisk writes the buffered frames to the WAL file and syncs the data to disk.
//
// Critical for durability: This function calls Sync() after Write() to ensure data hits
// physical disk, not just the OS page cache. Without Sync(), a crash immediately after
// Write() could lose recent entries, defeating the WAL's crash recovery guarantee.
func (wal *minedRelaysWriteAheadLog) flushBufferToDisk() error {
	wal.bufferMu.Lock()
	defer wal.bufferMu.Unlock()

	// Nothing to flush, return early
	if len(wal.bufferedLogBytes) == 0 {
		return nil
	}

	buffered := wal.bufferedLogBytes
	wal.bufferedLogBytes = []byte{}

	if _, err := wal.logFile.Write(buffered); err != nil {
		wal.logger.Error().Err(err).Msg("❌️ Failed to write mined relays WAL entries to file. ❗Check disk space and permissions. ❗Relay evidence may be lost on restart.")
		return err
	}

	// Sync to disk to ensure durability (fsync). This is critical for crash recovery.
	// Without it, writes may sit in OS page cache and never hit disk on abrupt shutdown.
	if err := wal.logFile.Sync(); err != nil {
		wal.logger.Error().Err(err).Msg("❌️ Failed to sync mined relays WAL to disk. ❗Relay evidence may be lost on crash.")
		return err
	}

	wal.logger.Info().Int("size_bytes", len(buffered)).Msg("✅️ Successfully flushed mined relays WAL buffer to disk.")

	return nil
}

// runPeriodicFlushLoop wakes up on every ticker tick to flush any pending buffered entries.
func (wal *minedRelaysWriteAheadLog) runPeriodicFlushLoop() {
	for range wal.flushTicker.C {
		if err := wal.flushBufferToDisk(); err != nil {
			wal.logger.Error().Err(err).Msg("❌️ Periodic flush of mined relays WAL buffer failed")
		}
	}
}

// reconstructSMTFromMinedRelaysLog replays the mined relays write-ahead log from disk
// to reconstruct the SMST representing the session state at crash/restart.
//
// Since the SMST is in-memory by design (for speed), replaying the WAL is how
// we recover exactly the same state after a crash or restart, avoiding mined relay loss
// and guaranteeing the same ordering of the inserted relays which is critical for correct proof generation.
//
// Rebuilding an in-memory SMST backup from the WAL involves:
// - Open the WAL file that contains all previously mined relays
// - For each relay, read fields in order and feed them into a fresh, in-memory SMST via Update
// - Stop on clean EOF; any other read error is logged/returned
// - The resulting trie represents the exact pre-crash state (deterministic replay)
func reconstructSMTFromMinedRelaysLog(
	minedRelaysLogFilePath string,
	treeStore kvstore.MapStore,
	logger polylog.Logger,
) (*smt.SMST, error) {
	file, err := os.Open(minedRelaysLogFilePath)
	if err != nil {
		logger.Error().Err(err).Msg("❌️ Failed to open mined relays WAL file for reading.")
		return nil, err
	}
	defer file.Close()

	// Create a new in-memory SMST backed by a SimpleMap and populate it by replaying the WAL
	trie := smt.NewSparseMerkleSumTrie(treeStore, protocol.NewTrieHasher(), protocol.SMTValueHasher())

	for {
		lengthPrefixBz, err := readExactlyNBytes(file, minedRelaysLogRelayPayloadLengthPrefixSizeBytes, logger)
		// Encountered clean EOF, done reading
		if err == io.EOF {
			break
		}

		// Any other read error is logged and returned, the replay has failed
		if err != nil {
			return nil, err
		}
		// Parse the mined relay bytes length
		relayBytesCount := binary.LittleEndian.Uint32(lengthPrefixBz)

		// Read compute units
		computeUnitsBz, err := readExactlyNBytes(file, minedRelaysLogComputeUnitsFieldSizeBytes, logger)
		if err != nil {
			return nil, err
		}
		computeUnits := binary.LittleEndian.Uint64(computeUnitsBz)

		// Read the mined relay hash
		relayHashBz, err := readExactlyNBytes(file, minedRelaysLogRelayHashSizeBytes, logger)
		if err != nil {
			return nil, err
		}

		// Read the mined relay bytes
		relayPayload, err := readExactlyNBytes(file, int(relayBytesCount), logger)
		if err != nil {
			return nil, err
		}

		// Update the SMST with the mined relay in the same order they were originally added
		// This is critical for deterministic replay and correct proof generation.
		if err := trie.Update(relayHashBz, relayPayload, computeUnits); err != nil {
			return nil, err
		}

	}

	return trie, nil
}

// readExactlyNBytes reads exactly expected bytes or returns an error.
// - Returns io.EOF if no bytes are read and the file is at EOF
// - Logs an error when fewer bytes than expected are read for any other error
func readExactlyNBytes(file *os.File, expected int, logger polylog.Logger) ([]byte, error) {
	buf := make([]byte, expected)
	n, err := io.ReadFull(file, buf)
	// Clean EOF, no more data to read
	if err == io.EOF && n == 0 {
		return nil, err
	}

	// Any other error (including EOF with bytes read) is logged and returned
	if err != nil {
		logger.Error().Err(err).Int("expected_bytes", expected).Int("read_bytes", n).Msg("❌️ Failed to read from mined relays WAL file.")
		return nil, err
	}

	return buf, nil
}

// encodeMinedRelaysLogEntry serializes a single mined relay in the WAL format.
// Format (little-endian):
// - [4]  payload length (uint32)
// - [8]  compute units (uint64)
// - [N]  relay hash (fixed-size)
// - [L]  payload bytes (opaque)
//
// - Fixed fields first enable simple and robust replay
// - Length prefix allows payloads of varying sizes without delimiters
// - Little-endian matches Go's binary package defaults and our protocol usage
func encodeMinedRelaysLogEntry(relayHash []byte, relayPayload []byte, computeUnitsPerRelay uint64) []byte {
	lengthPrefixBz := make([]byte, minedRelaysLogRelayPayloadLengthPrefixSizeBytes)
	binary.LittleEndian.PutUint32(lengthPrefixBz, uint32(len(relayPayload)))

	computeUnitsBz := make([]byte, minedRelaysLogComputeUnitsFieldSizeBytes)
	binary.LittleEndian.PutUint64(computeUnitsBz, computeUnitsPerRelay)

	totalEntryLen := minedRelaysLogRelayPayloadLengthPrefixSizeBytes + minedRelaysLogComputeUnitsFieldSizeBytes + minedRelaysLogRelayHashSizeBytes + len(relayPayload)
	entry := make([]byte, 0, totalEntryLen)

	entry = append(entry, lengthPrefixBz...)
	entry = append(entry, computeUnitsBz...)
	entry = append(entry, relayHash...)
	entry = append(entry, relayPayload...)

	return entry
}
