package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/app/pocket"
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
//   - Validates the owner address is bech32 encoded.
//   - Validates the operator address is bech32 encoded.
//   - If the stake is not nil:
//   - Validates the stake amount is positive.
//   - Validates the stake denom is upokt.
//   - If services configs are provided:
//   - Validates each service config
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
		// TODO_MAINNET: Centralize stake related verification and share across different
		// parts of the source code
		// Validate the stake amount
		stake, err := sdk.ParseCoinNormalized(msg.Stake.String())
		if err != nil {
			return ErrSupplierInvalidStake.Wrapf("cannot parse supplier stake %s; (%s)", msg.Stake, err)
		}
		if !stake.IsValid() {
			return ErrSupplierInvalidStake.Wrapf("invalid supplier stake %s; (%s)", msg.Stake, stake.Validate())
		}
		if stake.IsZero() || stake.IsNegative() {
			return ErrSupplierInvalidStake.Wrapf("invalid stake amount for supplier: %s <= 0", msg.Stake)
		}
		if stake.Denom != pocket.DenomuPOKT {
			return ErrSupplierInvalidStake.Wrapf("invalid stake amount denom for supplier: expected %s, got %s", pocket.DenomuPOKT, stake.Denom)
		}
	}

	if !msg.IsSigner(msg.GetOperatorAddress()) && len(msg.GetServices()) > 0 {
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
