package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	servicehelpers "github.com/pokt-network/poktroll/x/shared/helpers"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ sdk.Msg = (*MsgStakeApplication)(nil)

// TODO_TECHDEBT: See `NewMsgStakeSupplier` and follow the same pattern for the `Services` parameter
func NewMsgStakeApplication(
	appAddr string,
	stake sdk.Coin,
	appServiceConfigs []*sharedtypes.ApplicationServiceConfig,
) *MsgStakeApplication {
	return &MsgStakeApplication{
		Address:  appAddr,
		Stake:    &stake,
		Services: appServiceConfigs,
	}
}

func (msg *MsgStakeApplication) ValidateBasic() error {
	// Validate the address
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return ErrAppInvalidAddress.Wrapf("invalid application address %s; (%v)", msg.Address, err)
	}

	// TODO_TECHDEBT(@red-0ne): Centralize stake related verification and share across different parts of the source code
	// Validate the stake amount
	if msg.Stake == nil {
		return ErrAppInvalidStake.Wrapf("nil application stake; (%v)", err)
	}
	stake, err := sdk.ParseCoinNormalized(msg.Stake.String())
	if !stake.IsValid() {
		return ErrAppInvalidStake.Wrapf("invalid application stake %v; (%v)", msg.Stake, stake.Validate())
	}
	if err != nil {
		return ErrAppInvalidStake.Wrapf("cannot parse application stake %v; (%v)", msg.Stake, err)
	}
	if stake.IsZero() || stake.IsNegative() {
		return ErrAppInvalidStake.Wrapf("invalid stake amount for application: %v <= 0", msg.Stake)
	}
	if stake.Denom != "upokt" {
		return ErrAppInvalidStake.Wrapf("invalid stake amount denom for application: %v", msg.Stake)
	}

	// Validate the application service configs
	if err := servicehelpers.ValidateAppServiceConfigs(msg.Services); err != nil {
		return ErrAppInvalidServiceConfigs.Wrapf("%s", err.Error())
	}

	return nil
}
