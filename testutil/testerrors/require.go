package testerrors

import sdkerrors "cosmossdk.io/errors"

var (
	// ErrAsync is returned when a test assertion fails in a goroutine other than
	// the main test goroutine. This is done to avoid concurrent usage of
	// t.Fatal() which can cause the test binary to exit before cleanup is complete.
	ErrAsync  = sdkerrors.Register(codespace, 1, "required assertion failed")
	codespace = "testerrors"
)
