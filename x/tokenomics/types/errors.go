package types

// DONTCOVER

import sdkerrors "cosmossdk.io/errors"

// x/tokenomics module sentinel errors
var (
	ErrTokenomicsInvalidSigner                = sdkerrors.Register(ModuleName, 1100, "the provided authority address does not match the on-chain governance address")
	ErrTokenomicsAddressInvalid               = sdkerrors.Register(ModuleName, 1101, "the provided authority address is not a valid bech32 address")
	ErrTokenomicsClaimNil                     = sdkerrors.Register(ModuleName, 1102, "provided claim is nil")
	ErrTokenomicsSessionHeaderNil             = sdkerrors.Register(ModuleName, 1103, "provided claim's session header is nil")
	ErrTokenomicsSessionHeaderInvalid         = sdkerrors.Register(ModuleName, 1104, "provided claim's session header is invalid")
	ErrTokenomicsSupplierModuleSendFailed     = sdkerrors.Register(ModuleName, 1105, "failed to send uPOKT to supplier module account")
	ErrTokenomicsSupplierAddressInvalid       = sdkerrors.Register(ModuleName, 1106, "the supplier address in the claim is not a valid bech32 address")
	ErrTokenomicsSupplierNotFound             = sdkerrors.Register(ModuleName, 1107, "supplier not found")
	ErrTokenomicsApplicationNotFound          = sdkerrors.Register(ModuleName, 1108, "application not found")
	ErrTokenomicsApplicationModuleBurn        = sdkerrors.Register(ModuleName, 1109, "failed to burn uPOKT from application module account")
	ErrTokenomicsApplicationAddressInvalid    = sdkerrors.Register(ModuleName, 1110, "the application address in the claim is not a valid bech32 address")
	ErrTokenomicsParamsInvalid                = sdkerrors.Register(ModuleName, 1111, "provided params are invalid")
	ErrTokenomicsRootHashInvalid              = sdkerrors.Register(ModuleName, 1112, "the root hash in the claim is invalid")
	ErrTokenomicsApplicationNewStakeInvalid   = sdkerrors.Register(ModuleName, 1113, "application stake cannot be reduced to a -ve amount")
	ErrTokenomicsParamNameInvalid             = sdkerrors.Register(ModuleName, 1114, "the provided param name is invalid")
	ErrTokenomicsParamInvalid                 = sdkerrors.Register(ModuleName, 1115, "the provided param is invalid")
	ErrTokenomicsUnmarshalInvalid             = sdkerrors.Register(ModuleName, 1116, "failed to unmarshal the provided bytes")
	ErrTokenomicsDuplicateIndex               = sdkerrors.Register(ModuleName, 1117, "cannot have a duplicate index")
	ErrTokenomicsMissingRelayMiningDifficulty = sdkerrors.Register(ModuleName, 1118, "missing relay mining difficulty")
	ErrTokenomicsApplicationOverserviced      = sdkerrors.Register(ModuleName, 1119, "application was overserviced")
	ErrTokenomicsServiceNotFound              = sdkerrors.Register(ModuleName, 1120, "service not found")
	ErrTokenomicsModuleMintFailed             = sdkerrors.Register(ModuleName, 1121, "failed to mint uPOKT to tokenomics module account")
	ErrTokenomicsSendingMindRewards           = sdkerrors.Register(ModuleName, 1122, "failed to send minted rewards")
	ErrTokenomicsSupplierModuleMintFailed     = sdkerrors.Register(ModuleName, 1123, "failed to mint uPOKT to supplier module account")
)
