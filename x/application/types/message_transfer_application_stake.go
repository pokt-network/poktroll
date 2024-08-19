package types

import (
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
)

var _ cosmostypes.Msg = (*MsgTransferApplicationStake)(nil)

func NewMsgTransferApplicationStake(address string, beneficiary string) *MsgTransferApplicationStake {
	return &MsgTransferApplicationStake{
		Address:     address,
		Beneficiary: beneficiary,
	}
}

func (msg *MsgTransferApplicationStake) ValidateBasic() error {
	if msg.Address == "" {
		return ErrAppInvalidAddress.Wrap("empty application address")
	}

	if msg.Beneficiary == "" {
		return ErrAppInvalidAddress.Wrap("empty beneficiary address")
	}

	_, addrErr := cosmostypes.AccAddressFromBech32(msg.Address)
	if addrErr != nil {
		return ErrAppInvalidAddress.Wrapf("invalid application address (%s): %v", msg.Address, addrErr)
	}

	_, beneficiaryErr := cosmostypes.AccAddressFromBech32(msg.Beneficiary)
	if beneficiaryErr != nil {
		return ErrAppInvalidAddress.Wrapf("invalid beneficiary address (%s): %v", msg.Address, beneficiaryErr)
	}
	return nil
}
