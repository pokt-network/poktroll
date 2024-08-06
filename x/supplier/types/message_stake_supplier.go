package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	servicehelpers "github.com/pokt-network/poktroll/x/shared/helpers"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const TypeMsgStakeSupplier = "stake_supplier"

var _ sdk.Msg = (*MsgStakeSupplier)(nil)

func NewMsgStakeSupplier(
	senderAddress string,
	ownerAddress string,
	supplierAddress string,
	stake sdk.Coin,
	services []*sharedtypes.SupplierServiceConfig,
) *MsgStakeSupplier {
	return &MsgStakeSupplier{
		Sender:          senderAddress,
		OwnerAddress:    ownerAddress,
		OperatorAddress: supplierAddress,
		Stake:           &stake,
		Services:        services,
	}
}

func (msg *MsgStakeSupplier) ValidateBasic() error {
	// Validate the owner address
	if _, err := sdk.AccAddressFromBech32(msg.OwnerAddress); err != nil {
		return ErrSupplierInvalidAddress.Wrapf("invalid owner address %s; (%v)", msg.OwnerAddress, err)
	}

	// Validate the operator address
	if _, err := sdk.AccAddressFromBech32(msg.OperatorAddress); err != nil {
		return ErrSupplierInvalidAddress.Wrapf("invalid supplier operator address %s; (%v)", msg.OperatorAddress, err)
	}

	// Ensure the sender address matches the owner address or the operator address.
	if msg.Sender != msg.OwnerAddress && msg.Sender != msg.OperatorAddress {
		return ErrSupplierInvalidAddress.Wrapf(
			"sender address %s does not match owner address %s or supplier address %s",
			msg.Sender,
			msg.OwnerAddress,
			msg.OperatorAddress,
		)
	}

	// Ensure the sender address matches the owner address or the operator address.
	if msg.Sender != msg.OwnerAddress && msg.Sender != msg.OperatorAddress {
		return ErrSupplierInvalidAddress.Wrapf(
			"sender address %s does not match owner address %s or supplier address %s",
			msg.Sender,
			msg.OwnerAddress,
			msg.OperatorAddress,
		)
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
