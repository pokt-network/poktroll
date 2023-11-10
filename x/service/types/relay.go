package types

// GetSignableBytes returns the signable bytes for the relay request
// this involves setting the signature to nil and marshaling the message.
// A value receiver is used to avoid overwriting any pre-existing signature
func (req RelayRequest) GetSignableBytes() ([]byte, error) {
	// set signature to nil
	sig := req.Meta.Signature
	req.Meta.Signature = nil

	bz, err := req.Marshal()
	if err == nil {
		req.Meta.Signature = sig
	}

	// return the marshaled message
	return bz, nil
}

// GetSignableBytes returns the signable bytes for the relay response
// this involves setting the signature to nil and marshaling the message.
// A value receiver is used to avoid overwriting any pre-existing signature
func (res RelayResponse) GetSignableBytes() ([]byte, error) {
	// set signature to nil
	sig := res.Meta.SupplierSignature
	res.Meta.SupplierSignature = nil

	bz, err := res.Marshal()
	if err == nil {
		res.Meta.SupplierSignature = sig
	}

	// return the marshaled message
	return bz, err
}
