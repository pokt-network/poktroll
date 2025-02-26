package types

import (
	"bytes"
	"crypto/sha256"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ cosmostypes.Msg = (*MsgImportMorseClaimableAccounts)(nil)

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
	if _, err := cosmostypes.AccAddressFromBech32(msg.Authority); err != nil {
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
	if len(givenHash) != sha256.Size {
		return ErrMorseAccountsImport.Wrapf("invalid MorseAccountStateHash size")
	}

	// Validate the given hash matches the computed hash.
	if !bytes.Equal(computedHash, givenHash) {
		return ErrMorseAccountsImport.Wrapf(
			"computed MorseAccountState hash (%s) doesn't match the given MorseAccountStateHash (%s)",
			computedHash, givenHash,
		)
	}
	return nil
}

// TotalTokens returns the sum of the unstaked balance, application stake, and supplier stake.
func (m MorseClaimableAccount) TotalTokens() cosmostypes.Coin {
	return m.UnstakedBalance.
		Add(m.ApplicationStake).
		Add(m.SupplierStake)
}
