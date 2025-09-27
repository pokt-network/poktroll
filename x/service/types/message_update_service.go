package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ sdk.Msg = (*MsgUpdateService)(nil)

func NewMsgUpdateService(serviceOwnerAddr string, service sharedtypes.Service) *MsgUpdateService {
	return &MsgUpdateService{
		OwnerAddress: serviceOwnerAddr,
		Service:      service,
	}
}

// ValidateBasic performs basic validation of the message and its fields
func (msg *MsgUpdateService) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.OwnerAddress); err != nil {
		return ErrServiceInvalidAddress.Wrapf("invalid signer address %s; (%v)", msg.OwnerAddress, err)
	}
	// Ensure that the signer of the update_service message is the owner of the service.
	if msg.Service.OwnerAddress != msg.OwnerAddress {
		return ErrServiceInvalidOwnerAddress.Wrapf("service owner address %q does not match the signer address %q", msg.Service.OwnerAddress, msg.OwnerAddress)
	}

	if err := msg.Service.ValidateBasic(); err != nil {
		return err
	}
	return nil
}