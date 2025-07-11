package types

import sdkerrors "cosmossdk.io/errors"

// x/tokenomics module sentinel errors
var (
	// Authority errors
	ErrTokenomicsInvalidAuthoritySigner  = sdkerrors.Register(ModuleName, 1100, "the provided authority address does not match the onchain governance address")
	ErrTokenomicsAuthorityAddressInvalid = sdkerrors.Register(ModuleName, 1101, "the provided authority address is not a valid bech32 address")

	// Claim errors
	ErrTokenomicsClaimSessionHeaderNil     = sdkerrors.Register(ModuleName, 1102, "provided claim's session header is nil")
	ErrTokenomicsClaimSessionHeaderInvalid = sdkerrors.Register(ModuleName, 1103, "provided claim's session header is invalid")
	ErrTokenomicsClaimRootHashInvalid      = sdkerrors.Register(ModuleName, 1109, "the root hash in the claim is invalid")

	// Supplier errors
	ErrTokenomicsSupplierOperatorAddressInvalid = sdkerrors.Register(ModuleName, 1104, "the supplier operator address in the claim is not a valid bech32 address")
	ErrTokenomicsSupplierNotFound               = sdkerrors.Register(ModuleName, 1105, "supplier not found")

	// Application errors
	ErrTokenomicsApplicationNotFound        = sdkerrors.Register(ModuleName, 1106, "application not found")
	ErrTokenomicsApplicationAddressInvalid  = sdkerrors.Register(ModuleName, 1107, "the application address in the claim is not a valid bech32 address")
	ErrTokenomicsApplicationNewStakeInvalid = sdkerrors.Register(ModuleName, 1110, "application stake cannot be reduced to a -ve amount")

	// Params errors
	ErrTokenomicsParamsInvalid    = sdkerrors.Register(ModuleName, 1108, "provided params are invalid")
	ErrTokenomicsParamNameInvalid = sdkerrors.Register(ModuleName, 1111, "the provided param name is invalid")
	ErrTokenomicsParamInvalid     = sdkerrors.Register(ModuleName, 1112, "the provided param is invalid")

	ErrTokenomicsUnmarshalInvalid    = sdkerrors.Register(ModuleName, 1113, "failed to unmarshal the provided bytes")
	ErrTokenomicsEmittingEventFailed = sdkerrors.Register(ModuleName, 1114, "failed to emit event")

	// Service errors
	ErrTokenomicsServiceNotFound = sdkerrors.Register(ModuleName, 1115, "service not found")

	// Settlement errors
	ErrTokenomicsConstraint         = sdkerrors.Register(ModuleName, 1116, "constraint violation")
	ErrTokenomicsSettlementInternal = sdkerrors.Register(ModuleName, 1117, "internal claim settlement error")
	ErrTokenomicsTLMInternal        = sdkerrors.Register(ModuleName, 1118, "internal token logic module error")
	ErrTokenomicsProcessingTLM      = sdkerrors.Register(ModuleName, 1119, "failed to process token logic module")
	ErrTokenomicsCoinIsZero         = sdkerrors.Register(ModuleName, 1120, "coin amount cannot be zero")
	ErrTokenomicsSettlementMint     = sdkerrors.Register(ModuleName, 1121, "failed to mint uPOKT while executing settlement state transitions")
	ErrTokenomicsSettlementBurn     = sdkerrors.Register(ModuleName, 1122, "failed to burn uPOKT while executing settlement state transitions")
	ErrTokenomicsSettlementTransfer = sdkerrors.Register(ModuleName, 1123, "failed to send coins while executing settlement state transitions")
)
