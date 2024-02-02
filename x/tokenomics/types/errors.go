package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/tokenomics module sentinel errors
var (
	ErrTokenomicsAuthorityAddressInvalid       = sdkerrors.Register(ModuleName, 1, "the provided authority address is not a valid bech32 address")
	ErrTokenomicsAuthorityAddressMismatch      = sdkerrors.Register(ModuleName, 2, "the provided authority address does not match the on-chain governance address")
	ErrTokenomicsClaimNil                      = sdkerrors.Register(ModuleName, 3, "provided claim is nil")
	ErrTokenomicsSessionHeaderNil              = sdkerrors.Register(ModuleName, 4, "provided claim's session header is nil")
	ErrTokenomicsSessionHeaderInvalid          = sdkerrors.Register(ModuleName, 5, "provided claim's session header is invalid")
	ErrTokenomicsSupplierModuleMintFailed      = sdkerrors.Register(ModuleName, 6, "failed to mint uPOKT to supplier module account")
	ErrTokenomicsSupplierRewardFailed          = sdkerrors.Register(ModuleName, 7, "failed to send uPOKT from supplier module account to supplier")
	ErrTokenomicsSupplierAddressInvalid        = sdkerrors.Register(ModuleName, 8, "the supplier address in the claim is not a valid bech32 address")
	ErrTokenomicsApplicationNotFound           = sdkerrors.Register(ModuleName, 9, "application not found")
	ErrTokenomicsApplicationModuleBurn         = sdkerrors.Register(ModuleName, 10, "failed to burn uPOKT from application module account")
	ErrTokenomicsApplicationModuleFeeFailed    = sdkerrors.Register(ModuleName, 11, "failed to send uPOKT from application module account to application")
	ErrTokenomicsApplicationUndelegationFailed = sdkerrors.Register(ModuleName, 12, "failed to undelegate uPOKT from the application module to the application account")
	ErrTokenomicsApplicationAddressInvalid     = sdkerrors.Register(ModuleName, 13, "the application address in the claim is not a valid bech32 address")
	ErrTokenomicsParamsInvalid                 = sdkerrors.Register(ModuleName, 14, "provided params are invalid")
	ErrTokenomicsRootHashInvalid               = sdkerrors.Register(ModuleName, 15, "the root hash in the claim is invalid")
)
