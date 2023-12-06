package types

import "github.com/cometbft/cometbft/crypto"

// GetSignableBytes returns the signable bytes for the relay request
// this involves setting the signature to nil and marshaling the message
// then hashing it to guarantee that the signable bytes are always of a
// constant and expected length.
// A value receiver is used to avoid overwriting any pre-existing signature
func (req RelayRequest) GetSignableBytes() ([]byte, error) {
	// set signature to nil
	req.Meta.Signature = nil

	// return the marshaled message
	requestBz, err := req.Marshal()
	if err != nil {
		return nil, err
	}

	// return the marshaled request hash to guarantee that the signable bytes are
	// always of a constant and expected length
	hash := crypto.Sha256(requestBz)
	return hash, nil
}

// GetSignableBytes returns the signable bytes for the relay response
// this involves setting the signature to nil and marshaling the message
// then hashing it to guarantee that the signable bytes are always of a
// constant and expected length.
// A value receiver is used to avoid overwriting any pre-existing signature
func (res RelayResponse) GetSignableBytes() ([]byte, error) {
	// set signature to nil
	res.Meta.SupplierSignature = nil

	responseBz, err := res.Marshal()
	if err != nil {
		return nil, err
	}

	// return the marshaled response hash to guarantee that the signable bytes are
	// always of a constant and expected length
	hash := crypto.Sha256(responseBz)
	return hash, nil
}

func (res *RelayResponse) ValidateBasic() error {
	// TODO_FUTURE: if a client gets a response with an invalid/incomplete
	// SessionHeader, consider sending an on-chain challenge, lowering their
	// QoS, or other future work.

	return nil
}
