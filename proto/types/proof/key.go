package proof

import "bytes"

// KeyDelimiter is the delimiter for composite keys.
var KeyDelimiter = []byte("/")

// ProofPrimaryKey returns the primary store key used to retrieve a Proof by creating a composite key of the sessionId and supplierAddr.
func ProofPrimaryKey(sessionId, supplierAddr string) []byte {
	// We are guaranteed uniqueness of the primary key if it's a composite of the (sessionId, supplierAddr).
	// because every supplier can only have one Proof per session.
	return KeyComposite([]byte(sessionId), []byte(supplierAddr))
}

// ClaimPrimaryKey returns the primary store key used to retrieve a Claim by creating a composite key of the sessionId and supplierAddr.
func ClaimPrimaryKey(sessionId, supplierAddr string) []byte {
	// We are guaranteed uniqueness of the primary key if it's a composite of the (sessionId, supplierAddr)
	// because every supplier can only have one claim per session.
	return KeyComposite([]byte(sessionId), []byte(supplierAddr))
}

// KeyComposite combines the given keys into a single key for use with KVStore.
func KeyComposite(keys ...[]byte) []byte {
	return bytes.Join(keys, KeyDelimiter)
}
