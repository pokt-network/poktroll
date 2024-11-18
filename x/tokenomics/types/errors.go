package types

// DONTCOVER

import sdkerrors "cosmossdk.io/errors"

// x/tokenomics module sentinel errors
var (
	ErrTokenomicsInvalidSigner                  = sdkerrors.Register(ModuleName, 1100, "the provided authority address does not match the on-chain governance address")
	ErrTokenomicsAddressInvalid                 = sdkerrors.Register(ModuleName, 1101, "the provided authority address is not a valid bech32 address")
	ErrTokenomicsSessionHeaderNil               = sdkerrors.Register(ModuleName, 1102, "provided claim's session header is nil")
	ErrTokenomicsSessionHeaderInvalid           = sdkerrors.Register(ModuleName, 1103, "provided claim's session header is invalid")
	ErrTokenomicsSupplierOperatorAddressInvalid = sdkerrors.Register(ModuleName, 1104, "the supplier operator address in the claim is not a valid bech32 address")
	ErrTokenomicsSupplierNotFound               = sdkerrors.Register(ModuleName, 1105, "supplier not found")
	ErrTokenomicsApplicationNotFound            = sdkerrors.Register(ModuleName, 1106, "application not found")
	ErrTokenomicsApplicationAddressInvalid      = sdkerrors.Register(ModuleName, 1107, "the application address in the claim is not a valid bech32 address")
	ErrTokenomicsParamsInvalid                  = sdkerrors.Register(ModuleName, 1108, "provided params are invalid")
	ErrTokenomicsRootHashInvalid                = sdkerrors.Register(ModuleName, 1109, "the root hash in the claim is invalid")
	ErrTokenomicsApplicationNewStakeInvalid     = sdkerrors.Register(ModuleName, 1110, "application stake cannot be reduced to a -ve amount")
	ErrTokenomicsParamNameInvalid               = sdkerrors.Register(ModuleName, 1111, "the provided param name is invalid")
	ErrTokenomicsParamInvalid                   = sdkerrors.Register(ModuleName, 1112, "the provided param is invalid")
	ErrTokenomicsUnmarshalInvalid               = sdkerrors.Register(ModuleName, 1113, "failed to unmarshal the provided bytes")
	ErrTokenomicsEmittingEventFailed            = sdkerrors.Register(ModuleName, 1114, "failed to emit event")
	ErrTokenomicsServiceNotFound                = sdkerrors.Register(ModuleName, 1115, "service not found")
	ErrTokenomicsConstraint                     = sdkerrors.Register(ModuleName, 1116, "constraint violation")
	ErrTokenomicsTLMInternal                    = sdkerrors.Register(ModuleName, 1117, "internal token logic module error")
	ErrTokenomicsProcessingTLM                  = sdkerrors.Register(ModuleName, 1118, "failed to process token logic module")
	ErrTokenomicsCoinIsZero                     = sdkerrors.Register(ModuleName, 1119, "coin amount cannot be zero")
	ErrTokenomicsSettlementModuleMint           = sdkerrors.Register(ModuleName, 1120, "failed to mint uPOKT while executing settlement state transitions")
	ErrTokenomicsSettlementModuleBurn           = sdkerrors.Register(ModuleName, 1121, "failed to burn uPOKT while executing settlement state transitions")
	ErrTokenomicsSettlementTransfer             = sdkerrors.Register(ModuleName, 1122, "failed to send coins while executing settlement state transitions")
)
