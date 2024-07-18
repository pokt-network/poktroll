package protocol

// GetHashFromBytes returns the hash of the relay (full, request or response) bytes.
// It is used as helper in the case that the relay is already marshaled and
// centralizes the hasher used.
func GetHashFromBytes(relayBz []byte) (hash [RelayHasherSize]byte) {
	hasher := NewRelayHasher()
	// NB: Intentionally ignoring the error, following sha256.Sum256 implementation.
	_, _ = hasher.Write(relayBz)
	hashBz := hasher.Sum(nil)
	copy(hash[:], hashBz)

	return hash
}
