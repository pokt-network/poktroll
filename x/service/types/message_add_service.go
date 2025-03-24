package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/pocket/x/shared/types"
)

const (
	DefaultComputeUnitsPerRelay uint64 = 1
)

var _ sdk.Msg = (*MsgAddService)(nil)

func NewMsgAddService(serviceOwnerAddr, serviceId, serviceName string, computeUnitsPerRelay uint64) *MsgAddService {
	return &MsgAddService{
		OwnerAddress: serviceOwnerAddr,
		Service: sharedtypes.Service{
			Id:                   serviceId,
			Name:                 serviceName,
			ComputeUnitsPerRelay: computeUnitsPerRelay,
			OwnerAddress:         serviceOwnerAddr,
		},
	}
}

// ValidateBasic performs basic validation of the message and its fields
func (msg *MsgAddService) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.OwnerAddress); err != nil {
		return ErrServiceInvalidAddress.Wrapf("invalid signer address %s; (%v)", msg.OwnerAddress, err)
	}
	// Ensure that the signer of the add_service message is the owner of the service.
	if msg.Service.OwnerAddress != msg.OwnerAddress {
		return ErrServiceInvalidOwnerAddress.Wrapf("service owner address %q does not match the signer address %q", msg.Service.OwnerAddress, msg.OwnerAddress)
	}

	if err := msg.Service.ValidateBasic(); err != nil {
		return err
	}
	return nil
}
