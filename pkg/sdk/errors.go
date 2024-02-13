package sdk

import (
	sdkerrors "cosmossdk.io/errors"
)

// TODO_TECHDEBT: Do a source code wise find-replace using regex pattern match
// of `sdkerrors\.Wrapf\(([a-zA-Z]+), ` with `$1.Wrapf(`

// TODO_TECHDEBT: Do a source code wise find-replace using regex pattern match
// of `sdkerrors\.Wrapf\(([a-zA-Z]+), ` with `$1.Wrapf(`
var (
	codespace                           = "poktrollsdk"
	ErrSDKHandleRelay                   = sdkerrors.Register(codespace, 1, "internal error handling relay request")
	ErrSDKInvalidRelayResponseSignature = sdkerrors.Register(codespace, 2, "invalid relay response signature")
	ErrSDKEmptyRelayResponseSignature   = sdkerrors.Register(codespace, 3, "empty relay response signature")
	ErrSDKVerifyResponseSignature       = sdkerrors.Register(codespace, 4, "error verifying relay response signature")
	ErrSDKGetApplications               = sdkerrors.Register(codespace, 5, "error querying on-chain applications")
	ErrSDKEmptySupplierPubKey           = sdkerrors.Register(codespace, 6, "empty supplier public key")
)
