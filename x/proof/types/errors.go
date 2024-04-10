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
)
