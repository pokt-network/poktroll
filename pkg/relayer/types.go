package relayer

import (
	"github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// MinedRelay is a wrapper around a relay that has been serialized and hashed.
type MinedRelay struct {
	types.Relay
	Bytes []byte
	Hash  []byte
}

type SessionProof struct {
	ProofBz       []byte
	SessionHeader *sessiontypes.SessionHeader
}

type SessionClaim struct {
	RootHash      []byte
	SessionHeader *sessiontypes.SessionHeader
}
