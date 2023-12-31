package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	types "github.com/cosmos/cosmos-sdk/types"

	servicehelpers "github.com/pokt-network/poktroll/x/shared/helpers"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const TypeMsgStakeSupplier = "stake_supplier"

var _ sdk.Msg = (*MsgStakeSupplier)(nil)

func NewMsgStakeSupplier(
	address string,
	stake types.Coin,
	services []*sharedtypes.SupplierServiceConfig,
) *MsgStakeSupplier {
	return &MsgStakeSupplier{
		Address:  address,
		Stake:    &stake,
		Services: services,
	}
}

func (msg *MsgStakeSupplier) Route() string {
	return RouterKey
}

func (msg *MsgStakeSupplier) Type() string {
	return TypeMsgStakeSupplier
}

func (msg *MsgStakeSupplier) GetSigners() []sdk.AccAddress {
	address, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{address}
}

func (msg *MsgStakeSupplier) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgStakeSupplier) ValidateBasic() error {
	// Validate the address
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return sdkerrors.Wrapf(ErrSupplierInvalidAddress, "invalid supplier address %s; (%v)", msg.Address, err)
	}

	// TODO_TECHDEBT: Centralize stake related verification and share across different parts of the source code
	// Validate the stake amount
	if msg.Stake == nil {
		return sdkerrors.Wrapf(ErrSupplierInvalidStake, "nil supplier stake; (%v)", err)
	}
	stake, err := sdk.ParseCoinNormalized(msg.Stake.String())
	if !stake.IsValid() {
		return sdkerrors.Wrapf(ErrSupplierInvalidStake, "invalid supplier stake %v; (%v)", msg.Stake, stake.Validate())
	}
	if err != nil {
		return sdkerrors.Wrapf(ErrSupplierInvalidStake, "cannot parse supplier stake %v; (%v)", msg.Stake, err)
	}
	if stake.IsZero() || stake.IsNegative() {
		return sdkerrors.Wrapf(ErrSupplierInvalidStake, "invalid stake amount for supplier: %v <= 0", msg.Stake)
	}
	if stake.Denom != "upokt" {
		return sdkerrors.Wrapf(ErrSupplierInvalidStake, "invalid stake amount denom for supplier %v", msg.Stake)
	}

	// Validate the supplier service configs
	if err := servicehelpers.ValidateSupplierServiceConfigs(msg.Services); err != nil {
		return sdkerrors.Wrapf(ErrSupplierInvalidServiceConfig, err.Error())
	}

	return nil
}
