package types

import (
	"bytes"

	"cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgCreateMorseAccountState{}

func NewMsgCreateMorseAccountState(authority string, morseAccountState MorseAccountState) *MsgCreateMorseAccountState {
	return &MsgCreateMorseAccountState{
		Authority:         authority,
		MorseAccountState: morseAccountState,
	}
}

func (msg *MsgCreateMorseAccountState) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address (%s)", err)
	}

	actualHash, err := msg.MorseAccountState.GetHash()
	if err != nil {
		return err
	}

	expectedHash := msg.GetMorseAccountStateHash()
	if bytes.Equal(actualHash, expectedHash) {
		return nil
	}

	return types.ErrInvalidRequest.Wrapf(
		"Morse account state hash (%s) doesn't match expected: (%s)",
		actualHash, expectedHash,
	)
}
