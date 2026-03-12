package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = (*MsgTransferService)(nil)

func NewMsgTransferService(ownerAddress, serviceId, newOwnerAddress string) *MsgTransferService {
	return &MsgTransferService{
		OwnerAddress:    ownerAddress,
		ServiceId:       serviceId,
		NewOwnerAddress: newOwnerAddress,
	}
}

// ValidateBasic performs basic validation of the MsgTransferService fields.
func (msg *MsgTransferService) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.OwnerAddress); err != nil {
		return ErrServiceInvalidAddress.Wrapf("invalid owner address %s; (%v)", msg.OwnerAddress, err)
	}

	if _, err := sdk.AccAddressFromBech32(msg.NewOwnerAddress); err != nil {
		return ErrServiceInvalidAddress.Wrapf("invalid new owner address %s; (%v)", msg.NewOwnerAddress, err)
	}

	if msg.ServiceId == "" {
		return ErrServiceMissingID.Wrap("service ID cannot be empty")
	}

	if msg.OwnerAddress == msg.NewOwnerAddress {
		return ErrServiceInvalidOwnerAddress.Wrap("new owner address cannot be the same as the current owner address")
	}

	return nil
}
