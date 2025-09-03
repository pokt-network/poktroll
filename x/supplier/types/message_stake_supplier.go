package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const TypeMsgStakeSupplier = "stake_supplier"

var _ sdk.Msg = (*MsgStakeSupplier)(nil)

func NewMsgStakeSupplier(
	signerAddress string,
	ownerAddress string,
	supplierOperatorAddress string,
	stake *sdk.Coin,
	services []*sharedtypes.SupplierServiceConfig,
) *MsgStakeSupplier {
	return &MsgStakeSupplier{
		Signer:          signerAddress,
		OwnerAddress:    ownerAddress,
		OperatorAddress: supplierOperatorAddress,
		Stake:           stake,
		Services:        services,
	}
}

// ValidateBasic performs the following validation steps:
//  1. Validates the owner address is bech32 encoded.
//  2. Validates the operator address is bech32 encoded.
//  3. If the stake is not nil:
//     - Validates the stake amount is positive.
//     - Validates the stake denom is upokt.
//  4. If services configs are provided:
//     - Validates each service config
func (msg *MsgStakeSupplier) ValidateBasic() error {
	// Validate the owner address
	if _, err := sdk.AccAddressFromBech32(msg.OwnerAddress); err != nil {
		return ErrSupplierInvalidAddress.Wrapf("invalid owner address %s; (%s)", msg.OwnerAddress, err)
	}

	// Validate the address
	if _, err := sdk.AccAddressFromBech32(msg.OperatorAddress); err != nil {
		return ErrSupplierInvalidAddress.Wrapf("invalid operator address %s; (%s)", msg.OperatorAddress, err)
	}

	// Validate the signer address
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return ErrSupplierInvalidAddress.Wrapf("invalid signer address %s; (%s)", msg.Signer, err)
	}

	// Validate the stake, if present.
	if msg.Stake != nil {
		if err := msg.ValidateStake(); err != nil {
			return ErrSupplierInvalidStake.Wrap(err.Error())
		}
	}

	// Ensure that ONLY the operator account is authorized to update the service configurations.
	msgHasServiceConfigs := len(msg.GetServices()) != 0
	isServicesUpdateAuthorized := msg.IsSigner(msg.GetOperatorAddress())
	if msgHasServiceConfigs && !isServicesUpdateAuthorized {
		return ErrSupplierInvalidServiceConfig.Wrap("only the operator account is authorized to update the service configurations")
	}

	// Validate the supplier service configs, if present
	if len(msg.GetServices()) != 0 {
		if err := sharedtypes.ValidateSupplierServiceConfigs(msg.GetServices()); err != nil {
			return ErrSupplierInvalidServiceConfig.Wrapf("%s", err.Error())
		}
	}

	return nil
}

func (msg *MsgStakeSupplier) IsSigner(address string) bool {
	return address == msg.Signer
}

// ValidateStake performs the following validation steps:
//   - Validates the stake amount is positive
//   - Validates the stake denom is upokt
func (msg *MsgStakeSupplier) ValidateStake() error {
	return sharedtypes.ValidatePositiveuPOKT(msg.Stake.String())
}
