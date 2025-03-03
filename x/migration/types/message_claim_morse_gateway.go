package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgClaimMorseGateway{}

func NewMsgClaimMorseGateway(shannonDestAddress string, morseSrcAddress string, morseSignature string, stake sdk.Coin) *MsgClaimMorseGateway {
  return &MsgClaimMorseGateway{
		ShannonDestAddress: shannonDestAddress,
    MorseSrcAddress: morseSrcAddress,
    MorseSignature: morseSignature,
    Stake: stake,
	}
}

func (msg *MsgClaimMorseGateway) ValidateBasic() error {
  _, err := sdk.AccAddressFromBech32(msg.ShannonDestAddress)
  	if err != nil {
  		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid shannonDestAddress address (%s)", err)
  	}
  return nil
}

