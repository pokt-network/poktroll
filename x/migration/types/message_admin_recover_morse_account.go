package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = (*MsgAdminRecoverMorseAccount)(nil)

func NewMsgAdminRecoverMorseAccount(authority string, shannonDestAddress string, morseSrcAddress string) *MsgAdminRecoverMorseAccount {
	return &MsgAdminRecoverMorseAccount{
		Authority:          authority,
		ShannonDestAddress: shannonDestAddress,
		MorseSrcAddress:    morseSrcAddress,
	}
}

func (msg *MsgAdminRecoverMorseAccount) ValidateBasic() error {
	// Validate authority address format
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address (%s)", err)
	}

	// Validate Shannon destination address format
	if _, err := sdk.AccAddressFromBech32(msg.ShannonDestAddress); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid shannon destination address (%s)", err)
	}

	// Morse source address should not be empty
	// (We don't validate the format here since it's a hex string that may be invalid intentionally)
	if msg.MorseSrcAddress == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "morse source address cannot be empty")
	}

	return nil
}
