package types

import (
	"crypto/sha256"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
)

// getSignableBytes returns the bytes resulting from marshaling the relay request
// A value receiver is used to avoid overwriting any pre-existing signature
func (req RelayRequest) getSignableBytes() ([]byte, error) {
	// set signature to nil
	req.Meta.Signature = nil

	return req.Marshal()
}

// GetSignableBytesHash returns the hash of the signable bytes of the relay request
// Hashing the marshaled request message guarantees that the signable bytes are
// always of a constant and expected length.
func (req *RelayRequest) GetSignableBytesHash() ([32]byte, error) {
	// Save the signature and restore it after getting the signable bytes
	// since getSignableBytes sets the signature to nil but does not preserve
	// its original value.
	signature := req.Meta.Signature
	requestBz, err := req.getSignableBytes()

	// Set the signature back to its original value
	req.Meta.Signature = signature

	if err != nil {
		return [32]byte{}, err
	}

	// return the marshaled request hash to guarantee that the signable bytes are
	// always of a constant and expected length
	return sha256.Sum256(requestBz), nil
}

func (req *RelayRequest) ValidateBasic() error {
	if req.GetMeta() == nil {
		return ErrServiceInvalidRelayRequest.Wrap("missing meta")
	}

	if err := req.GetMeta().GetSessionHeader().ValidateBasic(); err != nil {
		return ErrServiceInvalidRelayRequest.Wrapf("invalid session header: %s", err)
	}

	if len(req.GetMeta().GetSignature()) == 0 {
		return ErrServiceInvalidRelayRequest.Wrap("missing signature")
	}

	return nil
}

// getSignableBytes returns the bytes resulting from marshaling the relay response
// A value receiver is used to avoid overwriting any pre-existing signature
func (res RelayResponse) getSignableBytes() ([]byte, error) {
	// set signature to nil
	res.Meta.SupplierSignature = nil

	return res.Marshal()
}

// GetSignableBytesHash returns the hash of the signable bytes of the relay response
// Hashing the marshaled response message guarantees that the signable bytes are
// always of a constant and expected length.
func (res *RelayResponse) GetSignableBytesHash() ([32]byte, error) {
	// Save the signature and restore it after getting the signable bytes
	// since getSignableBytes sets the signature to nil but does not preserve
	// its original value.
	signature := res.Meta.SupplierSignature
	responseBz, err := res.getSignableBytes()

	// Set the signature back to its original value
	res.Meta.SupplierSignature = signature

	if err != nil {
		return [32]byte{}, err
	}

	// return the marshaled response hash to guarantee that the signable bytes are
	// always of a constant and expected length
	return sha256.Sum256(responseBz), nil
}

func (res *RelayResponse) ValidateBasic() error {
	// TODO_FUTURE: if a client gets a response with an invalid/incomplete
	// SessionHeader, consider sending an on-chain challenge, lowering their
	// QoS, or other future work.

	if res.GetMeta() == nil {
		return ErrServiceInvalidRelayResponse.Wrap("missing meta")
	}

	if err := res.GetMeta().GetSessionHeader().ValidateBasic(); err != nil {
		return ErrServiceInvalidRelayResponse.Wrapf("invalid session header: %s", err)
	}

	if len(res.GetMeta().GetSupplierSignature()) == 0 {
		return ErrServiceInvalidRelayResponse.Wrap("missing supplier signature")
	}

	return nil
}

func (res *RelayResponse) VerifySignature(supplierPubKey cryptotypes.PubKey) error {
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
