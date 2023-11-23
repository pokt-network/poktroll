package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// ProofKeyPrefix is the prefix to retrieve all Proof
	ProofKeyPrefix = "Proof/value/"
)

// ProofKey returns the store key to retrieve a Proof from the index fields
// TODO_UPNEXT(@Olshansk): Implement a similar indexing strategy for Proofs as we do for Claims
func ProofKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
