package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/proof module sentinel errors
var (
	ErrProofInvalidAddress = sdkerrors.Register(ModuleName, 2, "invalid address")
	//ErrProofUnauthorized = sdkerrors.Register(ModuleName, 3, "unauthorized supplier signer")
	ErrProofNotFound = sdkerrors.Register(ModuleName, 4, "supplier not found")
	//ErrProofInvalidServiceConfig      = sdkerrors.Register(ModuleName, 5, "invalid service config")
	ErrProofInvalidSessionStartHeight = sdkerrors.Register(ModuleName, 6, "invalid session start height")
	ErrProofInvalidSessionId          = sdkerrors.Register(ModuleName, 7, "invalid session ID")
	ErrProofInvalidService            = sdkerrors.Register(ModuleName, 8, "invalid service in supplier")
	ErrProofInvalidClaimRootHash      = sdkerrors.Register(ModuleName, 9, "invalid root hash")
	ErrProofInvalidSessionEndHeight   = sdkerrors.Register(ModuleName, 10, "invalid session ending height")
	ErrProofInvalidQueryRequest       = sdkerrors.Register(ModuleName, 11, "invalid query request")
	ErrProofClaimNotFound             = sdkerrors.Register(ModuleName, 12, "claim not found")
	ErrProofProofNotFound             = sdkerrors.Register(ModuleName, 13, "proof not found")
	ErrProofInvalidProof              = sdkerrors.Register(ModuleName, 14, "invalid proof")
	//ErrProofInvalidClosestMerkleProof = sdkerrors.Register(ModuleName, 15, "invalid closest merkle proof")
	ErrInvalidSigner = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
)
