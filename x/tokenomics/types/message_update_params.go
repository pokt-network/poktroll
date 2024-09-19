package types

import sdk "github.com/cosmos/cosmos-sdk/types"

var _ sdk.Msg = (*MsgUpdateParams)(nil)

func NewMsgUpdateParams(authority string) *MsgUpdateParams {
	return &MsgUpdateParams{
		Authority: authority,
		Params:    Params{},
	}
}

// ValidateBasic does a sanity check on the provided data.
func (msg *MsgUpdateParams) ValidateBasic() error {
	// Validate the address
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return ErrTokenomicsAddressInvalid.Wrapf("invalid authority address %s; (%v)", msg.Authority, err)
	}

	// Validate the params
	if err := msg.Params.ValidateBasic(); err != nil {
		return err
	}

	return nil
}
