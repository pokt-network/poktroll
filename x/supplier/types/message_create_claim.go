package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedhelpers "github.com/pokt-network/poktroll/x/shared/helpers"
)

const TypeMsgCreateClaim = "create_claim"

var _ sdk.Msg = (*MsgCreateClaim)(nil)

func NewMsgCreateClaim(supplierAddress string, sessionHeader *sessiontypes.SessionHeader, rootHash []byte) *MsgCreateClaim {
	return &MsgCreateClaim{
		SupplierAddress: supplierAddress,
		SessionHeader:   sessionHeader,
		RootHash:        rootHash,
	}
}

func (msg *MsgCreateClaim) Route() string {
	return RouterKey
}

func (msg *MsgCreateClaim) Type() string {
	return TypeMsgCreateClaim
}

func (msg *MsgCreateClaim) GetSigners() []sdk.AccAddress {
	supplierAddress, err := sdk.AccAddressFromBech32(msg.SupplierAddress)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{supplierAddress}
}

func (msg *MsgCreateClaim) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgCreateClaim) ValidateBasic() error {
	// Validate the supplier address
	_, err := sdk.AccAddressFromBech32(msg.SupplierAddress)
	if err != nil {
		return sdkerrors.Wrapf(ErrSupplierInvalidAddress, "invalid supplierAddress address (%s)", err)
	}

	// Validate the session header
	sessionHeader := msg.SessionHeader
	if sessionHeader.SessionStartBlockHeight < 1 {
		return sdkerrors.Wrapf(ErrSupplierInvalidSessionStartHeight, "invalid session start block height (%d)", sessionHeader.SessionStartBlockHeight)
	}
	if len(sessionHeader.SessionId) == 0 {
		return sdkerrors.Wrapf(ErrSupplierInvalidSessionId, "invalid session ID (%v)", sessionHeader.SessionId)
	}
	if !sharedhelpers.IsValidService(sessionHeader.Service) {
		return sdkerrors.Wrapf(ErrSupplierInvalidService, "invalid service (%v)", sessionHeader.Service)
	}

	// Validate the root hash
	// TODO_IMPROVE: Only checking to make sure a non-nil hash was provided for now, but we can validate the length as well.
	if len(msg.RootHash) == 0 {
		return sdkerrors.Wrapf(ErrSupplierInvalidClaimRootHash, "invalid root hash (%v)", msg.RootHash)
	}

	return nil
}
