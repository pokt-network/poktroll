package proxy

import (
	"github.com/cometbft/cometbft/crypto"

	"pocket/x/service/types"
)

// SignRelayResponse is a shared method used by the RelayServers to sign a RelayResponse.Payload.
// It uses the keyring and keyName to sign the payload and returns the signature.
func (rp *relayerProxy) SignRelayResponse(relayResponse *types.RelayResponse) ([]byte, error) {
	var payloadBz []byte
	_, err := relayResponse.Payload.MarshalTo(payloadBz)
	if err != nil {
		return nil, err
	}

	hash := crypto.Sha256(payloadBz)
	signature, _, err := rp.keyring.Sign(rp.keyName, hash)

	return signature, err
}
