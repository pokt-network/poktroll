package types

import sdk "github.com/cosmos/cosmos-sdk/types"

const TypeMsgUnstakeSupplier = "unstake_supplier"

var _ sdk.Msg = (*MsgUnstakeSupplier)(nil)

func NewMsgUnstakeSupplier(ownerAddress, address string) *MsgUnstakeSupplier {
	return &MsgUnstakeSupplier{
		OwnerAddress: ownerAddress,
		Address:      address,
	}
}

func (msg *MsgUnstakeSupplier) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.OwnerAddress); err != nil {
		return ErrSupplierInvalidAddress.Wrapf("invalid address address (%s)", err)
	}

	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return ErrSupplierInvalidAddress.Wrapf("invalid address address (%s)", err)
	}

	return nil
}
