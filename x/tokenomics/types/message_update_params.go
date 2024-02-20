package types

import sdk "github.com/cosmos/cosmos-sdk/types"

var _ sdk.Msg = (*MsgUpdateParams)(nil)

func NewMsgUpdateParams(
	authority string,
	compute_units_to_tokens_multiplier uint64,
) *MsgUpdateParams {
	return &MsgUpdateParams{
		Authority: authority,
		Params: Params{
			ComputeUnitsToTokensMultiplier: compute_units_to_tokens_multiplier,
		},
	}
}

// ValidateBasic does a sanity check on the provided data.
func (msg *MsgUpdateParams) ValidateBasic() error {
	// Validate the address
	_, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return ErrTokenomicsAuthorityAddressInvalid.Wrapf("invalid authority address %s; (%v)", msg.Authority, err)
	}

	// Validate the params
	if err := msg.Params.ValidateBasic(); err != nil {
		return err
	}

	return nil
}

func (params *Params) ValidateBasic() error {
	// Validate the ComputeUnitsToTokensMultiplier
	if params.ComputeUnitsToTokensMultiplier == 0 {
		return ErrTokenomicsParamsInvalid.Wrapf("invalid ComputeUnitsToTokensMultiplier; (%v)", params.ComputeUnitsToTokensMultiplier)
	}
	return nil
}
