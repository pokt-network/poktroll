package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/supplier module sentinel errors
var (
	ErrSupplierInvalidStake              = sdkerrors.Register(ModuleName, 1, "invalid supplier stake")
	ErrSupplierInvalidAddress            = sdkerrors.Register(ModuleName, 2, "invalid address")
	ErrSupplierUnauthorized              = sdkerrors.Register(ModuleName, 3, "unauthorized supplier signer")
	ErrSupplierNotFound                  = sdkerrors.Register(ModuleName, 4, "supplier not found")
	ErrSupplierInvalidServiceConfig      = sdkerrors.Register(ModuleName, 5, "invalid service config")
	ErrSupplierInvalidSessionStartHeight = sdkerrors.Register(ModuleName, 6, "invalid session start height")
	ErrSupplierInvalidSessionId          = sdkerrors.Register(ModuleName, 7, "invalid session ID")
	ErrSupplierInvalidService            = sdkerrors.Register(ModuleName, 8, "invalid service in supplier")
	ErrSupplierInvalidClaimRootHash      = sdkerrors.Register(ModuleName, 9, "invalid root hash")
	ErrSupplierInvalidSessionEndHeight   = sdkerrors.Register(ModuleName, 10, "invalid session ending height")
	ErrSupplierInvalidQueryRequest       = sdkerrors.Register(ModuleName, 11, "invalid query request")
	ErrSupplierClaimNotFound             = sdkerrors.Register(ModuleName, 12, "claim not found")
	ErrSupplierProofNotFound             = sdkerrors.Register(ModuleName, 13, "proof not found")
	ErrSupplierInvalidClosestMerkleProof = sdkerrors.Register(ModuleName, 14, "invalid closest merkle proof")
)
