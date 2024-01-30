package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg              = (*MsgStakeGateway)(nil)
	_ sdk.HasValidateBasic = (*MsgStakeGateway)(nil)
)

// NewMsgStakeGateway creates a new MsgStakeGateway instance.
func NewMsgStakeGateway(address string, stake sdk.Coin) *MsgStakeGateway {
	return &MsgStakeGateway{
		Address: address,
		Stake:   stake,
	}
}

// ValidateBasic performs validation on the message allowing it to verify
// the fields it was created with.
func (msg *MsgStakeGateway) ValidateBasic() error {
	// Validate the address
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return ErrGatewayInvalidAddress.Wrapf(
			"invalid gateway address %s; (%v)",
			msg.Address, err,
		)
	}

	// Validate the stake amount
	stake, err := sdk.ParseCoinNormalized(msg.Stake.String())
	if !stake.IsValid() {
		return ErrGatewayInvalidStake.Wrapf(
			"invalid gateway stake %v; (%v)",
			msg.Stake, stake.Validate(),
		)
	}
	if err != nil {
		return ErrGatewayInvalidStake.Wrapf(
			"cannot parse gateway stake %v; (%v)",
			msg.Stake, err,
		)
	}
	if stake.IsZero() || stake.IsNegative() {
		return ErrGatewayInvalidStake.Wrapf(
			"invalid stake amount for gateway: %v <= 0",
			msg.Stake,
		)
	}
	if stake.Denom != "upokt" {
		return ErrGatewayInvalidStake.Wrapf(
			"invalid stake amount denom for gateway %v",
			msg.Stake,
		)
	}
	return nil
}
