package types

import sdk "github.com/cosmos/cosmos-sdk/types"

const TypeMsgUnstakeSupplier = "unstake_supplier"

var _ sdk.Msg = (*MsgUnstakeSupplier)(nil)

func NewMsgUnstakeSupplier(signerAddress, address string) *MsgUnstakeSupplier {
	return &MsgUnstakeSupplier{
		Signer:  signerAddress,
		Address: address,
	}
}

func (msg *MsgUnstakeSupplier) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return ErrSupplierInvalidAddress.Wrapf("invalid address address (%s)", err)
	}

	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return ErrSupplierInvalidAddress.Wrapf("invalid address address (%s)", err)
	}

	return nil
}
