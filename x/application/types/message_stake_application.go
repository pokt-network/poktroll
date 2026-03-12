package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ sdk.Msg = (*MsgStakeApplication)(nil)

// TODO_TECHDEBT: See `NewMsgStakeSupplier` and follow the same pattern for the `Services` parameter
func NewMsgStakeApplication(
	appAddr string,
	stake sdk.Coin,
	appServiceConfigs []*sharedtypes.ApplicationServiceConfig,
	perSessionSpendLimit *sdk.Coin,
) *MsgStakeApplication {
	return &MsgStakeApplication{
		Address:              appAddr,
		Stake:                &stake,
		Services:             appServiceConfigs,
		PerSessionSpendLimit: perSessionSpendLimit,
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
	if err := sharedtypes.ValidateAppServiceConfigs(msg.Services); err != nil {
		return ErrAppInvalidServiceConfigs.Wrapf("%s", err.Error())
	}

	// Validate the per-session spend limit if set
	if err := ValidatePerSessionSpendLimit(msg.PerSessionSpendLimit); err != nil {
		return err
	}

	return nil
}

// ValidatePerSessionSpendLimit validates the per-session spend limit coin.
// nil = valid (no limit), zero = valid (no limit, treated same as nil),
// positive upokt = valid (active limit), negative or wrong denom = invalid.
func ValidatePerSessionSpendLimit(limit *sdk.Coin) error {
	if limit == nil {
		return nil
	}
	if limit.IsNegative() {
		return ErrAppInvalidPerSessionSpendLimit.Wrapf("per-session spend limit cannot be negative: %v", limit)
	}
	// Zero is valid (treated as no limit)
	if limit.IsZero() {
		return nil
	}
	if limit.Denom != "upokt" {
		return ErrAppInvalidPerSessionSpendLimit.Wrapf("invalid per-session spend limit denom, expecting: upokt, got: %s", limit.Denom)
	}
	if limit.Amount.LT(MinPerSessionSpendLimit.Amount) {
		return ErrAppInvalidPerSessionSpendLimit.Wrapf(
			"per_session_spend_limit %s must be at least %s",
			limit, MinPerSessionSpendLimit,
		)
	}
	return nil
}
