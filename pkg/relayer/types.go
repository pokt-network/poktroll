package relayer

import (
	"github.com/pokt-network/pocket/x/service/types"
)

// MinedRelay is a wrapper around a relay that has been serialized and hashed.
type MinedRelay struct {
	types.Relay
	Bytes []byte
	Hash  []byte
}
