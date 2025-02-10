package types

import (
	"bytes"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgCreateMorseAccountState{}

func NewMsgCreateMorseAccountState(
	authority string,
	morseAccountState MorseAccountState,
) (*MsgCreateMorseAccountState, error) {
	morseAccountStateHash, err := morseAccountState.GetHash()
	if err != nil {
		return nil, err
	}

	return &MsgCreateMorseAccountState{
		Authority:             authority,
		MorseAccountState:     morseAccountState,
		MorseAccountStateHash: morseAccountStateHash,
	}, nil
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
	if len(expectedHash) == 0 {
		return ErrMorseAccountState.Wrapf("expected hash is empty")
	}

	if !bytes.Equal(actualHash, expectedHash) {
		return ErrMorseAccountState.Wrapf(
			"Morse account state hash (%x) doesn't match expected: (%x)",
			actualHash, expectedHash,
		)
	}

	return nil
}
