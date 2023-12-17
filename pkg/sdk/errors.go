package sdk

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	codespace                           = "poktrollsdk"
	ErrSDKHandleRelay                   = sdkerrors.Register(codespace, 1, "internal error handling relay request")
	ErrSDKInvalidRelayResponseSignature = sdkerrors.Register(codespace, 2, "invalid relay response signature")
	ErrSDKEmptyRelayResponseSignature   = sdkerrors.Register(codespace, 3, "empty relay response signature")
	ErrSDKVerifyResponseSignature       = sdkerrors.Register(codespace, 4, "error verifying relay response signature")
)
