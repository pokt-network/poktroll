package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	types "github.com/cosmos/cosmos-sdk/types"
)

const TypeMsgStakeApplication = "stake_application"

var _ sdk.Msg = &MsgStakeApplication{}

func NewMsgStakeApplication(
	address string,
	stakeAmount types.Coin,

) *MsgStakeApplication {
	return &MsgStakeApplication{
		Address: address,
		Stake:   &stakeAmount,
	}
}

func (msg *MsgStakeApplication) Route() string {
	return RouterKey
}

func (msg *MsgStakeApplication) Type() string {
	return TypeMsgStakeApplication
}

func (msg *MsgStakeApplication) GetSigners() []sdk.AccAddress {
	address, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{address}
}

func (msg *MsgStakeApplication) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgStakeApplication) ValidateBasic() error {
	// Validate the address
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return errorsmod.Wrapf(ErrInvalidAppAddress, "invalid application address %s; (%v)", msg.Address, err)
	}

	// Validate the stake amount
	if msg.Stake == nil {
		return errorsmod.Wrapf(ErrInvalidAppStake, "nil application stake; (%v)", err)
	}
	stakeAmount, err := sdk.ParseCoinNormalized(msg.Stake.String())
	if !stakeAmount.IsValid() {
		return errorsmod.Wrapf(ErrInvalidAppStake, "invalid application stake %v; (%v)", msg.Stake, stakeAmount.Validate())
	}
	if err != nil {
		return errorsmod.Wrapf(ErrInvalidAppStake, "cannot parse application stake %v; (%v)", msg.Stake, err)
	}

	return nil
}
