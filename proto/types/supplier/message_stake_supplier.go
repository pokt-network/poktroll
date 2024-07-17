package supplier

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/proto/types/shared"
	servicehelpers "github.com/pokt-network/poktroll/x/shared/helpers"
)

const TypeMsgStakeSupplier = "stake_supplier"

var _ sdk.Msg = (*MsgStakeSupplier)(nil)

func NewMsgStakeSupplier(
	address string,
	stake sdk.Coin,
	services []*sharedtypes.SupplierServiceConfig,
) *MsgStakeSupplier {
	return &MsgStakeSupplier{
		Address:  address,
		Stake:    &stake,
		Services: services,
	}
}

func (msg *MsgStakeSupplier) ValidateBasic() error {
	// Validate the address
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return ErrSupplierInvalidAddress.Wrapf("invalid supplier address %s; (%v)", msg.Address, err)
	}

	// TODO_MAINNET: Centralize stake related verification and share across different
	// parts of the source code
	// Validate the stake amount
	if msg.Stake == nil {
		return ErrSupplierInvalidStake.Wrap("nil supplier stake")
	}
	stake, err := sdk.ParseCoinNormalized(msg.Stake.String())
	if !stake.IsValid() {
		return ErrSupplierInvalidStake.Wrapf("invalid supplier stake %v; (%v)", msg.Stake, stake.Validate())
	}
	if err != nil {
		return ErrSupplierInvalidStake.Wrapf("cannot parse supplier stake %v; (%v)", msg.Stake, err)
	}
	if stake.IsZero() || stake.IsNegative() {
		return ErrSupplierInvalidStake.Wrapf("invalid stake amount for supplier: %v <= 0", msg.Stake)
	}
	if stake.Denom != "upokt" {
		return ErrSupplierInvalidStake.Wrapf("invalid stake amount denom for supplier %v", msg.Stake)
	}

	// Validate the supplier service configs
	if err := servicehelpers.ValidateSupplierServiceConfigs(msg.Services); err != nil {
		return ErrSupplierInvalidServiceConfig.Wrapf(err.Error())
	}

	return nil
}
