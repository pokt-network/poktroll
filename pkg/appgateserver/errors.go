package appgateserver

import sdkerrors "cosmossdk.io/errors"

var (
	codespace                           = "appgateserver"
	ErrAppGateNoRelayEndpoints          = sdkerrors.Register(codespace, 1, "no relay endpoints found")
	ErrAppGateMissingAppAddress         = sdkerrors.Register(codespace, 2, "missing application address")
	ErrAppGateMissingSigningInformation = sdkerrors.Register(codespace, 3, "missing app client signing information")
	ErrAppGateMissingListeningEndpoint  = sdkerrors.Register(codespace, 4, "missing app client listening endpoint")
	ErrAppGateHandleRelay               = sdkerrors.Register(codespace, 5, "internal error handling relay request")
	ErrAppGateUpstreamError             = sdkerrors.Register(codespace, 6, "upstream error")
)
