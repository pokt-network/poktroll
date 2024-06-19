package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/poktroll/x/shared/types"
)

const (
	DefaultComputeUnitsPerRelay uint64 = 1
	// ComputeUnitsPerRelayMax is the maximum allowed compute_units_per_relay value when adding or updating a service.
	// TODO_MAINNET: The reason we have a maximum is to account for potential integer overflows. This is
	// something that needs to be revisited or reconsidered prior to mainnet.
	ComputeUnitsPerRelayMax uint64 = 2 ^ 16
)

var _ sdk.Msg = (*MsgAddService)(nil)

func NewMsgAddService(address, serviceId, serviceName string, computeUnitsPerRelay uint64) *MsgAddService {
	return &MsgAddService{
		Address: address,
		Service: *types.NewService(
			serviceId,
			serviceName,
			computeUnitsPerRelay,
		),
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
	if err := ValidateComputeUnitsPerRelay(msg.Service.ComputeUnitsPerRelay); err != nil {
		return err
	}
	return nil
}

// ValidateComputeUnitsPerRelay makes sure the compute units per relay is a valid value
func ValidateComputeUnitsPerRelay(computeUnitsPerRelay uint64) error {
	if computeUnitsPerRelay == 0 {
		return ErrServiceInvalidComputUnitsPerRelay.Wrap("compute units per relay must be greater than 0")
	} else if computeUnitsPerRelay > ComputeUnitsPerRelayMax {
		return ErrServiceInvalidComputUnitsPerRelay.Wrapf("compute units per relay must be less than %d", ComputeUnitsPerRelayMax)
	}
	return nil
}
