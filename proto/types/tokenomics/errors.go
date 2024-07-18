package tokenomics

// DONTCOVER

import sdkerrors "cosmossdk.io/errors"

const codespace = "tokenomics"

// x/tokenomics module sentinel errors
var (
	ErrTokenomicsInvalidSigner                = sdkerrors.Register(codespace, 1100, "the provided authority address does not match the on-chain governance address")
	ErrTokenomicsAddressInvalid               = sdkerrors.Register(codespace, 1101, "the provided authority address is not a valid bech32 address")
	ErrTokenomicsClaimNil                     = sdkerrors.Register(codespace, 1102, "provided claim is nil")
	ErrTokenomicsSessionHeaderNil             = sdkerrors.Register(codespace, 1103, "provided claim's session header is nil")
	ErrTokenomicsSessionHeaderInvalid         = sdkerrors.Register(codespace, 1104, "provided claim's session header is invalid")
	ErrTokenomicsSupplierModuleMintFailed     = sdkerrors.Register(codespace, 1105, "failed to mint uPOKT to supplier module account")
	ErrTokenomicsSupplierRewardFailed         = sdkerrors.Register(codespace, 1106, "failed to send uPOKT from supplier module account to supplier")
	ErrTokenomicsSupplierAddressInvalid       = sdkerrors.Register(codespace, 1107, "the supplier address in the claim is not a valid bech32 address")
	ErrTokenomicsApplicationNotFound          = sdkerrors.Register(codespace, 1108, "application not found")
	ErrTokenomicsApplicationModuleBurn        = sdkerrors.Register(codespace, 1109, "failed to burn uPOKT from application module account")
	ErrTokenomicsApplicationAddressInvalid    = sdkerrors.Register(codespace, 1112, "the application address in the claim is not a valid bech32 address")
	ErrTokenomicsParamsInvalid                = sdkerrors.Register(codespace, 1113, "provided params are invalid")
	ErrTokenomicsRootHashInvalid              = sdkerrors.Register(codespace, 1114, "the root hash in the claim is invalid")
	ErrTokenomicsApplicationNewStakeInvalid   = sdkerrors.Register(codespace, 1115, "application stake cannot be reduced to a -ve amount")
	ErrTokenomicsParamNameInvalid             = sdkerrors.Register(codespace, 1116, "the provided param name is invalid")
	ErrTokenomicsParamInvalid                 = sdkerrors.Register(codespace, 1117, "the provided param is invalid")
	ErrTokenomicsUnmarshalInvalid             = sdkerrors.Register(codespace, 1118, "failed to unmarshal the provided bytes")
	ErrTokenomicsDuplicateIndex               = sdkerrors.Register(codespace, 1119, "cannot have a duplicate index")
	ErrTokenomicsMissingRelayMiningDifficulty = sdkerrors.Register(codespace, 1120, "missing relay mining difficulty")
	ErrTokenomicsApplicationOverserviced      = sdkerrors.Register(codespace, 1121, "application was overserviced")
)
