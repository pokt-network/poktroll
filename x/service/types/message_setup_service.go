package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const (
	DefaultComputeUnitsPerRelay uint64 = 1
)

var _ sdk.Msg = (*MsgSetupService)(nil)

func NewMsgSetupService(signer, serviceOwnerAddr, serviceId, serviceName string, computeUnitsPerRelay uint64) *MsgSetupService {
	return &MsgSetupService{
		Signer: signer,
		Service: sharedtypes.Service{
			Id:                   serviceId,
			Name:                 serviceName,
			ComputeUnitsPerRelay: computeUnitsPerRelay,
			OwnerAddress:         serviceOwnerAddr,
		},
	}
}

// ValidateBasic performs basic validation of the message and its fields
func (msg *MsgSetupService) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return ErrServiceInvalidAddress.Wrapf("invalid signer address %s; (%v)", msg.Signer, err)
	}

	if err := msg.Service.ValidateBasic(); err != nil {
		return err
	}
	return nil
}
