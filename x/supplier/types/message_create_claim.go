package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	sessiontypes "pocket/x/session/types"
)

const TypeMsgCreateClaim = "create_claim"

var _ sdk.Msg = &MsgCreateClaim{}

func NewMsgCreateClaim(supplierAddress string, sessionHeader *sessiontypes.SessionHeader, rootHash string) *MsgCreateClaim {
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
	_, err := sdk.AccAddressFromBech32(msg.SupplierAddress)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid supplierAddress address (%s)", err)
	}
	return nil
}
