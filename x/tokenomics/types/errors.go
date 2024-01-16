package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/tokenomics module sentinel errors
var (
	ErrTokenomicsAuthorityInvalidAddress    = sdkerrors.Register(ModuleName, 1, "provided authority address is invalid")
	ErrTokenomicsAuthorityAddressIncorrect  = sdkerrors.Register(ModuleName, 2, "provided authority address is incorrect")
	ErrTokenomicsClaimNil                   = sdkerrors.Register(ModuleName, 3, "provided claim is nil")
	ErrTokenomicsSessionHeaderNil           = sdkerrors.Register(ModuleName, 4, "provided claim's session header is nil")
	ErrTokenomicsSupplierModuleMintFailed   = sdkerrors.Register(ModuleName, 5, "failed to mint uPOKT to supplier module account")
	ErrTokenomicsSupplierRewardFailed       = sdkerrors.Register(ModuleName, 6, "failed to send uPOKT from supplier module account to supplier")
	ErrTokenomicsApplicationModuleBurn      = sdkerrors.Register(ModuleName, 7, "failed to burn uPOKT from application module account")
	ErrTokenomicsApplicationModuleFeeFailed = sdkerrors.Register(ModuleName, 8, "failed to send uPOKT from application module account to application")
	ErrTokenomicsParamsInvalid              = sdkerrors.Register(ModuleName, 9, "provided params are invalid")
)
