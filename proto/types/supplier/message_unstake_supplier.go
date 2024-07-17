package supplier

import sdk "github.com/cosmos/cosmos-sdk/types"

const TypeMsgUnstakeSupplier = "unstake_supplier"

var _ sdk.Msg = (*MsgUnstakeSupplier)(nil)

func NewMsgUnstakeSupplier(address string) *MsgUnstakeSupplier {
	return &MsgUnstakeSupplier{
		Address: address,
	}
}

func (msg *MsgUnstakeSupplier) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return ErrSupplierInvalidAddress.Wrapf("invalid address address (%s)", err)
	}
	return nil
}
