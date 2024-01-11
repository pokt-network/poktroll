package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const TypeMsgUpdateParams = "update_params"

var _ sdk.Msg = (*MsgUpdateParams)(nil)

func NewMsgUpdateParams(authority string) *MsgUpdateParams {
	return &MsgUpdateParams{
		Authority: authority,
	}
}

func (msg *MsgUpdateParams) Route() string {
	return RouterKey
}

func (msg *MsgUpdateParams) Type() string {
	return TypeMsgUpdateParams
}

func (msg *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	authority, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{authority}
}

func (msg *MsgUpdateParams) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgUpdateParams) ValidateBasic() error {
	// Validate the address
	_, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return sdkerrors.Wrapf(ErrTokenomicsAuthorityInvalidAddress, "invalid authority address %s; (%v)", msg.Authority, err)
	}

	return nil
}

// https://github.com/cosmos/cosmos-sdk/blob/main/docs/architecture/adr-030-authz-module.md
// What if we let authority grant MsgUpdateParams permission to a particular address that'll be m of n?
