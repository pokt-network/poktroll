package relay_authenticator

import (
	"github.com/pokt-network/poktroll/pkg/signer"
	"github.com/pokt-network/poktroll/x/service/types"
)

// SignRelayResponse signs the hash of a RelayResponse given the supplier operator address.
// It uses the keyring and keyName to sign the payload and returns the signature.
func (ra *relayAuthenticator) SignRelayResponse(relayResponse *types.RelayResponse, supplierOperatorAddr string) error {
	// create a simple signer for the request
	operatorKeyName, ok := ra.operatorAddressToSigningKeyNameMap[supplierOperatorAddr]
	if !ok {
		return ErrRelayAuthenticatorUndefinedSigningKeyNames.Wrapf("unable to resolve the signing key name for %s", supplierOperatorAddr)
	}
	signer := signer.NewSimpleSigner(ra.keyring, operatorKeyName)

	// extract and hash the relay response's signable bytes
	signableBz, err := relayResponse.GetSignableBytesHash()
	if err != nil {
		return ErrRelayAuthenticatorInvalidRelayResponse.Wrapf("error getting signable bytes: %v", err)
	}

	// sign the relay response
	responseSig, err := signer.Sign(signableBz)
	if err != nil {
		return ErrRelayAuthenticatorInvalidRelayResponse.Wrapf("error signing relay response: %v", err)
	}

	// set the relay response's signature
	relayResponse.Meta.SupplierOperatorSignature = responseSig
	return nil
}
