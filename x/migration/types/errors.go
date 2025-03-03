package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/migration module sentinel errors
var (
	ErrInvalidSigner         = sdkerrors.Register(ModuleName, 1100, "expected x/gov module account as the only signer for migration state import messages")
	ErrMorseAccountsImport   = sdkerrors.Register(ModuleName, 1101, "unable to import morse claimable accounts")
	ErrMorseAccountClaim     = sdkerrors.Register(ModuleName, 1102, "unable to claim morse account")
	ErrMorseApplicationClaim = sdkerrors.Register(ModuleName, 1104, "unable to claim morse account as a staked application")
	ErrMorseSupplierClaim    = sdkerrors.Register(ModuleName, 1105, "unable to claim morse account as a staked supplier")
	ErrMorseGatewayClaim     = sdkerrors.Register(ModuleName, 1106, "unable to claim morse account as a staked gateway")
)
