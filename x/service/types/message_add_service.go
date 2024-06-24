package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/shared/types"
)

var _ sdk.Msg = (*MsgAddService)(nil)

func NewMsgAddService(address, serviceId, serviceName string) *MsgAddService {
	return &MsgAddService{
		Address: address,
		Service: types.Service{
			Id:   serviceId,
			Name: serviceName,
		},
	}
}

// ValidateBasic performs basic validation of the message and its fields
func (msg *MsgAddService) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return ErrServiceInvalidAddress.Wrapf("invalid supplier address %s; (%v)", msg.Address, err)
	}
	// TODO_BETA: Add a validate basic function to the `Service` object
	if msg.Service.Id == "" {
		return ErrServiceMissingID
	}
	if msg.Service.Name == "" {
		return ErrServiceMissingName
	}
	return nil
}
