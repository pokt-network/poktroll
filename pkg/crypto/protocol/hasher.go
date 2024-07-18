package protocol

import "crypto/sha256"

const (
	RelayHasherSize = sha256.Size
)

var (
	NewRelayHasher = sha256.New
)
