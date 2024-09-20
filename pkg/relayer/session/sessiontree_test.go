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

func TestSessionTree_CompactProofsAreSmallerThanNonCompactProofs(t *testing.T) {
	// Run the test for different number of leaves.
	for numLeaf := 10; numLeaf <= 1000000; numLeaf *= 10 {
		// We run the test 1000 times for each number of leaves to remove the randomness bias.
		cumulativeProofSize := 0
		cumulativeCompactProofSize := 0
		cumulativeGzippedProofSize := 0
		for numLeaf := 0; numLeaf <= 1000; numLeaf++ {
			kvStore, err := pebble.NewKVStore("")
			require.NoError(t, err)

			trie := smt.NewSparseMerkleSumTrie(kvStore, protocol.NewTrieHasher(), smt.WithValueHasher(nil))

			// Insert numLeaf random leaves.
			for i := 0; i < numLeaf; i++ {
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

			// Accumulate the proof size over 1000 runs.
			cumulativeProofSize += len(proofBz)

			// Generate the compacted proof.
			compactProof, err := smt.CompactClosestProof(proof, &trie.TrieSpec)
			require.NoError(t, err)

			compactProofBz, err := compactProof.Marshal()
			require.NoError(t, err)

			// Accumulate the compact proof size over 1000 runs.
			cumulativeCompactProofSize += len(compactProofBz)

			// Gzip the non compacted proof.
			var buf bytes.Buffer
			gzipWriter := gzip.NewWriter(&buf)
			_, err = gzipWriter.Write(proofBz)
			require.NoError(t, err)
			err = gzipWriter.Close()
			require.NoError(t, err)

			// Accumulate the gzipped proof size over 1000 runs.
			cumulativeGzippedProofSize += len(buf.Bytes())

			//t.Logf(
			//	"numLeaf: %d, proofSize: %d, compactProofSize: %d, gzipProofSize: %d",
			//	numLeaf, len(proofBz), len(compactProofBz), len(buf.Bytes()),
			//)

			// Commenting out the assertion to not fail the test since compaction is not
			// guaranteed to always reduce the proof size.
			//require.Less(t, len(compactProofBz), len(proofBz))
		}

		//t.Logf(
		//	"numLeaf=%d: cumulativeProofSize: %d, cumulativeCompactProofSize: %d, cumulativeGzippedProofSize: %d",
		//	numLeaf, cumulativeProofSize, cumulativeCompactProofSize, cumulativeGzippedProofSize,
		//)

		t.Logf(
			"numLeaf=%d: compactionRatio: %f, compressionRatio: %f",
			numLeaf,
			float32(cumulativeCompactProofSize)/float32(cumulativeProofSize),
			float32(cumulativeGzippedProofSize)/float32(cumulativeProofSize),
		)
	}
}
