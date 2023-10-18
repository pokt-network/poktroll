package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	types "github.com/cosmos/cosmos-sdk/types"
)

const TypeMsgStakeGateway = "stake_gateway"

var _ sdk.Msg = &MsgStakeGateway{}

func NewMsgStakeGateway(address string, stake types.Coin) *MsgStakeGateway {
	return &MsgStakeGateway{
		Address: address,
		Stake:   &stake,
	}
}

func (msg *MsgStakeGateway) Route() string {
	return RouterKey
}

func (msg *MsgStakeGateway) Type() string {
	return TypeMsgStakeGateway
}

func (msg *MsgStakeGateway) GetSigners() []sdk.AccAddress {
	address, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{address}
}

func (msg *MsgStakeGateway) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgStakeGateway) ValidateBasic() error {
	// Validate the address
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return sdkerrors.Wrapf(ErrGatewayInvalidAddress, "invalid gateway address %s; (%v)", msg.Address, err)
	}

	// Validate the stake amount
	if msg.Stake == nil {
		return sdkerrors.Wrapf(ErrGatewayInvalidStake, "nil gateway stake; (%v)", err)
	}
	stakeAmount, err := sdk.ParseCoinNormalized(msg.Stake.String())
	if !stakeAmount.IsValid() {
		return sdkerrors.Wrapf(ErrGatewayInvalidStake, "invalid gateway stake %v; (%v)", msg.Stake, stakeAmount.Validate())
	}
	if err != nil {
		return sdkerrors.Wrapf(ErrGatewayInvalidStake, "cannot parse gateway stake %v; (%v)", msg.Stake, err)
	}
	if stakeAmount.IsZero() || stakeAmount.IsNegative() {
		return sdkerrors.Wrapf(ErrGatewayInvalidStake, "invalid stake amount for gateway: %v <= 0", msg.Stake)
	}
	if stakeAmount.Denom != "upokt" {
		return sdkerrors.Wrapf(ErrGatewayInvalidStake, "invalid stake amount denom for gateway %v", msg.Stake)
	}
	return nil
}
