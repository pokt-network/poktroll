package types

import (
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
)

var _ cosmostypes.Msg = &MsgTransferApplicationStake{}

func NewMsgTransferApplicationStake(address string, beneficiary string) *MsgTransferApplicationStake {
	return &MsgTransferApplicationStake{
		Address:     address,
		Beneficiary: beneficiary,
	}
}

func (msg *MsgTransferApplicationStake) ValidateBasic() error {
	_, addrErr := cosmostypes.AccAddressFromBech32(msg.Address)
	if addrErr != nil {
		return ErrAppInvalidAddress.Wrapf("invalid application address (%s): %v", msg.Address, addrErr)
	}

	_, beneficiaryErr := cosmostypes.AccAddressFromBech32(msg.Address)
	if beneficiaryErr != nil {
		return ErrAppInvalidAddress.Wrapf("invalid beneficiary address (%s): %v", msg.Address, beneficiaryErr)
	}
	return nil
}
