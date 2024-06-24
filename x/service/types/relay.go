package types

import (
	"crypto/sha256"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"

	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// GetHashFromBytes returns the hash of the relay (full, request or response) bytes.
// It is used as helper in the case that the relay is already marshaled and
// centralizes the hasher used.
func GetHashFromBytes(relayBz []byte) [32]byte {
	return sha256.Sum256(relayBz)
}

// GetHash returns the hash of the relay, which contains both the signed
// relay request and the relay response. It is used as the key for insertion
// into the SMT.
func (relay *Relay) GetHash() ([32]byte, error) {
	relayBz, err := relay.Marshal()
	if err != nil {
		return [32]byte{}, err
	}

	return GetHashFromBytes(relayBz), nil
}

// GetSignableBytesHash returns the hash of the signable bytes of the relay request
// Hashing the marshaled request message guarantees that the signable bytes are
// always of a constant and expected length.
func (req RelayRequest) GetSignableBytesHash() ([32]byte, error) {
	// req and req.Meta are not pointers, so we can set the signature to nil
	// in order to generate the signable bytes hash without the need restore it.
	req.Meta.Signature = nil
	requestBz, err := req.Marshal()
	if err != nil {
		return [32]byte{}, err
	}

	// return the marshaled request hash to guarantee that the signable bytes
	// are always of a constant and expected length
	return GetHashFromBytes(requestBz), nil
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

	if meta.GetSupplierAddress() == "" {
		return ErrServiceInvalidRelayRequest.Wrap("relay metadata missing supplier address")
	}

	return nil
}

// GetSignableBytesHash returns the hash of the signable bytes of the relay response
// Hashing the marshaled response message guarantees that the signable bytes are
// always of a constant and expected length.
func (res RelayResponse) GetSignableBytesHash() ([32]byte, error) {
	// res and res.Meta are not pointers, so we can set the signature to nil
	// in order to generate the signable bytes hash without the need restore it.
	res.Meta.SupplierSignature = nil
	responseBz, err := res.Marshal()
	if err != nil {
		return [32]byte{}, err
	}

	// return the marshaled response hash to guarantee that the signable bytes
	// are always of a constant and expected length
	return GetHashFromBytes(responseBz), nil
}

// ValidateBasic performs basic validation of the RelayResponse Meta, SessionHeader
// and SupplierSignature fields.
// TODO_TEST: Add tests for RelayResponse validation
func (res *RelayResponse) ValidateBasic() error {
	// TODO: if a client gets a response with an invalid/incomplete
	// SessionHeader, consider sending an on-chain challenge, lowering their
	// QoS, or other future work.

	meta := res.GetMeta()

	if meta.GetSessionHeader() == nil {
		return ErrServiceInvalidRelayResponse.Wrap("missing meta")
	}

	if err := meta.GetSessionHeader().ValidateBasic(); err != nil {
		return ErrServiceInvalidRelayResponse.Wrapf("invalid session header: %v", err)
	}

	if len(meta.GetSupplierSignature()) == 0 {
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
	if ok := supplierPubKey.VerifySignature(signableBz[:], res.GetMeta().SupplierSignature); !ok {
		return ErrServiceInvalidRelayResponse.Wrap("invalid signature")
	}

	return nil
}

// nullifyForObservability generates an empty RelayRequest that has the same
// service and payload as the source RelayRequest if they are not nil.
// It is meant to be used when replying with an error but no valid RelayRequest is available.
func (sourceRelayRequest *RelayRequest) NullifyForObservability() *RelayRequest {
	emptyRelayRequest := &RelayRequest{
		Meta: RelayRequestMetadata{
			SessionHeader: &sessiontypes.SessionHeader{
				Service: &sharedtypes.Service{
					Id: "",
				},
			},
		},
		Payload: []byte{},
	}

	if sourceRelayRequest == nil {
		return emptyRelayRequest
	}

	if sourceRelayRequest.Payload != nil {
		emptyRelayRequest.Payload = sourceRelayRequest.Payload
	}

	if sourceRelayRequest.Meta.SessionHeader != nil {
		if sourceRelayRequest.Meta.SessionHeader.Service != nil {
			emptyRelayRequest.Meta.SessionHeader.Service = sourceRelayRequest.Meta.SessionHeader.Service
		}
	}

	return emptyRelayRequest
}
