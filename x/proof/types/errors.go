package types

// DONTCOVER

import sdkerrors "cosmossdk.io/errors"

// x/proof module sentinel errors
var (
	ErrProofInvalidSigner             = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrProofInvalidAddress            = sdkerrors.Register(ModuleName, 1101, "invalid address")
	ErrProofNotFound                  = sdkerrors.Register(ModuleName, 1102, "supplier not found")
	ErrProofInvalidSessionStartHeight = sdkerrors.Register(ModuleName, 1103, "invalid session start height")
	ErrProofInvalidSessionId          = sdkerrors.Register(ModuleName, 1104, "invalid session ID")
	ErrProofInvalidService            = sdkerrors.Register(ModuleName, 1105, "invalid service in supplier")
	ErrProofInvalidClaimRootHash      = sdkerrors.Register(ModuleName, 1106, "invalid root hash")
	ErrProofInvalidSessionEndHeight   = sdkerrors.Register(ModuleName, 1107, "invalid session ending height")
	ErrProofInvalidQueryRequest       = sdkerrors.Register(ModuleName, 1108, "invalid query request")
	ErrProofClaimNotFound             = sdkerrors.Register(ModuleName, 1109, "claim not found")
	ErrProofProofNotFound             = sdkerrors.Register(ModuleName, 1110, "proof not found")
	ErrProofInvalidProof              = sdkerrors.Register(ModuleName, 1111, "invalid proof")
	ErrProofInvalidRelay              = sdkerrors.Register(ModuleName, 1112, "invalid relay")
	ErrProofInvalidRelayRequest       = sdkerrors.Register(ModuleName, 1113, "invalid relay request")
	ErrProofInvalidRelayResponse      = sdkerrors.Register(ModuleName, 1114, "invalid relay response")
	ErrProofNotSecp256k1Curve         = sdkerrors.Register(ModuleName, 1115, "not secp256k1 curve")
	ErrProofApplicationNotFound       = sdkerrors.Register(ModuleName, 1116, "application not found")
	ErrProofPubKeyNotFound            = sdkerrors.Register(ModuleName, 1117, "public key not found")
)
