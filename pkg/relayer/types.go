package relayer

import (
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// MinedRelay is a wrapper around a relay that has been serialized and hashed.
type MinedRelay struct {
	types.Relay
	Bytes []byte
	Hash  []byte
}

// SessionProof is a struct that contains a proof and its corresponding session header.
// It is used to submit a proof batches to the chain.
type SessionProof struct {
	ProofBz         []byte
	SupplierAddress cosmostypes.AccAddress
	SessionHeader   *sessiontypes.SessionHeader
}

// SessionClaim is a struct that contains a root hash and its corresponding session header.
// It is used to submit a claim batches to the chain.
type SessionClaim struct {
	RootHash        []byte
	SupplierAddress cosmostypes.AccAddress
	SessionHeader   *sessiontypes.SessionHeader
}
