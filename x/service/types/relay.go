package types

import (
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
)

// GetHash returns the hash of the relay, which contains both the signed
// relay request and the relay response. It is used as the key for insertion
// into the SMT.
func (relay *Relay) GetHash() ([protocol.RelayHasherSize]byte, error) {
	relayBz, err := relay.Marshal()
	if err != nil {
		return [protocol.RelayHasherSize]byte{}, err
	}

	return protocol.GetRelayHashFromBytes(relayBz), nil
}

// GetSignableBytesHash returns the hash of the signable bytes of the relay request
// Hashing the marshaled request message guarantees that the signable bytes are
// always of a constant and expected length.
func (req RelayRequest) GetSignableBytesHash() ([protocol.RelayHasherSize]byte, error) {
	// req and req.Meta are not pointers, so we can set the signature to nil
	// in order to generate the signable bytes hash without the need restore it.
	req.Meta.Signature = nil
	requestBz, err := req.Marshal()
	if err != nil {
		return [protocol.RelayHasherSize]byte{}, err
	}

	// return the marshaled request hash to guarantee that the signable bytes
	// are always of a constant and expected length
	return protocol.GetRelayHashFromBytes(requestBz), nil
}

// ValidateBasic performs basic validation of the RelayResponse Meta, SessionHeader
// and Signature fields.
// TODO_TEST: Add tests for RelayRequest validation
func (req *RelayRequest) ValidateBasic() error {
	meta := req.GetMeta()

	if meta.GetSessionHeader() == nil {
		return ErrServiceInvalidRelayRequest.Wrap("missing session header")
	}

	if err := meta.GetSessionHeader().ValidateBasic(); err != nil {
		return ErrServiceInvalidRelayRequest.Wrapf("invalid session header: %s", err)
	}

	if len(meta.GetSignature()) == 0 {
		return ErrServiceInvalidRelayRequest.Wrap("missing application signature")
	}

	if meta.GetSupplierOperatorAddress() == "" {
		return ErrServiceInvalidRelayRequest.Wrap("relay metadata missing supplier operator address")
	}

	return nil
}

// GetSignableBytesHash returns the hash of the signable bytes of the relay response
// Hashing the marshaled response message guarantees that the signable bytes are
// always of a constant and expected length.
func (res RelayResponse) GetSignableBytesHash() ([protocol.RelayHasherSize]byte, error) {
	// res and res.Meta are not pointers, so we can set the signature to nil
	// in order to generate the signable bytes hash without the need restore it.
	res.Meta.SupplierOperatorSignature = nil

	// TODO_TECHDEBT: REMOVE THIS BACKWARDS COMPATIBILITY CHECK ONCE ALL ACTORS ARE UPGRADED.
	// See #1604 for the original implementation and delete after `v0.1.26` release is live
	//
	// Set the response payload to nil to reduce the size of SMST & onchain proofs.
	//
	// DEV_NOTE: This MUST be done in order to support onchain response signature verification,
	// without including the entire response payload in the SMST/proof.
	//
	// DEV_NOTE: Backward compatibility implementation for signature verification:
	// During network upgrades, different components (Chain, Gateway, RelayMiner) may run
	// different software versions.
	// The logic below enables compatibility between all versions without requiring synchronized upgrades.
	//
	// PayloadHash only gets set as of v0.1.25.
	// Prior to v0.1.25, the payload was never nill and PayloadHash did not exist.
	// By only setting the payload to nil if PayloadHash is not nil, we ensure compatibility before and after the upgrade.
	//
	// Ref for additional details: docusaurus/docs/4_develop/upgrades/10_backward_compatibility.md
	if res.PayloadHash != nil {
		res.Payload = nil
	}

	responseBz, err := res.Marshal()
	if err != nil {
		return [protocol.RelayHasherSize]byte{}, err
	}

	// return the marshaled response hash to guarantee that the signable bytes
	// are always of a constant and expected length
	return protocol.GetRelayHashFromBytes(responseBz), nil
}

// ValidateBasic performs basic validation of the RelayResponse Meta, SessionHeader
// and SupplierOperatorSignature fields.
// TODO_TEST: Add tests for RelayResponse validation
func (res *RelayResponse) ValidateBasic() error {
	// TODO_POST_MAINNET: if a client gets a response with an invalid/incomplete
	// SessionHeader, consider sending an onchain challenge, lowering their
	// QoS, or other future work.

	// TODO_TECHDEBT(red-0ne): Reenable once RelayMiners are updated to generate payload hashes.
	// if len(res.GetPayloadHash()) == 0 {
	// 	return ErrServiceInvalidRelayResponse.Wrapf("missing payload hash")
	// }

	meta := res.GetMeta()

	if meta.GetSessionHeader() == nil {
		return ErrServiceInvalidRelayResponse.Wrap("missing meta")
	}

	if err := meta.GetSessionHeader().ValidateBasic(); err != nil {
		return ErrServiceInvalidRelayResponse.Wrapf("invalid session header: %v", err)
	}

	if len(meta.GetSupplierOperatorSignature()) == 0 {
		return ErrServiceInvalidRelayResponse.Wrap("missing supplier operator signature")
	}

	return nil
}

// VerifySupplierOperatorSignature ensures the signature provided by the supplier is
// valid according to their relay response.
func (res *RelayResponse) VerifySupplierOperatorSignature(supplierOperatorPubKey cryptotypes.PubKey) error {
	// Get the signable bytes hash of the response.
	signableBz, err := res.GetSignableBytesHash()
	if err != nil {
		return err
	}
	if ok := supplierOperatorPubKey.VerifySignature(signableBz[:], res.GetMeta().SupplierOperatorSignature); !ok {
		return ErrServiceInvalidRelayResponse.Wrap("invalid signature")
	}

	return nil
}

// UpdatePayloadHash computes the hash of the response payload and set it on res (this relay response).
// This is necessary for onchain proof verification without requiring the full payload.
// If the response payload is empty, an error is returned.
func (res *RelayResponse) UpdatePayloadHash() error {
	if len(res.GetPayload()) == 0 {
		return ErrServiceInvalidRelayResponse.Wrapf("attempted to update payload hash with an empty payload")
	}

	responseHash := protocol.GetRelayHashFromBytes(res.GetPayload())
	res.PayloadHash = responseHash[:]
	return nil
}
