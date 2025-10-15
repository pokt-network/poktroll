package session

import (
	"crypto/rand"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore/simplemap"
)

func TestMinedRelaysWAL_CreateAndCloseFlushes(t *testing.T) {
	// Create a temporary WAL file and ensure it's cleaned up at the end.
	wal, walPath, cleanup := createTempWal(t)
	defer cleanup()

	// Append a single mined relay but do NOT exceed the in-memory flush threshold,
	// so the WAL should still be buffered (not yet flushed to disk) until Close().
	relayHash := randomBytes(t, 32)
	relayPayload := randomBytes(t, 16)
	wal.AppendMinedRelay(relayHash, relayPayload, 7)

	// The file should still be empty (buffer not flushed yet) because the threshold (10MB)
	// was not exceeded and the periodic 10s ticker hasn't fired.
	walFileInfoBeforeClose, statErr := os.Stat(walPath)
	require.NoError(t, statErr)
	// Size may be zero or very small if a periodic flush raced; accept both by capturing size before close.
	sizeBeforeClose := walFileInfoBeforeClose.Size()
	require.LessOrEqual(t, sizeBeforeClose, int64(0)) // expecting zero normally

	// Close triggers flush
	require.NoError(t, wal.Close())
	walFileInfoAfterClose, statErr := os.Stat(walPath)
	require.NoError(t, statErr)
	require.Greater(t, walFileInfoAfterClose.Size(), sizeBeforeClose, "file size should grow after Close flush")
}

func TestMinedRelaysWAL_CloseAndRemove(t *testing.T) {
	// Create a temporary WAL file.
	wal, walPath, _ := createTempWal(t)
	relayHash := randomBytes(t, 32)
	relayPayload := randomBytes(t, 8)
	wal.AppendMinedRelay(relayHash, relayPayload, 1)

	// Close and remove the WAL file.
	require.NoError(t, wal.CloseAndRemove())

	// Ensure the file is removed.
	_, statErr := os.Stat(walPath)
	require.True(t, os.IsNotExist(statErr), "file should be removed")
}

func TestEncodeMinedRelaysLogEntry_Format(t *testing.T) {
	// Prepare a single WAL entry (hash + payload + compute units) and validate its binary layout.
	relayHash := randomBytes(t, 32)
	relayPayload := randomBytes(t, 50)
	computeUnits := uint64(123456789)

	// Encode the entry
	entry := encodeMinedRelaysLogEntry(relayHash, relayPayload, computeUnits)

	// Parse back the entry to validate: [payload_len(4)][cu(8)][hash(32)][payload(n)]

	// Validate payload length
	payloadLen := binary.LittleEndian.Uint32(entry[:4])
	require.Equal(t, uint32(len(relayPayload)), payloadLen)

	// Validate compute units
	cu := binary.LittleEndian.Uint64(entry[4:12])
	require.Equal(t, computeUnits, cu)

	// Validate hash
	decodedHash := entry[12 : 12+len(relayHash)]
	require.Equal(t, relayHash, decodedHash)

	// Validate payload
	decodedPayload := entry[12+len(relayHash):]
	require.Equal(t, relayPayload, decodedPayload)
}

func TestReadExactlyNBytes_SuccessAndEOF(t *testing.T) {
	logger := newTestLogger()
	tmpFile, err := os.CreateTemp(t.TempDir(), "waltest")
	require.NoError(t, err)
	defer tmpFile.Close()

	// Write some content to the temporary file.
	content := []byte("abcdef")
	_, err = tmpFile.Write(content)
	require.NoError(t, err)
	_, err = tmpFile.Seek(0, io.SeekStart)
	require.NoError(t, err)

	// Read first 3 bytes and verify.
	chunk, err := readExactlyNBytes(tmpFile, 3, logger)
	require.NoError(t, err)
	require.Equal(t, []byte("abc"), chunk)

	// Read next 3 bytes and verify.
	chunk, err = readExactlyNBytes(tmpFile, 3, logger)
	require.NoError(t, err)
	require.Equal(t, []byte("def"), chunk)

	// Now further reads should return a clean EOF.
	_, err = readExactlyNBytes(tmpFile, 1, logger)
	require.ErrorIs(t, err, io.EOF)
}

func TestReadExactlyNBytes_ShortReadError(t *testing.T) {
	logger := newTestLogger()
	tmpFile, err := os.CreateTemp(t.TempDir(), "waltest")
	require.NoError(t, err)
	defer tmpFile.Close()

	// Write some content to the temporary file.
	_, err = tmpFile.Write([]byte{0x01, 0x02})
	require.NoError(t, err)
	_, err = tmpFile.Seek(0, io.SeekStart)
	require.NoError(t, err)

	// Request more bytes than present -> should error (not clean EOF because some bytes read).
	_, err = readExactlyNBytes(tmpFile, 4, logger)
	require.Error(t, err)
	require.NotErrorIs(t, err, io.EOF)
}

func TestReconstructSMTFromMinedRelaysLog_BasicReplay(t *testing.T) {
	// Create a WAL and a reference SMT that we'll update in lockstep.
	wal, walPath, cleanup := createTempWal(t)
	defer cleanup()

	// Build a reference trie simultaneously using an in-memory KV store.
	referenceKVStore := simplemap.NewSimpleMap()
	expectedTrie := smt.NewSparseMerkleSumTrie(referenceKVStore, protocol.NewTrieHasher(), protocol.SMTValueHasher())

	// Track all appended relays to verify they can be retrieved after replay
	type minedRelay struct {
		hash    []byte
		payload []byte
		cu      uint64
	}
	appendedRelays := make([]minedRelay, 0, 5)

	// Append relays to the WAL and update the reference trie.
	for i := range 5 {
		relayHash := randomBytes(t, 32)
		relayPayload := randomBytes(t, 10+i)
		cu := uint64(i + 1)

		// Update WAL and reference trie identically.
		wal.AppendMinedRelay(relayHash, relayPayload, cu)
		expectedTrie.Update(relayHash, relayPayload, cu)
		appendedRelays = append(appendedRelays, minedRelay{relayHash, relayPayload, cu})
	}
	require.NoError(t, wal.Close())

	// Reconstruct a trie from the WAL log to verify the replay logic.
	replayKVStore := simplemap.NewSimpleMap()
	reconstructedTrie, err := reconstructSMTFromMinedRelaysLog(walPath, replayKVStore, newTestLogger())
	require.NoError(t, err)

	// Compare roots
	require.Equal(t, expectedTrie.Root(), reconstructedTrie.Root())

	// Spot check a couple leaves by querying values.
	for _, e := range appendedRelays[:2] { // first two
		value, _, err := reconstructedTrie.Get(e.hash)
		require.NoError(t, err)
		require.Equal(t, e.payload, value)
	}
}

func TestMinedRelaysWAL_ConcurrentAppends(t *testing.T) {
	// Ensure WAL correctly handles concurrent appends and the replayed count matches.
	wal, walPath, cleanup := createTempWal(t)
	defer cleanup()

	numRelays := 50

	// Use a WaitGroup to wait for all goroutines to finish.
	var appendWG sync.WaitGroup
	for i := range numRelays {
		appendWG.Add(1)
		go func(i int) {
			relayHash := randomBytes(t, 32)
			payload := []byte{byte(i)}
			wal.AppendMinedRelay(relayHash, payload, uint64(i+1))
			appendWG.Done()
		}(i)
	}

	appendWG.Wait()
	require.NoError(t, wal.Close())

	// Reconstruct a trie from the WAL log to verify the replay logic.
	replayKVStore := simplemap.NewSimpleMap()
	reconstructedTrie, err := reconstructSMTFromMinedRelaysLog(walPath, replayKVStore, newTestLogger())
	require.NoError(t, err)

	// Count should equal number of appended relays
	root := reconstructedTrie.Root()
	totalMinedRelays, err := smt.MerkleSumRoot(root).Count()
	require.NoError(t, err)
	require.Equal(t, uint64(numRelays), totalMinedRelays)
}

func TestMinedRelaysWAL_LargeEntryTriggersImmediateFlushOnClose(t *testing.T) {
	// Even with a "large" entry (still below threshold), the WAL should flush on Close().
	wal, walPath, cleanup := createTempWal(t)
	defer cleanup()

	// Create a large payload bigger than threshold/2 to ensure it's still buffered until Close().
	// Since the threshold is large (10MB), we can't exceed it quickly in CI.
	payload := randomBytes(t, 1024) // 1KB
	relayHash := randomBytes(t, 32)
	wal.AppendMinedRelay(relayHash, payload, 42)

	// Wait small duration to ensure periodic tick (10s) has not yet occurred.
	time.Sleep(50 * time.Millisecond)
	walFileInfoBeforeClose, _ := os.Stat(walPath)
	sizeBeforeClose := walFileInfoBeforeClose.Size()
	require.Equal(t, sizeBeforeClose, int64(0))

	// Close triggers flush
	require.NoError(t, wal.Close())

	// File size should now be greater than before Close().
	walFileInfoAfterClose, _ := os.Stat(walPath)
	require.Greater(t, walFileInfoAfterClose.Size(), sizeBeforeClose)
}

// Test that the periodic timer causes a flush even when threshold isn't hit.
func TestMinedRelaysWAL_PeriodicTimerFlushes(t *testing.T) {
	// Use the default ticker (10s). We'll wait a bit over 10 seconds to allow the periodic flush.
	wal, walPath, cleanup := createTempWal(t)
	defer cleanup()

	// Append a small entry that won't trigger threshold flush.
	relayHash := randomBytes(t, 32)
	payload := []byte("hello")
	wal.AppendMinedRelay(relayHash, payload, 1)

	// Immediately after append the file should still be empty.
	fileStat, err := os.Stat(walPath)
	require.NoError(t, err)
	require.Equal(t, int64(0), fileStat.Size())

	// Wait a bit over the 10s periodic interval so the flush loop runs.
	time.Sleep(11 * time.Second)

	// Now the file should have data because the timer triggered a flush.
	fileStat, err = os.Stat(walPath)
	require.NoError(t, err)
	require.Greater(t, fileStat.Size(), int64(0))
}

// Test that exceeding the in-memory buffer threshold triggers an immediate flush.
func TestMinedRelaysWAL_ThresholdFlush(t *testing.T) {
	wal, walPath, cleanup := createTempWal(t)
	defer cleanup()

	// Create payload larger than the threshold to trigger immediate flush in AppendMinedRelay.
	bigPayload := make([]byte, maxBufferedMinedRelaysBytesBeforeFlush+1_024)
	_, err := rand.Read(bigPayload)
	require.NoError(t, err)

	relayHash := randomBytes(t, 32)
	wal.AppendMinedRelay(relayHash, bigPayload, 99)

	// After append, threshold exceeded path should have flushed buffer synchronously.
	// Assert file now has non-zero size and in-memory buffer is empty.
	walFileInfo, err := os.Stat(walPath)
	require.NoError(t, err)
	require.Greater(t, walFileInfo.Size(), int64(0), "expected file to have contents after threshold-triggered flush")
	require.Equal(t, 0, len(wal.bufferedLogBytes), "expected in-memory buffer to be cleared after flush")
}

// randomBytes returns a slice of random bytes of the given length.
func randomBytes(t *testing.T, n int) []byte {
	b := make([]byte, n)
	_, err := rand.Read(b)
	require.NoError(t, err)
	return b
}

// newTestLogger returns a quiet test logger.
func newTestLogger() polylog.Logger {
	return polyzero.NewLogger(polyzero.WithLevel(polyzero.ErrorLevel))
}

// createTempWal creates a new WAL in a temporary directory and returns it plus a cleanup func.
func createTempWal(t *testing.T) (*minedRelaysWriteAheadLog, string, func()) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "mined_relays.wal")
	wal, err := NewMinedRelaysWriteAheadLog(filePath, newTestLogger())
	require.NoError(t, err)
	cleanup := func() { _ = wal.Close(); _ = os.RemoveAll(dir) }
	return wal, filePath, cleanup
}
