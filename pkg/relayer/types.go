package relayer

import "github.com/pokt-network/poktroll/proto/types/service"

// MinedRelay is a wrapper around a relay that has been serialized and hashed.
type MinedRelay struct {
	service.Relay
	Bytes []byte
	Hash  []byte
}
