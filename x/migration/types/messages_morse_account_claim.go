package types

import (
	"encoding/hex"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const morseAddressByteLength = 20

var _ sdk.Msg = &MsgCreateMorseAccountClaim{}

func NewMsgCreateMorseAccountClaim(
	shannonDestAddress string,
	morseSrcAddress string,
	morseSignature string,

) *MsgCreateMorseAccountClaim {
	return &MsgCreateMorseAccountClaim{
		ShannonDestAddress: shannonDestAddress,
		MorseSrcAddress:    morseSrcAddress,
		MorseSignature:     morseSignature,
	}
}

// ValidateBasic checks the validity of the morseSrcAddress and shannonDestAddress fields.
func (msg *MsgCreateMorseAccountClaim) ValidateBasic() error {
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
