package keeper

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore/simplemap"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
)

// A closest proof for the seed path (the pre-weighted proof format) must be rejected
// unless it happens to prove the leaf the seed selects by relay count.
func TestVerifyWeightedProof_RejectsClosestProofOfOtherLeaf(t *testing.T) {
	trie := smt.NewSparseMerkleSumTrie(simplemap.NewSimpleMap(), protocol.NewTrieHasher(), protocol.SMTValueHasher())
	for i := 0; i < 64; i++ {
		relayBz := []byte(fmt.Sprintf("relay-%d", i))
		relayHash := sha256.Sum256(relayBz)
		require.NoError(t, trie.Update(relayHash[:], relayBz, 1))
	}
	require.NoError(t, trie.Commit())
	root := trie.Root()

	numRejected := 0
	for i := 0; i < 32; i++ {
		seed := protocol.GetPathForProof([]byte(fmt.Sprintf("block-%d", i)), "session")

		weightedProof, err := trie.ProveWeighted(seed)
		require.NoError(t, err)
		require.NoError(t, verifyWeightedProof(weightedProof, root, seed))

		closestProof, err := trie.ProveClosest(seed)
		require.NoError(t, err)
		require.NoError(t, ensureWeightedProofForm(weightedProof))
		require.Error(t, ensureWeightedProofForm(closestProof))
		if verifyWeightedProof(closestProof, root, seed) != nil {
			numRejected++
		}
	}
	// A closest proof is only accepted when it is the weighted proof itself; with a
	// random seed its path differs from the leaf path, so it is rejected.
	require.Equal(t, 32, numRejected)
}

// A relay is accepted only at the path of its own hash.
func TestEnsureRelayAtCanonicalPath(t *testing.T) {
	relayBz := []byte("relay")
	relayHash := protocol.GetRelayHashFromBytes(relayBz)

	trie := smt.NewSparseMerkleSumTrie(simplemap.NewSimpleMap(), protocol.NewTrieHasher(), protocol.SMTValueHasher())
	require.NoError(t, trie.Update(relayHash[:], relayBz, 1))
	for i := 0; i < 16; i++ {
		otherKey := sha256.Sum256([]byte(fmt.Sprintf("other-%d", i)))
		require.NoError(t, trie.Update(otherKey[:], relayBz, 1))
	}
	require.NoError(t, trie.Commit())
	root := trie.Root()

	numCanonical, numOther := 0, 0
	for i := 0; i < 64; i++ {
		seed := protocol.GetPathForProof([]byte(fmt.Sprintf("block-%d", i)), "session")
		proof, err := trie.ProveWeighted(seed)
		require.NoError(t, err)
		require.NoError(t, verifyWeightedProof(proof, root, seed))

		canonicalPath := sha256.Sum256(relayHash[:])
		if bytes.Equal(proof.ClosestPath, canonicalPath[:]) {
			require.NoError(t, ensureRelayAtCanonicalPath(proof, relayBz))
			numCanonical++
		} else {
			require.Error(t, ensureRelayAtCanonicalPath(proof, relayBz))
			numOther++
		}
	}
	require.Positive(t, numCanonical)
	require.Positive(t, numOther)
}
