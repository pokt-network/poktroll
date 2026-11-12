package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const TypeMsgUnstakeSupplier = "unstake_supplier"

var _ sdk.Msg = (*MsgUnstakeSupplier)(nil)

func NewMsgUnstakeSupplier(signerAddress, operatorAddress string) *MsgUnstakeSupplier {
	return &MsgUnstakeSupplier{
		Signer:          signerAddress,
		OperatorAddress: operatorAddress,
	}
}

func (msg *MsgUnstakeSupplier) ValidateBasic() error {
	if err := sharedtypes.ValidateCanonicalAddress(msg.Signer); err != nil {
		return ErrSupplierInvalidAddress.Wrapf("invalid signer address (%s)", err)
	}

	if err := sharedtypes.ValidateCanonicalAddress(msg.OperatorAddress); err != nil {
		return ErrSupplierInvalidAddress.Wrapf("invalid operator address (%s)", err)
	}

	return nil
}
