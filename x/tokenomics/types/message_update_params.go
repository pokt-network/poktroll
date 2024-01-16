package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const TypeMsgUpdateParams = "update_params"

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

func (msg *MsgUpdateParams) Route() string {
	return RouterKey
}

func (msg *MsgUpdateParams) Type() string {
	return TypeMsgUpdateParams
}

func (msg *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	authority, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{authority}
}

func (msg *MsgUpdateParams) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgUpdateParams) ValidateBasic() error {
	// Validate the address
	_, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return sdkerrors.Wrapf(ErrTokenomicsAuthorityInvalidAddress, "invalid authority address %s; (%v)", msg.Authority, err)
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
		return sdkerrors.Wrapf(ErrTokenomicsParamsInvalid, "invalid ComputeUnitsToTokensMultiplier; (%v)", params.ComputeUnitsToTokensMultiplier)
	}
	return nil
}
