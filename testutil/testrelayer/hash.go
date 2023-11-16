package testrelayer

import (
	"hash"
	"testing"

	"github.com/stretchr/testify/require"
)

func HashBytes(t *testing.T, newHasher func() hash.Hash, relayBz []byte) []byte {
	hasher := newHasher()
	_, err := hasher.Write(relayBz)
	require.NoError(t, err)
	return hasher.Sum(nil)
}
