package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = (*MsgDeleteService)(nil)

func NewMsgDeleteService(serviceOwnerAddr, serviceId string) *MsgDeleteService {
	return &MsgDeleteService{
		OwnerAddress: serviceOwnerAddr,
		ServiceId:    serviceId,
	}
}

// ValidateBasic performs basic validation of the message and its fields
func (msg *MsgDeleteService) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.OwnerAddress); err != nil {
		return ErrServiceInvalidAddress.Wrapf("invalid signer address %s; (%v)", msg.OwnerAddress, err)
	}

	if msg.ServiceId == "" {
		return ErrServiceInvalidServiceId.Wrap("service ID cannot be empty")
	}

	return nil
}