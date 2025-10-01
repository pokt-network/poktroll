package relay_authenticator

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	codespace                                           = "relay_authenticator"
	ErrRelayAuthenticatorInvalidSession                 = sdkerrors.Register(codespace, 1, "invalid session in relayer request")
	ErrRelayAuthenticatorInvalidSessionSupplier         = sdkerrors.Register(codespace, 2, "supplier does not belong to session")
	ErrRelayAuthenticatorUndefinedSigner                = sdkerrors.Register(codespace, 3, "undefined supplier signer")
	ErrRelayAuthenticatorInvalidRelayRequest            = sdkerrors.Register(codespace, 4, "invalid relay request")
	ErrRelayAuthenticatorInvalidRelayResponse           = sdkerrors.Register(codespace, 5, "invalid relay response")
	ErrRelayAuthenticatorMissingSupplierOperatorAddress = sdkerrors.Register(codespace, 6, "supplier operator address is missing")
)
