package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	types "github.com/cosmos/cosmos-sdk/types"

	servicehelpers "github.com/pokt-network/poktroll/x/shared/helpers"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const TypeMsgStakeApplication = "stake_application"

var _ sdk.Msg = (*MsgStakeApplication)(nil)

// TODO_TECHDEBT: See `NewMsgStakeSupplier` and follow the same pattern for the `Services` parameter
func NewMsgStakeApplication(
	address string,
	stake types.Coin,
	appServiceConfigs []*sharedtypes.ApplicationServiceConfig,
) *MsgStakeApplication {

	return &MsgStakeApplication{
		Address:  address,
		Stake:    &stake,
		Services: appServiceConfigs,
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
		return sdkerrors.Wrapf(ErrAppInvalidAddress, "invalid application address %s; (%v)", msg.Address, err)
	}

	// TODO_TECHDEBT: Centralize stake related verification and share across different parts of the source code
	// Validate the stake amount
	if msg.Stake == nil {
		return sdkerrors.Wrapf(ErrAppInvalidStake, "nil application stake; (%v)", err)
	}
	stake, err := sdk.ParseCoinNormalized(msg.Stake.String())
	if !stake.IsValid() {
		return sdkerrors.Wrapf(ErrAppInvalidStake, "invalid application stake %v; (%v)", msg.Stake, stake.Validate())
	}
	if err != nil {
		return sdkerrors.Wrapf(ErrAppInvalidStake, "cannot parse application stake %v; (%v)", msg.Stake, err)
	}
	if stake.IsZero() || stake.IsNegative() {
		return sdkerrors.Wrapf(ErrAppInvalidStake, "invalid stake amount for application: %v <= 0", msg.Stake)
	}
	if stake.Denom != "upokt" {
		return sdkerrors.Wrapf(ErrAppInvalidStake, "invalid stake amount denom for application: %v", msg.Stake)
	}

	// Validate the application service configs
	if err := servicehelpers.ValidateAppServiceConfigs(msg.Services); err != nil {
		return sdkerrors.Wrapf(ErrAppInvalidServiceConfigs, err.Error())
	}

	return nil
}
