package session_test

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"testing"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore/pebble"
	"github.com/stretchr/testify/require"
)

const (
	maxLeafs      = 10000
	numIterations = 100
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

			trie := smt.NewSparseMerkleSumTrie(kvStore, protocol.NewTrieHasher(), smt.WithValueHasher(nil))

			// Insert numLeaf random leaves.
			for i := 0; i < numLeafs; i++ {
				key := make([]byte, 32)
				_, err := rand.Read(key)
				require.NoError(t, err)
				// Insert an empty value since this does not get affected by the compaction,
				// this is also to not favor proof compression that compresses the value too.
				trie.Update(key, []byte{}, 1)
			}

			// Generate a random path.
			var path = make([]byte, 32)
			_, err = rand.Read(path)
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

		compactionRatio := float32(cumulativeCompactProofSize) / float32(cumulativeProofSize)
		compressionRatio := float32(cumulativeGzippedProofSize) / float32(cumulativeProofSize)

		// Gzip compression is more efficient than compaction.
		require.Greater(t, compactionRatio, compressionRatio)

		t.Logf(
			"numLeaf=%d: compactionRatio: %f, compressionRatio: %f",
			numLeafs, compactionRatio, compressionRatio,
		)
	}
}
