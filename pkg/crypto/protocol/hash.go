package protocol

import "crypto/sha256"

// GetHashFromBytes returns the hash of the relay (full, request or response) bytes.
// It is used as helper in the case that the relay is already marshaled and
// centralizes the hasher used.
func GetHashFromBytes(relayBz []byte) [sha256.Size]byte {
	return sha256.Sum256(relayBz)
}
