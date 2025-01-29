package types

import (
	"encoding/hex"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgClaimMorsePokt{}

func NewMsgClaimMorsePokt(shannonDestAddress string, morseSrcAddress string, morseSignature []byte) *MsgClaimMorsePokt {
	return &MsgClaimMorsePokt{
		ShannonDestAddress: shannonDestAddress,
		MorseSrcAddress:    morseSrcAddress,
		MorseSignature:     morseSignature,
	}
}

func (msg *MsgClaimMorsePokt) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.ShannonDestAddress)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid shannonDestAddress address (%s)", err)
	}

	morseAddrBz, err := hex.DecodeString(msg.MorseSrcAddress)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid morseSrcAddress address (%s)", err)
	}

	if len(morseAddrBz) != 20 {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid morseSrcAddress address (%s)", err)
	}

	return nil
}
