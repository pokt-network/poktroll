package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/pokt-network/poktroll/x/shared/types"
)

var _ sdk.Msg = &MsgAddService{}

func NewMsgAddService(address, serviceId, serviceName string) *MsgAddService {
	return &MsgAddService{
		Address: address,
		Service: types.Service{
			Id:   serviceId,
			Name: serviceName,
		},
	}
}

func (msg *MsgAddService) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid address address (%s)", err)
	}
	return nil
}
