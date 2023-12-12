package types

import "crypto/sha256"

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
	requestBz, err := req.getSignableBytes()
	if err != nil {
		return [32]byte{}, err
	}

	// return the marshaled request hash to guarantee that the signable bytes are
	// always of a constant and expected length
	return sha256.Sum256(requestBz), nil
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
	responseBz, err := res.getSignableBytes()
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

	return nil
}
