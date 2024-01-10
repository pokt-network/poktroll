package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgAddService = "add_service"

var _ sdk.Msg = (*MsgAddService)(nil)

func NewMsgAddService(address string) *MsgAddService {
	return &MsgAddService{
		Address: address,
	}
}

func (msg *MsgAddService) Route() string {
	return RouterKey
}

func (msg *MsgAddService) Type() string {
	return TypeMsgAddService
}

func (msg *MsgAddService) GetSigners() []sdk.AccAddress {
	address, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{address}
}

func (msg *MsgAddService) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgAddService) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid address address (%s)", err)
	}
	return nil
}
