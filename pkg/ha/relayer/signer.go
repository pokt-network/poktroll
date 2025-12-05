package relayer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdktypes "github.com/pokt-network/shannon-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/pokt-network/poktroll/pkg/polylog"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// privKeySigner implements the signer interface using a private key directly.
type privKeySigner struct {
	privKey cryptotypes.PrivKey
}

func (s *privKeySigner) Sign(msg [32]byte) ([]byte, error) {
	return s.privKey.Sign(msg[:])
}

// Signer interface for signing messages.
type Signer interface {
	Sign(msg [32]byte) (signature []byte, err error)
}

// ResponseSigner handles signing of relay responses.
// It manages multiple supplier signing keys and can sign responses on behalf of any
// supplier whose key is loaded.
type ResponseSigner struct {
	logger polylog.Logger

	// signers maps supplier operator address -> signer
	signers map[string]Signer

	// operatorAddresses is the list of loaded operator addresses
	operatorAddresses []string
}

// NewResponseSigner creates a new ResponseSigner from a map of operator address to private key.
// This is the preferred constructor that works with pre-loaded keys from KeyProvider.
func NewResponseSigner(
	logger polylog.Logger,
	keys map[string]cryptotypes.PrivKey,
) (*ResponseSigner, error) {
	rs := &ResponseSigner{
		logger:            logger,
		signers:           make(map[string]Signer, len(keys)),
		operatorAddresses: make([]string, 0, len(keys)),
	}

	for operatorAddr, privKey := range keys {
		rs.signers[operatorAddr] = &privKeySigner{privKey: privKey}
		rs.operatorAddresses = append(rs.operatorAddresses, operatorAddr)

		logger.Info().
			Str("operator_address", operatorAddr).
			Msg("loaded signing key")
	}

	return rs, nil
}

// GetOperatorAddresses returns the list of operator addresses that can sign.
func (rs *ResponseSigner) GetOperatorAddresses() []string {
	return rs.operatorAddresses
}

// HasSigner returns true if a signer exists for the given operator address.
func (rs *ResponseSigner) HasSigner(operatorAddress string) bool {
	_, ok := rs.signers[operatorAddress]
	return ok
}

// SignRelayResponse signs a relay response for the given supplier operator.
// It computes the payload hash, then signs and sets the signature on the response.
func (rs *ResponseSigner) SignRelayResponse(
	relayResponse *servicetypes.RelayResponse,
	supplierOperatorAddr string,
) error {
	signer, ok := rs.signers[supplierOperatorAddr]
	if !ok {
		return fmt.Errorf("no signer for operator %s (available: %v)", supplierOperatorAddr, rs.operatorAddresses)
	}

	// Ensure payload hash is set
	if err := relayResponse.UpdatePayloadHash(); err != nil {
		return fmt.Errorf("failed to update payload hash: %w", err)
	}

	// Get signable bytes hash
	signableBz, err := relayResponse.GetSignableBytesHash()
	if err != nil {
		return fmt.Errorf("failed to get signable bytes: %w", err)
	}

	// Sign
	sig, err := signer.Sign(signableBz)
	if err != nil {
		return fmt.Errorf("failed to sign response: %w", err)
	}

	// Set signature
	relayResponse.Meta.SupplierOperatorSignature = sig

	return nil
}

// SignRelayResponseWithContext signs a relay response and returns the signature.
// This method implements the RelaySignerKeyring interface for use with RelayProcessor.
func (rs *ResponseSigner) SignRelayResponseWithContext(
	ctx context.Context,
	relayResponse *servicetypes.RelayResponse,
	supplierOperatorAddr string,
) ([]byte, error) {
	signer, ok := rs.signers[supplierOperatorAddr]
	if !ok {
		return nil, fmt.Errorf("no signer for operator %s (available: %v)", supplierOperatorAddr, rs.operatorAddresses)
	}

	// Ensure payload hash is set
	if err := relayResponse.UpdatePayloadHash(); err != nil {
		return nil, fmt.Errorf("failed to update payload hash: %w", err)
	}

	// Get signable bytes hash
	signableBz, err := relayResponse.GetSignableBytesHash()
	if err != nil {
		return nil, fmt.Errorf("failed to get signable bytes: %w", err)
	}

	// Sign
	sig, err := signer.Sign(signableBz)
	if err != nil {
		return nil, fmt.Errorf("failed to sign response: %w", err)
	}

	// Set signature on the response
	relayResponse.Meta.SupplierOperatorSignature = sig

	return sig, nil
}

// BuildAndSignRelayResponse creates a signed RelayResponse from the relay request
// and backend HTTP response.
// This is the main entry point for building responses to send back to clients.
func (rs *ResponseSigner) BuildAndSignRelayResponse(
	relayRequest *servicetypes.RelayRequest,
	backendResp *http.Response,
	maxBodySize int64,
) (*servicetypes.RelayResponse, []byte, error) {
	if relayRequest == nil || relayRequest.Meta.SessionHeader == nil {
		return nil, nil, fmt.Errorf("invalid relay request: missing session header")
	}

	supplierOperatorAddr := relayRequest.Meta.SupplierOperatorAddress
	if supplierOperatorAddr == "" {
		return nil, nil, fmt.Errorf("missing supplier operator address in relay request")
	}

	if !rs.HasSigner(supplierOperatorAddr) {
		return nil, nil, fmt.Errorf("no signer for supplier %s", supplierOperatorAddr)
	}

	// Serialize the backend HTTP response to POKT format
	_, poktResponseBz, err := rs.serializeHTTPResponse(backendResp, maxBodySize)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to serialize HTTP response: %w", err)
	}

	// Build the RelayResponse
	relayResponse := &servicetypes.RelayResponse{
		Meta: servicetypes.RelayResponseMetadata{
			SessionHeader: relayRequest.Meta.SessionHeader,
			// SupplierOperatorSignature will be set by SignRelayResponse
		},
		Payload: poktResponseBz,
		// PayloadHash will be set by SignRelayResponse
	}

	// Sign the response
	if err := rs.SignRelayResponse(relayResponse, supplierOperatorAddr); err != nil {
		return nil, nil, fmt.Errorf("failed to sign relay response: %w", err)
	}

	// Marshal the signed response
	relayResponseBz, err := relayResponse.Marshal()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal relay response: %w", err)
	}

	return relayResponse, relayResponseBz, nil
}

// BuildAndSignRelayResponseFromBody creates a signed RelayResponse from the relay request
// and raw response body (for streaming responses where we've already read the body).
func (rs *ResponseSigner) BuildAndSignRelayResponseFromBody(
	relayRequest *servicetypes.RelayRequest,
	respBody []byte,
	respHeaders http.Header,
	respStatus int,
) (*servicetypes.RelayResponse, []byte, error) {
	if relayRequest == nil || relayRequest.Meta.SessionHeader == nil {
		return nil, nil, fmt.Errorf("invalid relay request: missing session header")
	}

	supplierOperatorAddr := relayRequest.Meta.SupplierOperatorAddress
	if supplierOperatorAddr == "" {
		return nil, nil, fmt.Errorf("missing supplier operator address in relay request")
	}

	if !rs.HasSigner(supplierOperatorAddr) {
		return nil, nil, fmt.Errorf("no signer for supplier %s", supplierOperatorAddr)
	}

	// Convert headers to POKT format
	headers := make(map[string]*sdktypes.Header, len(respHeaders))
	for key := range respHeaders {
		values := respHeaders.Values(key)
		headers[key] = &sdktypes.Header{
			Key:    key,
			Values: values,
		}
	}

	// Create POKT HTTP response
	poktResponse := &sdktypes.POKTHTTPResponse{
		StatusCode: uint32(respStatus),
		Header:     headers,
		BodyBz:     respBody,
	}

	// Serialize with deterministic marshaling
	marshalOpts := proto.MarshalOptions{Deterministic: true}
	poktResponseBz, err := marshalOpts.Marshal(poktResponse)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal POKT HTTP response: %w", err)
	}

	// Build the RelayResponse
	relayResponse := &servicetypes.RelayResponse{
		Meta: servicetypes.RelayResponseMetadata{
			SessionHeader: relayRequest.Meta.SessionHeader,
		},
		Payload: poktResponseBz,
	}

	// Sign the response
	if err := rs.SignRelayResponse(relayResponse, supplierOperatorAddr); err != nil {
		return nil, nil, fmt.Errorf("failed to sign relay response: %w", err)
	}

	// Marshal the signed response
	relayResponseBz, err := relayResponse.Marshal()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal relay response: %w", err)
	}

	return relayResponse, relayResponseBz, nil
}

// BuildErrorRelayResponse creates a signed RelayResponse for an error condition.
// This allows the relayer to return proper signed errors to clients.
func (rs *ResponseSigner) BuildErrorRelayResponse(
	sessionHeader *sessiontypes.SessionHeader,
	supplierOperatorAddr string,
	errorCode uint32,
	errorMessage string,
) (*servicetypes.RelayResponse, []byte, error) {
	if sessionHeader == nil {
		return nil, nil, fmt.Errorf("session header is required for error response")
	}

	if !rs.HasSigner(supplierOperatorAddr) {
		return nil, nil, fmt.Errorf("no signer for supplier %s", supplierOperatorAddr)
	}

	// Create error POKT HTTP response
	poktResponse := &sdktypes.POKTHTTPResponse{
		StatusCode: errorCode,
		Header:     make(map[string]*sdktypes.Header),
		BodyBz:     []byte(fmt.Sprintf(`{"error":"%s"}`, errorMessage)),
	}
	poktResponse.Header["Content-Type"] = &sdktypes.Header{
		Key:    "Content-Type",
		Values: []string{"application/json"},
	}

	marshalOpts := proto.MarshalOptions{Deterministic: true}
	poktResponseBz, err := marshalOpts.Marshal(poktResponse)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal error response: %w", err)
	}

	// Build the RelayResponse with error
	relayResponse := &servicetypes.RelayResponse{
		Meta: servicetypes.RelayResponseMetadata{
			SessionHeader: sessionHeader,
		},
		Payload: poktResponseBz,
		RelayMinerError: &servicetypes.RelayMinerError{
			Code:        errorCode,
			Description: errorMessage,
		},
	}

	// Sign the response
	if err := rs.SignRelayResponse(relayResponse, supplierOperatorAddr); err != nil {
		return nil, nil, fmt.Errorf("failed to sign error response: %w", err)
	}

	// Marshal
	relayResponseBz, err := relayResponse.Marshal()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal error relay response: %w", err)
	}

	return relayResponse, relayResponseBz, nil
}

// serializeHTTPResponse converts an http.Response into a protobuf-serialized byte slice.
// This is a local implementation similar to pkg/relayer/proxy/http_utils.go.
func (rs *ResponseSigner) serializeHTTPResponse(
	response *http.Response,
	maxBodySize int64,
) (*sdktypes.POKTHTTPResponse, []byte, error) {
	defer response.Body.Close()

	// Read body with size limit
	limitedReader := io.LimitReader(response.Body, maxBodySize+1)
	var buf bytes.Buffer
	bytesRead, err := buf.ReadFrom(limitedReader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}
	if bytesRead > maxBodySize {
		return nil, nil, fmt.Errorf("response body too large: %d > %d", bytesRead, maxBodySize)
	}

	// Convert headers
	headers := make(map[string]*sdktypes.Header, len(response.Header))
	for key := range response.Header {
		values := response.Header.Values(key)
		headers[key] = &sdktypes.Header{
			Key:    key,
			Values: values,
		}
	}

	// Create POKT HTTP response
	poktResponse := &sdktypes.POKTHTTPResponse{
		StatusCode: uint32(response.StatusCode),
		Header:     headers,
		BodyBz:     buf.Bytes(),
	}

	// Deterministic marshal
	marshalOpts := proto.MarshalOptions{Deterministic: true}
	poktResponseBz, err := marshalOpts.Marshal(poktResponse)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal POKT HTTP response: %w", err)
	}

	return poktResponse, poktResponseBz, nil
}

// ResponseSignerAdapter wraps ResponseSigner to implement the RelaySignerKeyring interface.
// This adapter provides the correct method signature expected by RelayProcessor.
type ResponseSignerAdapter struct {
	signer *ResponseSigner
}

// NewResponseSignerAdapter creates a new adapter that wraps ResponseSigner.
func NewResponseSignerAdapter(signer *ResponseSigner) *ResponseSignerAdapter {
	return &ResponseSignerAdapter{signer: signer}
}

// SignRelayResponse implements the RelaySignerKeyring interface.
func (a *ResponseSignerAdapter) SignRelayResponse(
	ctx context.Context,
	response *servicetypes.RelayResponse,
	supplierOperatorAddr string,
) ([]byte, error) {
	return a.signer.SignRelayResponseWithContext(ctx, response, supplierOperatorAddr)
}

// Verify interface compliance.
var _ RelaySignerKeyring = (*ResponseSignerAdapter)(nil)
