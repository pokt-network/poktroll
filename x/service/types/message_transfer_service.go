package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
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
	if err := sharedtypes.ValidateCanonicalAddress(msg.OwnerAddress); err != nil {
		return ErrServiceInvalidAddress.Wrapf("invalid owner address %s; (%v)", msg.OwnerAddress, err)
	}

	if err := sharedtypes.ValidateCanonicalAddress(msg.NewOwnerAddress); err != nil {
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
