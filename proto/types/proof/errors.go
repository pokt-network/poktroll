package proof

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

const codespace = "proof"

// x/proof module sentinel errors
var (
	ErrProofInvalidSigner             = sdkerrors.Register(codespace, 1100, "expected gov account as only signer for proposal message")
	ErrProofInvalidAddress            = sdkerrors.Register(codespace, 1101, "invalid address")
	ErrProofNotFound                  = sdkerrors.Register(codespace, 1102, "supplier not found")
	ErrProofInvalidService            = sdkerrors.Register(codespace, 1103, "invalid service in supplier")
	ErrProofInvalidClaimRootHash      = sdkerrors.Register(codespace, 1104, "invalid root hash")
	ErrProofInvalidQueryRequest       = sdkerrors.Register(codespace, 1105, "invalid query request")
	ErrProofClaimNotFound             = sdkerrors.Register(codespace, 1106, "claim not found")
	ErrProofProofNotFound             = sdkerrors.Register(codespace, 1107, "proof not found")
	ErrProofInvalidProof              = sdkerrors.Register(codespace, 1108, "invalid proof")
	ErrProofInvalidRelay              = sdkerrors.Register(codespace, 1109, "invalid relay")
	ErrProofInvalidRelayRequest       = sdkerrors.Register(codespace, 1110, "invalid relay request")
	ErrProofInvalidRelayResponse      = sdkerrors.Register(codespace, 1111, "invalid relay response")
	ErrProofNotSecp256k1Curve         = sdkerrors.Register(codespace, 1112, "not secp256k1 curve")
	ErrProofApplicationNotFound       = sdkerrors.Register(codespace, 1113, "application not found")
	ErrProofPubKeyNotFound            = sdkerrors.Register(codespace, 1114, "public key not found")
	ErrProofInvalidSessionHeader      = sdkerrors.Register(codespace, 1115, "invalid session header")
	ErrProofInvalidSessionId          = sdkerrors.Register(codespace, 1116, "invalid session ID")
	ErrProofInvalidSessionEndHeight   = sdkerrors.Register(codespace, 1117, "invalid session end height")
	ErrProofInvalidSessionStartHeight = sdkerrors.Register(codespace, 1118, "invalid session start height")
	ErrProofParamNameInvalid          = sdkerrors.Register(codespace, 1119, "the provided param name is invalid")
	ErrProofParamInvalid              = sdkerrors.Register(codespace, 1120, "the provided param is invalid")
	ErrProofClaimOutsideOfWindow      = sdkerrors.Register(codespace, 1121, "claim attempted outside of the session's claim window")
	ErrProofProofOutsideOfWindow      = sdkerrors.Register(codespace, 1122, "proof attempted outside of the session's proof window")
)
