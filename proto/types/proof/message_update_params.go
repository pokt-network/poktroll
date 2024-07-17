package proof

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = (*MsgUpdateParams)(nil)

// ValidateBasic does a sanity check on the provided data.
func (msg *MsgUpdateParams) ValidateBasic() error {
	// Validate the address
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return ErrProofInvalidAddress.Wrapf("invalid authority address %s; (%v)", msg.Authority, err)
	}

	if err := msg.Params.ValidateBasic(); err != nil {
		return err
	}

	return nil
}
