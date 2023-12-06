package signer

import (
	"fmt"

	ringtypes "github.com/athanorlabs/go-dleq/types"
	ring "github.com/noot/ring-go"
)

var _ Signer = (*RingSigner)(nil)

// RingSigner is a signer implementation that uses a ring to sign messages, for
// verification the ring signature must be verified and confirmed to be using
// the expected ring.
type RingSigner struct {
	ring    *ring.Ring
	privKey ringtypes.Scalar
}

// NewRingSigner creates a new RingSigner instance with the ring and private key provided
func NewRingSigner(ring *ring.Ring, privKey ringtypes.Scalar) *RingSigner {
	return &RingSigner{ring: ring, privKey: privKey}
}

// Sign uses the ring and private key to sign the message provided and returns the
// serialized ring signature that can be deserialized and verified by the verifier
func (r *RingSigner) Sign(msg []byte) ([]byte, error) {
	if len(msg) != 32 {
		return nil, fmt.Errorf("message must be 32 bytes long, got %d", len(msg))
	}
	var msg32 [32]byte
	copy(msg32[:], msg)
	ringSig, err := r.ring.Sign(msg32, r.privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign message [%v]: %w", msg, err)
	}
	return ringSig.Serialize()
}
