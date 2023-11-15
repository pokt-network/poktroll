package appgateserver

import sdkerrors "cosmossdk.io/errors"

var (
	codespace                               = "appgateserver"
	ErrAppGateInvalidRelayResponseSignature = sdkerrors.Register(codespace, 1, "invalid relay response signature")
	ErrAppGateNoRelayEndpoints              = sdkerrors.Register(codespace, 2, "no relay endpoints found")
	ErrAppGateInvalidRequestURL             = sdkerrors.Register(codespace, 3, "invalid request URL")
	ErrAppGateMissingAppAddress             = sdkerrors.Register(codespace, 4, "missing application address")
	ErrAppGateMissingSigningInformation     = sdkerrors.Register(codespace, 5, "missing app client signing information")
	ErrAppGateMissingListeningEndpoint      = sdkerrors.Register(codespace, 6, "missing app client listening endpoint")
	ErrAppGateEmptyRelayResponseMeta        = sdkerrors.Register(codespace, 7, "empty relay response metadata")
	ErrAppGateEmptyRelayResponseSignature   = sdkerrors.Register(codespace, 8, "empty relay response signature")
	ErrAppGateHandleRelay                   = sdkerrors.Register(codespace, 9, "internal error handling relay request")
)
