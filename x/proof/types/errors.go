package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/proof module sentinel errors
var (
	ErrProofInvalidSigner             = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrProofInvalidAddress            = sdkerrors.Register(ModuleName, 1101, "invalid address")
	ErrProofNotFound                  = sdkerrors.Register(ModuleName, 1102, "supplier not found")
	ErrProofInvalidService            = sdkerrors.Register(ModuleName, 1103, "invalid service in supplier")
	ErrProofInvalidClaimRootHash      = sdkerrors.Register(ModuleName, 1104, "invalid root hash")
	ErrProofInvalidQueryRequest       = sdkerrors.Register(ModuleName, 1105, "invalid query request")
	ErrProofClaimNotFound             = sdkerrors.Register(ModuleName, 1106, "claim not found")
	ErrProofProofNotFound             = sdkerrors.Register(ModuleName, 1107, "proof not found")
	ErrProofInvalidProof              = sdkerrors.Register(ModuleName, 1108, "invalid proof")
	ErrProofInvalidRelay              = sdkerrors.Register(ModuleName, 1109, "invalid relay")
	ErrProofInvalidRelayRequest       = sdkerrors.Register(ModuleName, 1110, "invalid relay request")
	ErrProofInvalidRelayResponse      = sdkerrors.Register(ModuleName, 1111, "invalid relay response")
	ErrProofNotSecp256k1Curve         = sdkerrors.Register(ModuleName, 1112, "not secp256k1 curve")
	ErrProofApplicationNotFound       = sdkerrors.Register(ModuleName, 1113, "application not found")
	ErrProofPubKeyNotFound            = sdkerrors.Register(ModuleName, 1114, "public key not found")
	ErrProofInvalidSessionHeader      = sdkerrors.Register(ModuleName, 1115, "invalid session header")
	ErrProofInvalidSessionId          = sdkerrors.Register(ModuleName, 1116, "invalid session ID")
	ErrProofInvalidSessionEndHeight   = sdkerrors.Register(ModuleName, 1117, "invalid session end height")
	ErrProofInvalidSessionStartHeight = sdkerrors.Register(ModuleName, 1118, "invalid session start height")
	ErrProofParamNameInvalid          = sdkerrors.Register(ModuleName, 1119, "the provided param name is invalid")
	ErrProofParamInvalid              = sdkerrors.Register(ModuleName, 1120, "the provided param is invalid")
	ErrProofClaimOutsideOfWindow      = sdkerrors.Register(ModuleName, 1121, "claim attempted outside of the session's claim window")
	ErrProofProofOutsideOfWindow      = sdkerrors.Register(ModuleName, 1122, "proof attempted outside of the session's proof window")
	ErrProofSupplierMismatch          = sdkerrors.Register(ModuleName, 1123, "supplier operator address does not match the claim or proof")
	ErrProofAccNotFound               = sdkerrors.Register(ModuleName, 1124, "account not found")
	ErrProofServiceNotFound           = sdkerrors.Register(ModuleName, 1125, "service not found")
	ErrProofComputeUnitsMismatch      = sdkerrors.Register(ModuleName, 1126, "mismatch: claim compute units != number of relays * service compute units per relay")
	ErrProofNotEnoughFunds            = sdkerrors.Register(ModuleName, 1127, "not enough funds to submit proof")
	ErrProofFailedToDeductFee         = sdkerrors.Register(ModuleName, 1128, "failed to deduct proof submission fee")
)
