package types

// GetSignableBytes returns the signable bytes for the relay request
// this involves setting the signature to nil and marshaling the message.
// A value receiver is used to avoid overwriting any pre-existing signature
func (req RelayRequest) GetSignableBytes() ([]byte, error) {
	// set signature to nil
	req.Meta.Signature = nil

	// return the marshaled message
	return req.Marshal()
}

// GetSignableBytes returns the signable bytes for the relay response
// this involves setting the signature to nil and marshaling the message.
// A value receiver is used to avoid overwriting any pre-existing signature
func (res RelayResponse) GetSignableBytes() ([]byte, error) {
	// set signature to nil
	res.Meta.SupplierSignature = nil

	// return the marshaled message
	return res.Marshal()
}
