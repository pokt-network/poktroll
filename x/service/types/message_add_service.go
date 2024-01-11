package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	shared "github.com/pokt-network/poktroll/x/shared/types"
)

// TypeMsgAddService is the name of the service message
const TypeMsgAddService = "add_service"

var _ sdk.Msg = (*MsgAddService)(nil)

// NewMsgAddService creates a new MsgAddService instance
func NewMsgAddService(address string, serviceID, serviceName string) *MsgAddService {
	return &MsgAddService{
		SupplierAddress: address,
		Service:         shared.Service{Id: serviceID, Name: serviceName},
	}
}

// Route returns the roter key for the message
func (msg *MsgAddService) Route() string {
	return RouterKey
}

// Type returns the message type
func (msg *MsgAddService) Type() string {
	return TypeMsgAddService
}

// GetSigners returns the signers of the message
func (msg *MsgAddService) GetSigners() []sdk.AccAddress {
	address, err := sdk.AccAddressFromBech32(msg.SupplierAddress)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{address}
}

// GetSignBytes returns the signable bytes of the message
func (msg *MsgAddService) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic performs basic validation of the message and its fields
func (msg *MsgAddService) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.SupplierAddress); err != nil {
		return sdkerrors.Wrapf(
			ErrServiceInvalidAddress,
			"invalid supplier address %s; (%v)", msg.SupplierAddress, err,
		)
	}
	if msg.Service.Id == "" {
		return ErrServiceMissingID
	}
	if msg.Service.Name == "" {
		return ErrServiceMissingName
	}

	return nil
}
