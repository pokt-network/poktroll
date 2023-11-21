package testrelayer

import (
	"hash"
	"testing"

	"github.com/stretchr/testify/require"
)

// TODO_TECHDEBT(@h5law): Retrieve the relay hasher mechanism from the `smt` repo.
func HashBytes(t *testing.T, newHasher func() hash.Hash, relayBz []byte) []byte {
	hasher := newHasher()
	_, err := hasher.Write(relayBz)
	require.NoError(t, err)
	return hasher.Sum(nil)
}
