package types

import (
	"encoding/hex"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const morseAddressByteLength = 20

var _ sdk.Msg = (*MsgClaimMorsePokt)(nil)

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

	if len(morseAddrBz) != morseAddressByteLength {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid morseSrcAddress address (%s)", err)
	}

	return nil
}
