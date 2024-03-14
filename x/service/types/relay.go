package types

import (
	"crypto/sha256"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
)

// GetSignableBytesHash returns the hash of the signable bytes of the relay request
// Hashing the marshaled request message guarantees that the signable bytes are
// always of a constant and expected length.
func (req *RelayRequest) GetSignableBytesHash() ([32]byte, error) {
	// Save the signature and restore it after getting the signable bytes
	// Since req.Meta is a pointer, this approach is not concurrent safe,
	// if two goroutines are calling this method at the same time, the last one
	// could get the nil signature resulting form the first go routine and restore
	// nil after getting the signable bytes.
	// TODO_TECHDEBT: Consider using a deep copy of the response to avoid this issue
	// by having req.Meta as a non-pointer type in the corresponding proto file.
	signature := req.Meta.Signature
	req.Meta.Signature = nil
	requestBz, err := req.Marshal()

	// Set the signature back to its original value
	req.Meta.Signature = signature

	if err != nil {
		return [32]byte{}, err
	}

	// return the marshaled request hash to guarantee that the signable bytes are
	// always of a constant and expected length
	return sha256.Sum256(requestBz), nil
}

// TODO_TEST: Add tests for RelayRequest validation
// ValidateBasic performs basic validation of the RelayResponse Meta, SessionHeader
// and Signature fields.
func (req *RelayRequest) ValidateBasic() error {
	if req.GetMeta() == nil {
		return ErrServiceInvalidRelayRequest.Wrap("missing meta")
	}

	if err := req.GetMeta().GetSessionHeader().ValidateBasic(); err != nil {
		return ErrServiceInvalidRelayRequest.Wrapf("invalid session header: %s", err)
	}

	if len(req.GetMeta().GetSignature()) == 0 {
		return ErrServiceInvalidRelayRequest.Wrap("missing application signature")
	}

	return nil
}

// GetSignableBytesHash returns the hash of the signable bytes of the relay response
// Hashing the marshaled response message guarantees that the signable bytes are
// always of a constant and expected length.
func (res *RelayResponse) GetSignableBytesHash() ([32]byte, error) {
	// Save the signature and restore it after getting the signable bytes
	// Since res.Meta is a pointer, this approach is not concurrent safe,
	// if two goroutines are calling this method at the same time, the last one
	// could get the nil signature resulting form the first go routine and restore
	// nil after getting the signable bytes.
	// TODO_TECHDEBT: Consider using a deep copy of the response to avoid this issue
	// by having res.Meta as a non-pointer type in the corresponding proto file.
	signature := res.Meta.SupplierSignature
	res.Meta.SupplierSignature = nil
	responseBz, err := res.Marshal()

	// Set the signature back to its original value
	res.Meta.SupplierSignature = signature

	if err != nil {
		return [32]byte{}, err
	}

	// return the marshaled response hash to guarantee that the signable bytes are
	// always of a constant and expected length
	return sha256.Sum256(responseBz), nil
}

// TODO_TEST: Add tests for RelayResponse validation
// ValidateBasic performs basic validation of the RelayResponse Meta, SessionHeader
// and SupplierSignature fields.
func (res *RelayResponse) ValidateBasic() error {
	// TODO_FUTURE: if a client gets a response with an invalid/incomplete
	// SessionHeader, consider sending an on-chain challenge, lowering their
	// QoS, or other future work.

	if res.GetMeta() == nil {
		return ErrServiceInvalidRelayResponse.Wrap("missing meta")
	}

	if err := res.GetMeta().GetSessionHeader().ValidateBasic(); err != nil {
		return ErrServiceInvalidRelayResponse.Wrapf("invalid session header: %v", err)
	}

	if len(res.GetMeta().GetSupplierSignature()) == 0 {
		return ErrServiceInvalidRelayResponse.Wrap("missing supplier signature")
	}

	return nil
}

// VerifySupplierSignature ensures the signature provided by the supplier is
// valid according to their relay response.
func (res *RelayResponse) VerifySupplierSignature(supplierPubKey cryptotypes.PubKey) error {
	// Get the signable bytes hash of the response.
	signableBz, err := res.GetSignableBytesHash()
	if err != nil {
		return err
	}

	if ok := supplierPubKey.VerifySignature(signableBz[:], res.GetMeta().GetSupplierSignature()); !ok {
		return ErrServiceInvalidRelayResponse.Wrap("invalid signature")
	}

	return nil
}
