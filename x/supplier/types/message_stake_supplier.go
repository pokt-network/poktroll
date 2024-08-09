package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	servicehelpers "github.com/pokt-network/poktroll/x/shared/helpers"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const TypeMsgStakeSupplier = "stake_supplier"

var _ sdk.Msg = (*MsgStakeSupplier)(nil)

func NewMsgStakeSupplier(
	signerAddress string,
	ownerAddress string,
	supplierAddress string,
	stake sdk.Coin,
	services []*sharedtypes.SupplierServiceConfig,
) *MsgStakeSupplier {
	return &MsgStakeSupplier{
		Signer:          signerAddress,
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

	// Validate the address
	if _, err := sdk.AccAddressFromBech32(msg.OperatorAddress); err != nil {
		return ErrSupplierInvalidAddress.Wrapf("invalid operator address %s; (%v)", msg.OperatorAddress, err)
	}

	// Validate the signer address
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return ErrSupplierInvalidAddress.Wrapf("invalid signer address %s; (%v)", msg.Signer, err)
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

func (msg *MsgStakeSupplier) IsSigner(address string) bool {
	return address == msg.Signer
}
