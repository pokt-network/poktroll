package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// ProofPrimaryKeyPrefix is the prefix to retrieve the entire Proof object (the primary store)
	// TODO_TECHDEBT: consider renaming to ProofSessionIDPrefix.
	ProofPrimaryKeyPrefix = "Proof/primary_key/"

	// ProofSupplierOperatorAddressPrefix is the key to retrieve a Proof's Primary Key from the Address index
	ProofSupplierOperatorAddressPrefix = "Proof/address/"

	// ProofSessionEndHeightPrefix is the key to retrieve a Proof's Primary Key from the Height index
	ProofSessionEndHeightPrefix = "Proof/height/"
)

// ProofPrimaryKey returns the primary store key used to retrieve a Proof by creating
// a composite key of the sessionId and supplierOperatorAddr.
func ProofPrimaryKey(sessionId, supplierOperatorAddr string) []byte {
	// We are guaranteed uniqueness of the primary key if it's a composite of the (sessionId, supplierOperatorAddr).
	// because every supplier can only have one Proof per session.
	return KeyComposite([]byte(sessionId), []byte(supplierOperatorAddr))
}

// ProofSupplierOperatorAddressKey returns the key used to iterate through Proofs given a supplier operator Address.
func ProofSupplierOperatorAddressKey(supplierOperatorAddr string, primaryKey []byte) []byte {
	return KeyComposite([]byte(supplierOperatorAddr), primaryKey)
}

// ProofSupplierEndSessionHeightKey returns the key used to iterate through Proofs given a session end height.
func ProofSupplierEndSessionHeightKey(sessionEndHeight int64, primaryKey []byte) []byte {
	heightBz := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBz, uint64(sessionEndHeight))

	return KeyComposite(heightBz, primaryKey)
}

// TODO_TECHDEBT(@olshanks): add helpers for composing query-side key prefixes & document key/value prefix design.
