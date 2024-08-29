package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const (
	DefaultComputeUnitsPerRelay uint64 = 1
	// ComputeUnitsPerRelayMax is the maximum allowed compute_units_per_relay value when adding or updating a service.
	// TODO_MAINNET: The reason we have a maximum is to account for potential integer overflows.
	// Should we revisit all uint64 and convert them to BigInts?
	ComputeUnitsPerRelayMax uint64 = 2 ^ 16
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
	// TODO_BETA: Add a validate basic function to the `Service` object
	if msg.Service.Id == "" {
		return ErrServiceMissingID
	}
	if msg.Service.Name == "" {
		return ErrServiceMissingName
	}
	if err := ValidateComputeUnitsPerRelay(msg.Service.ComputeUnitsPerRelay); err != nil {
		return err
	}
	return nil
}

// ValidateComputeUnitsPerRelay makes sure the compute units per relay is a valid value
func ValidateComputeUnitsPerRelay(computeUnitsPerRelay uint64) error {
	if computeUnitsPerRelay == 0 {
		return ErrServiceInvalidComputeUnitsPerRelay.Wrap("compute units per relay must be greater than 0")
	} else if computeUnitsPerRelay > ComputeUnitsPerRelayMax {
		return ErrServiceInvalidComputeUnitsPerRelay.Wrapf("compute units per relay must be less than %d", ComputeUnitsPerRelayMax)
	}
	return nil
}
