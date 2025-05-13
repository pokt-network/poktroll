package crypto

import (
	"github.com/tendermint/go-amino"
)

var cdc = amino.NewCodec()

func init() {
	RegisterAmino()
}

// RegisterAmino registers all go-crypto related types in the given (amino) codec.
func RegisterAmino() {
	cdc.RegisterInterface((*PublicKey)(nil), nil)
	cdc.RegisterConcrete(Ed25519PublicKey{}, "crypto/ed25519_public_key", nil)
	cdc.RegisterInterface((*MultiSig)(nil), nil)
	cdc.RegisterInterface((*PublicKeyMultiSig)(nil), nil)
	cdc.RegisterConcrete(PublicKeyMultiSignature{}, "crypto/public_key_multi_signature", nil)
	cdc.RegisterConcrete(MultiSignature{}, "crypto/multi_signature", nil)
}
