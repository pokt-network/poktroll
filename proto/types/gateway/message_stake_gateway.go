package gateway

import sdk "github.com/cosmos/cosmos-sdk/types"

var _ sdk.Msg = (*MsgStakeGateway)(nil)

func NewMsgStakeGateway(address string, stake sdk.Coin) *MsgStakeGateway {
	return &MsgStakeGateway{
		Address: address,
		Stake:   &stake,
	}
}

func (msg *MsgStakeGateway) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return ErrGatewayInvalidAddress.Wrapf("invalid gateway address %s; (%v)", msg.Address, err)
	}

	// Validate the stake amount
	if msg.Stake == nil {
		return ErrGatewayInvalidStake.Wrapf("nil gateway stake; (%v)", err)
	}
	stake, err := sdk.ParseCoinNormalized(msg.Stake.String())
	if !stake.IsValid() {
		return ErrGatewayInvalidStake.Wrapf("invalid gateway stake %v; (%v)", msg.Stake, stake.Validate())
	}
	if err != nil {
		return ErrGatewayInvalidStake.Wrapf("cannot parse gateway stake %v; (%v)", msg.Stake, err)
	}
	if stake.IsZero() || stake.IsNegative() {
		return ErrGatewayInvalidStake.Wrapf("invalid stake amount for gateway: %v <= 0", msg.Stake)
	}
	if stake.Denom != "upokt" {
		return ErrGatewayInvalidStake.Wrapf("invalid stake amount denom for gateway %v", msg.Stake)
	}
	return nil
}
