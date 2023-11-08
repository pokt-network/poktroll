package proxy

import (
	"github.com/cometbft/cometbft/crypto"

	"github.com/pokt-network/poktroll/x/service/types"
)

// SignRelayResponse is a shared method used by the RelayServers to sign the hash of the RelayResponse..
// It uses the keyring and keyName to sign the payload and returns the signature.
// TODO_TECHDEBT(@red-0ne): This method should be moved out of the RelayerProxy interface
// that should not be responsible for signing relay responses.
// See https://github.com/pokt-network/poktroll/issues/160 for a better design.
func (rp *relayerProxy) SignRelayResponse(relayResponse *types.RelayResponse) (*types.RelayResponse, error) {
	var responseBz []byte
	_, err := relayResponse.MarshalTo(responseBz)
	if err != nil {
		return nil, err
	}

	hash := crypto.Sha256(responseBz)
	signature, _, err := rp.keyring.Sign(rp.keyName, hash)

	relayResponse.Meta.SupplierSignature = signature

	return relayResponse, err
}
