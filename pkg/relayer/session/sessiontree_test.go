package session_test

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore/pebble"
	"github.com/pokt-network/smt/kvstore/simplemap"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/session"
	"github.com/pokt-network/poktroll/testutil/sample"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

const (
	// Test multiple SMST sizes to see how the compaction ratio changes when the number
	// of leaves increases.
	// maxLeafs is the maximum number of leaves to test, after which the test stops.
	maxLeafs = 10000
	// Since the inserted leaves are random, we run the test for a given leaf count
	// multiple times to remove the randomness bias.
	numIterations = 100

	// Directory path for the mined relays WAL files
	minedRelaysWALDirectoryPath = "mined_relays_wal"
)

// No significant performance gains were observed when using compact proofs compared
// to non-compact proofs.
// In fact, compact proofs appear to be less efficient than gzipped proofs, even
// without considering the "proof closest value" compression.
// For a sample comparison between compression and compaction ratios, see:
// https://github.com/pokt-network/poktroll/pull/823#issuecomment-2363987920
func TestSessionTree_CompactProofsAreSmallerThanNonCompactProofs(t *testing.T) {
	// Run the test for different number of leaves.
	for numLeafs := 10; numLeafs <= maxLeafs; numLeafs *= 10 {
		cumulativeProofSize := 0
		cumulativeCompactProofSize := 0
		cumulativeGzippedProofSize := 0
		// We run the test numIterations times for each number of leaves to remove the randomness bias.
		for iteration := 0; iteration <= numIterations; iteration++ {
			kvStore, err := pebble.NewKVStore("")
			require.NoError(t, err)

			trie := smt.NewSparseMerkleSumTrie(kvStore, protocol.NewTrieHasher(), protocol.SMTValueHasher())

			// Insert numLeaf random leaves.
			for i := 0; i < numLeafs; i++ {
				key := make([]byte, 32)
				_, err = rand.Read(key) //nolint:staticcheck // Using rand.Read in tests as a deterministic pseudo-random source is okay.
				require.NoError(t, err)
				// Insert an empty value since this does not get affected by the compaction,
				// this is also to not favor proof compression that compresses the value too.
				trie.Update(key, []byte{}, 1)
			}

			// Generate a random path.
			var path = make([]byte, 32)
			_, err = rand.Read(path) //nolint:staticcheck // Using rand.Read in tests as a deterministic pseudo-random source is okay.
			require.NoError(t, err)

			// Create the proof.
			proof, err := trie.ProveClosest(path)
			require.NoError(t, err)

			proofBz, err := proof.Marshal()
			require.NoError(t, err)

			// Accumulate the proof size over numIterations runs.
			cumulativeProofSize += len(proofBz)

			// Generate the compacted proof.
			compactProof, err := smt.CompactClosestProof(proof, &trie.TrieSpec)
			require.NoError(t, err)

			compactProofBz, err := compactProof.Marshal()
			require.NoError(t, err)

			// Accumulate the compact proof size over numIterations runs.
			cumulativeCompactProofSize += len(compactProofBz)

			// Gzip the non compacted proof.
			var buf bytes.Buffer
			gzipWriter := gzip.NewWriter(&buf)
			_, err = gzipWriter.Write(proofBz)
			require.NoError(t, err)
			err = gzipWriter.Close()
			require.NoError(t, err)

			// Accumulate the gzipped proof size over numIterations runs.
			cumulativeGzippedProofSize += len(buf.Bytes())
		}

		// Calculate how much more efficient compact SMT proofs are compared to non-compact proofs.
		compactionRatio := float32(cumulativeProofSize) / float32(cumulativeCompactProofSize)

		// Claculate how much more efficient gzipped proofs are compared to non-compact proofs.
		compressionRatio := float32(cumulativeProofSize) / float32(cumulativeGzippedProofSize)

		// Gzip compression is more efficient than SMT compaction.
		require.Greater(t, compressionRatio, compactionRatio)

		t.Logf(
			"numLeaf=%d: compactionRatio: %f, compressionRatio: %f",
			numLeafs, compactionRatio, compressionRatio,
		)
	}
}

// WAL (Write-Ahead Log) Tests
// These tests verify that the SessionTree properly integrates with the WAL
// to ensure mined relays are persisted to disk and can survive crashes/restarts.

// TestSessionTree_WAL_Update_AppendsToWAL verifies that Update() appends mined relays to the WAL
func TestSessionTree_WAL_Update_AppendsToWAL(t *testing.T) {
	// Create a new session tree
	sessionTree, walPath, cleanup := createTestSessionTree(t)
	defer cleanup()

	// Update with a single relay
	relayHash := randomBytes(t, 32)
	relayPayload := randomBytes(t, 100)
	computeUnits := uint64(42)
	err := sessionTree.Update(relayHash, relayPayload, computeUnits)
	require.NoError(t, err)

	// Close the session tree to flush WAL
	require.NoError(t, sessionTree.Close())

	// Verify the WAL file exists and has content
	walFileInfo, err := os.Stat(walPath)
	require.NoError(t, err)
	require.Greater(t, walFileInfo.Size(), int64(0), "WAL file should contain data after Update()")

	// Verify WAL contents by reconstructing the SMT

	originalRoot := sessionTree.GetSMSTRoot()

	treeStore := simplemap.NewSimpleMap()
	logger := polyzero.NewLogger()
	reconstructedTrie, err := session.ReconstructSMTFromMinedRelaysLog(walPath, treeStore, logger)
	require.NoError(t, err, "Should be able to reconstruct SMT from WAL")

	reconstructedRoot := reconstructedTrie.Root()
	require.Equal(t, []byte(originalRoot), []byte(reconstructedRoot), "Reconstructed SMT root should match original")
}

// TestSessionTree_WAL_Update_MultipleRelays verifies that multiple Update() calls
// append all relays to the WAL
func TestSessionTree_WAL_Update_MultipleRelays(t *testing.T) {
	sessionTree, walPath, cleanup := createTestSessionTree(t)
	defer cleanup()

	// Add multiple relays
	numRelays := 10
	for i := range numRelays {
		relayHash := randomBytes(t, 32)
		relayPayload := randomBytes(t, 50+i)
		cu := uint64((i + 1) * 100)

		err := sessionTree.Update(relayHash, relayPayload, cu)
		require.NoError(t, err)
	}

	// Close to flush WAL
	require.NoError(t, sessionTree.Close())

	// Verify WAL file exists and has substantial content
	walFileInfo, err := os.Stat(walPath)
	require.NoError(t, err)
	require.Greater(t, walFileInfo.Size(), int64(0), "WAL file should contain all relays")

	// Verify WAL contents by reconstructing the SMT

	originalRoot := sessionTree.GetSMSTRoot()

	treeStore := simplemap.NewSimpleMap()
	logger := polyzero.NewLogger()
	reconstructedTrie, err := session.ReconstructSMTFromMinedRelaysLog(walPath, treeStore, logger)
	require.NoError(t, err, "Should be able to reconstruct SMT from WAL")

	reconstructedRoot := reconstructedTrie.Root()
	require.Equal(t, []byte(originalRoot), []byte(reconstructedRoot), "Reconstructed SMT root should match original for all relays")
}

// TestSessionTree_WAL_Flush_PreservesWAL verifies that Flush() doesn't affect the WAL
func TestSessionTree_WAL_Flush_PreservesWAL(t *testing.T) {
	sessionTree, walPath, cleanup := createTestSessionTree(t)
	defer cleanup()

	// Add relays
	for i := range 5 {
		relayHash := randomBytes(t, 32)
		relayPayload := randomBytes(t, 50)
		err := sessionTree.Update(relayHash, relayPayload, uint64(i+1)*10)
		require.NoError(t, err)
	}

	// Get root before flush
	rootBeforeFlush := sessionTree.GetSMSTRoot()
	require.NotNil(t, rootBeforeFlush)

	// Flush
	claimedRoot, err := sessionTree.Flush()
	require.NoError(t, err)
	require.NotNil(t, claimedRoot)
	require.Equal(t, []byte(rootBeforeFlush), claimedRoot, "Claimed root should match root before flush")

	// Close to ensure WAL is flushed to disk
	require.NoError(t, sessionTree.Close())

	// Verify WAL file still exists with content
	walFileInfo, err := os.Stat(walPath)
	require.NoError(t, err)
	require.Greater(t, walFileInfo.Size(), int64(0), "WAL should be preserved after Flush()")

	// Verify WAL contents match the claimed root by reconstructing

	treeStore := simplemap.NewSimpleMap()

	logger := polyzero.NewLogger()
	reconstructedTrie, err := session.ReconstructSMTFromMinedRelaysLog(walPath, treeStore, logger)
	require.NoError(t, err, "Should be able to reconstruct SMT from WAL after Flush()")

	reconstructedRoot := reconstructedTrie.Root()
	require.Equal(t, claimedRoot, []byte(reconstructedRoot), "WAL should reconstruct to same root as claimed root")
}

// TestSessionTree_WAL_Close_FlushesBuffer verifies that Close() flushes the WAL buffer to disk
func TestSessionTree_WAL_Close_FlushesBuffer(t *testing.T) {
	sessionTree, walPath, cleanup := createTestSessionTree(t)
	defer cleanup()

	// Add a relay (will be buffered in WAL)
	relayHash := randomBytes(t, 32)
	relayPayload := randomBytes(t, 50)
	err := sessionTree.Update(relayHash, relayPayload, 100)
	require.NoError(t, err)

	// Before Close, the WAL file might not exist or be empty (buffered)
	// After Close, it must be flushed to disk
	require.NoError(t, sessionTree.Close())

	// Verify WAL file has exact expected content
	walFileInfo, err := os.Stat(walPath)
	require.NoError(t, err)
	expectedSize := int64(4 + 8 + 32 + 50) // 94 bytes
	require.Equal(t, expectedSize, walFileInfo.Size(), "WAL should be flushed to disk after Close()")
}

// TestSessionTree_WAL_Delete_RemovesWAL verifies that Delete() removes the WAL file
func TestSessionTree_WAL_Delete_RemovesWAL(t *testing.T) {
	sessionTree, walPath, _ := createTestSessionTree(t)
	// Don't use defer cleanup() because we want to test Delete() directly

	// Add a relay
	relayHash := randomBytes(t, 32)
	relayPayload := randomBytes(t, 50)
	err := sessionTree.Update(relayHash, relayPayload, 100)
	require.NoError(t, err)

	// Don't call Close() here - Delete() should handle cleanup directly
	// Verify WAL file exists
	_, err = os.Stat(walPath)
	require.NoError(t, err, "WAL file should exist before Delete()")

	// Delete should close and remove the WAL file
	require.NoError(t, sessionTree.Delete())

	// Verify the WAL file is removed
	_, err = os.Stat(walPath)
	require.True(t, os.IsNotExist(err), "WAL file should be removed after Delete()")
}

// TestSessionTree_WAL_NewSessionTree_CreatesWALDirectory verifies that NewSessionTree()
// creates the WAL directory if it doesn't exist
func TestSessionTree_WAL_NewSessionTree_CreatesWALDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	logger := polyzero.NewLogger()

	sessionHeader := &sessiontypes.SessionHeader{
		SessionId:               "test_session",
		ApplicationAddress:      sample.AccAddressBech32(),
		ServiceId:               "test_service",
		SessionStartBlockHeight: 1,
		SessionEndBlockHeight:   100,
	}
	supplierAddr := sample.AccAddressBech32()

	// Create session tree - should create WAL directory
	sessionTree, err := session.NewSessionTree(logger, sessionHeader, supplierAddr, tmpDir)
	require.NoError(t, err)
	require.NotNil(t, sessionTree)

	// Verify directory was created
	expectedWALDir := filepath.Join(tmpDir, minedRelaysWALDirectoryPath, supplierAddr, sessionHeader.SessionId)
	_, err = os.Stat(filepath.Dir(expectedWALDir))
	require.NoError(t, err, "WAL directory should be created")

	// Cleanup
	require.NoError(t, sessionTree.Delete())
}

// TestSessionTree_WAL_ConcurrentUpdates verifies that concurrent Update() calls
// are handled safely (mutex protection)
func TestSessionTree_WAL_ConcurrentUpdates(t *testing.T) {
	sessionTree, walPath, cleanup := createTestSessionTree(t)
	defer cleanup()

	// Number of concurrent goroutines
	numGoroutines := 10
	relaysPerGoroutine := 10

	// Channel to signal completion
	done := make(chan bool, numGoroutines)

	// Start concurrent updates
	for g := range numGoroutines {
		go func(goroutineID int) {
			for i := range relaysPerGoroutine {
				relayHash := randomBytes(t, 32)
				relayPayload := randomBytes(t, 50)
				cu := uint64(goroutineID*100 + i)
				err := sessionTree.Update(relayHash, relayPayload, cu)
				require.NoError(t, err)
			}
			done <- true
		}(g)
	}

	// Wait for all goroutines to complete
	for range numGoroutines {
		<-done
	}

	// Verify count
	smtRoot := sessionTree.GetSMSTRoot()
	count, err := smtRoot.Count()
	require.NoError(t, err)
	expectedCount := uint64(numGoroutines * relaysPerGoroutine)
	require.Equal(t, expectedCount, count, "All concurrent updates should be recorded")

	// Close and verify all relays were persisted to WAL
	require.NoError(t, sessionTree.Close())
	walFileInfo, err := os.Stat(walPath)
	require.NoError(t, err)
	require.Greater(t, walFileInfo.Size(), int64(0), "WAL should contain all concurrent updates")

	// Verify all concurrent updates are correctly stored by reconstructing the SMT

	originalRoot := sessionTree.GetSMSTRoot()

	treeStore := simplemap.NewSimpleMap()
	logger := polyzero.NewLogger()
	reconstructedTrie, err := session.ReconstructSMTFromMinedRelaysLog(walPath, treeStore, logger)
	require.NoError(t, err, "Should be able to reconstruct SMT from WAL after concurrent updates")

	reconstructedRoot := reconstructedTrie.Root()
	require.Equal(t, []byte(originalRoot), []byte(reconstructedRoot), "Reconstructed SMT should match original after concurrent updates")
}

// TestSessionTree_WAL_LargePayloads verifies WAL handles large relay payloads
func TestSessionTree_WAL_LargePayloads(t *testing.T) {
	sessionTree, walPath, cleanup := createTestSessionTree(t)
	defer cleanup()

	// Add relays with large payloads
	largeSizes := []int{1024, 10240, 102400, 1048576} // 1KB, 10KB, 100KB, 1MB
	for i, size := range largeSizes {
		relayHash := randomBytes(t, 32)
		relayPayload := randomBytes(t, size)
		cu := uint64((i + 1) * 1000)
		err := sessionTree.Update(relayHash, relayPayload, cu)
		require.NoError(t, err)
	}

	// Note: With 1MB payload, the total size exceeds maxBufferedMinedRelaysBytesBeforeFlush (10MB)
	// However, since we need to add multiple large relays to exceed 10MB, let's verify
	// that Close() properly flushes all the data
	require.NoError(t, sessionTree.Close())

	// Verify WAL file exists and has substantial size
	walFileInfo, err := os.Stat(walPath)
	require.NoError(t, err)
	// Expect at least the sum of payload sizes plus overhead (4 + 8 + 32 per relay)
	minExpectedSize := int64(1024 + 10240 + 102400 + 1048576 + 4*44)
	require.GreaterOrEqual(t, walFileInfo.Size(), minExpectedSize, "Large payloads should be written to WAL")

	// Verify large payloads are correctly stored by reconstructing the SMT

	originalRoot := sessionTree.GetSMSTRoot()

	treeStore := simplemap.NewSimpleMap()
	logger := polyzero.NewLogger()
	reconstructedTrie, err := session.ReconstructSMTFromMinedRelaysLog(walPath, treeStore, logger)
	require.NoError(t, err, "Should be able to reconstruct SMT from WAL with large payloads")

	reconstructedRoot := reconstructedTrie.Root()
	require.Equal(t, []byte(originalRoot), []byte(reconstructedRoot), "Large payloads should reconstruct correctly")
}

// TestSessionTree_WAL_AutoFlushOnSizeThreshold verifies WAL automatically flushes when buffer exceeds size threshold
func TestSessionTree_WAL_AutoFlushOnSizeThreshold(t *testing.T) {
	sessionTree, walPath, cleanup := createTestSessionTree(t)
	defer cleanup()

	// Add relays with payloads totaling >10MB to trigger auto-flush
	// maxBufferedMinedRelaysBytesBeforeFlush = 10 * 1024 * 1024 = 10485760 bytes
	// Add 11 relays of 1MB each to exceed the threshold
	numLargeRelays := 11
	largePayloadSize := 1024 * 1024 // 1MB

	for i := range numLargeRelays {
		relayHash := randomBytes(t, 32)
		relayPayload := randomBytes(t, largePayloadSize)
		err := sessionTree.Update(relayHash, relayPayload, uint64(i+1)*1000)
		require.NoError(t, err)
	}

	// Give a moment for the auto-flush to complete (it's triggered asynchronously)
	time.Sleep(100 * time.Millisecond)

	// Verify WAL file was created and flushed WITHOUT calling Close()
	walFileInfo, err := os.Stat(walPath)
	require.NoError(t, err)
	require.Greater(t, walFileInfo.Size(), int64(10*1024*1024), "WAL should auto-flush when buffer exceeds threshold")

	require.NoError(t, sessionTree.Close())

	// Verify auto-flushed content is correct by reconstructing

	originalRoot := sessionTree.GetSMSTRoot()

	treeStore := simplemap.NewSimpleMap()
	logger := polyzero.NewLogger()
	reconstructedTrie, err := session.ReconstructSMTFromMinedRelaysLog(walPath, treeStore, logger)

	require.NoError(t, err, "Should be able to reconstruct SMT from auto-flushed WAL")

	reconstructedRoot := reconstructedTrie.Root()
	require.Equal(t, []byte(originalRoot), []byte(reconstructedRoot), "Auto-flushed WAL should reconstruct correctly")
}

// TestSessionTree_WAL_PeriodicFlush verifies WAL flushes periodically every 10 seconds
func TestSessionTree_WAL_PeriodicFlush(t *testing.T) {
	sessionTree, walPath, cleanup := createTestSessionTree(t)
	defer cleanup()

	// Add a relay
	relayHash := randomBytes(t, 32)
	relayPayload := randomBytes(t, 50)
	err := sessionTree.Update(relayHash, relayPayload, 100)
	require.NoError(t, err)

	// Wait longer than the periodic flush interval (10s)
	time.Sleep(11 * time.Second)

	// Verify WAL file exists and has content due to periodic flush
	walFileInfo, err := os.Stat(walPath)
	require.NoError(t, err, "WAL file should exist before")
	require.Greater(t, walFileInfo.Size(), int64(0), "WAL file should contain data after Update()")
}

// Helper functions

// createTestSessionTree creates a new session tree for testing and returns
// the tree, WAL path, and a cleanup function
func createTestSessionTree(t *testing.T) (relayer.SessionTree, string, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	logger := polyzero.NewLogger()

	sessionHeader := &sessiontypes.SessionHeader{
		SessionId:               "test_session",
		ApplicationAddress:      sample.AccAddressBech32(),
		ServiceId:               "test_service",
		SessionStartBlockHeight: 1,
		SessionEndBlockHeight:   100,
	}
	supplierAddr := sample.AccAddressBech32()

	sessionTree, err := session.NewSessionTree(logger, sessionHeader, supplierAddr, tmpDir)
	require.NoError(t, err)

	walPath := filepath.Join(tmpDir, minedRelaysWALDirectoryPath, supplierAddr, sessionHeader.SessionId)

	cleanup := func() {
		_ = sessionTree.Delete()
	}

	return sessionTree, walPath, cleanup
}

// randomBytes generates random bytes of the specified length
func randomBytes(t *testing.T, length int) []byte {
	t.Helper()
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	require.NoError(t, err)
	return bytes
}
