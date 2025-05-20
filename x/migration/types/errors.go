package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/migration module sentinel errors
var (
	ErrInvalidSigner                = sdkerrors.Register(ModuleName, 1100, "expected x/gov module account as the only signer for migration state import messages")
	ErrMorseAccountsImport          = sdkerrors.Register(ModuleName, 1101, "unable to import morse claimable accounts")
	ErrMorseAccountClaim            = sdkerrors.Register(ModuleName, 1102, "unable to claim morse account")
	ErrMorseApplicationClaim        = sdkerrors.Register(ModuleName, 1104, "unable to claim morse account as a staked application")
	ErrMorseSupplierClaim           = sdkerrors.Register(ModuleName, 1105, "unable to claim morse account as a staked supplier")
	ErrMigrationParamInvalid        = sdkerrors.Register(ModuleName, 1106, "the provided param is invalid")
	ErrMorseSrcAddress              = sdkerrors.Register(ModuleName, 1107, "invalid Morse source account address")
	ErrMorseSignature               = sdkerrors.Register(ModuleName, 1108, "invalid morse signature")
	ErrMorseRecoverableAccountClaim = sdkerrors.Register(ModuleName, 1109, "unable to recover Morse account")
)
