package relay_authenticator

import (
	"github.com/pokt-network/poktroll/pkg/signer"
	"github.com/pokt-network/poktroll/x/service/types"
)

// Background on RelayResponse.Payload && RelayResponse.PayloadHash:
//
// The following steps are important to understand as of the changes introduced in v0.1.25.
//
// In v0.1.25, the RelayResponse.PayloadHash field was introduced to:
// - Reduce the size of onchain proofs
// - While, maintaining the integrity of the response
//
// The steps can be summarized as follows:
// 1. The RelayMiner gets the response payload from the backend server
// 2. The RelayMiner computes and updates the RelayResponse.PayloadHash with the response payload's hash
// 3. The RelayMiner replies to the Application/Gateway with the response which:
// 	- Must contain the RelayResponse.Payload for QoS verification
// 	- Must contain the RelayResponse.PayloadHash for signature verification
// 4. The RelayMiner forwards the response inside its internal communication channels
// 5. The RelayMiner nullifies the RelayResponse.Payload prior to marshalling and tree insertion
// 6. The RelayMiner checks reward eligibility (if a minable relay) using the marshalled bytes of the RelayResponse which
//    - Contains RelayResponse.PayloadHash
//    - Does not contain RelayResponse.Payload

// SignRelayResponse signs the hash of a RelayResponse given the supplier operator address.
// It uses the keyring and keyName to sign the payload and returns the signature.
func (ra *relayAuthenticator) SignRelayResponse(relayResponse *types.RelayResponse, supplierOperatorAddr string) error {
	if err := relayResponse.Meta.SessionHeader.ValidateBasic(); err != nil {
		return ErrRelayAuthenticatorInvalidRelayResponse.Wrapf("invalid session header: %v", err)
	}

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
