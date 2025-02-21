package types

import (
	"bytes"
	"crypto/sha256"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = (*MsgImportMorseClaimableAccounts)(nil)

// NewMsgImportMorseClaimableAccounts constructs a MsgImportMorseClaimableAccounts
// from the given authority and morseAccountState.
func NewMsgImportMorseClaimableAccounts(
	authority string,
	morseAccountState MorseAccountState,
) (*MsgImportMorseClaimableAccounts, error) {
	morseAccountStateHash, err := morseAccountState.GetHash()
	if err != nil {
		return nil, err
	}

	return &MsgImportMorseClaimableAccounts{
		Authority:             authority,
		MorseAccountState:     morseAccountState,
		MorseAccountStateHash: morseAccountStateHash,
	}, nil
}

// ValidateBasic ensures that:
// - The authority address is valid (i.e. well-formed).
// - The MorseAccountStateHash field hash matches computed hash of the MorseAccountState field.
func (msg *MsgImportMorseClaimableAccounts) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address (%s)", err)
	}

	// Compute the hash (right now) of the MorseAccountState field.
	computedHash, err := msg.MorseAccountState.GetHash()
	if err != nil {
		return err
	}

	// givenHash is computed off-chain and included as part of MsgImportMorseClaimableAccounts;
	// the rationale being, that this consensus of the correctness of this hash will have been
	// established off-chain. It is included here to simplify the validation of the MorseAccountState
	// itself, as a ground-truth to which an on-chain computation of the hash can be compared (below).
	givenHash := msg.GetMorseAccountStateHash()

	// Validate the given hash (i.e. the MorseAccountStateHash field) length.
	if len(givenHash) == sha256.Size {
		return ErrMorseAccountState.Wrapf("expected hash is empty")
	}

	// Validate the given hash matches the computed hash.
	if !bytes.Equal(computedHash, givenHash) {
		return ErrMorseAccountState.Wrapf(
			"computed MorseAccountState hash (%s) doesn't match the given MorseAccountStateHash (%s)",
			computedHash, givenHash,
		)
	}
	return nil
}
