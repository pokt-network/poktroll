package types

import (
	"encoding/hex"

	errorsmod "cosmossdk.io/errors"
	cometcrypto "github.com/cometbft/cometbft/crypto/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/gogoproto/proto"
)

var _ sdk.Msg = &MsgClaimMorseAccount{}

func NewMsgClaimMorseAccount(
	shannonDestAddress string,
	morseSrcAddress string,
	morsePrivateKey cometcrypto.PrivKey,
) (*MsgClaimMorseAccount, error) {
	msg := &MsgClaimMorseAccount{
		ShannonDestAddress: shannonDestAddress,
		MorseSrcAddress:    morseSrcAddress,
	}
	msgBz, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	morseSignature, err := morsePrivateKey.Sign(msgBz)
	msg.MorseSignature = hex.EncodeToString(morseSignature)

	return msg, nil
}

func (msg *MsgClaimMorseAccount) ValidateBasic() error {

	// TODO_UPNEXT(@bryanchriswhite#1034): Add validation

	_, err := sdk.AccAddressFromBech32(msg.ShannonDestAddress)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid shannonDestAddress address (%s)", err)
	}
	return nil
}
