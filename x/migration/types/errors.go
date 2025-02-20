package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/migration module sentinel errors
var (
	ErrInvalidSigner       = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrMorseAccountsImport = sdkerrors.Register(ModuleName, 1101, "unable to import morse claimable accounts")
	ErrUnauthorized        = sdkerrors.Register(ModuleName, 1102, "unauthorized")
	ErrMorseAccountClaim   = sdkerrors.Register(ModuleName, 1103, "unable to claim morse account")
)
