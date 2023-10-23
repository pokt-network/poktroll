package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	types "github.com/cosmos/cosmos-sdk/types"

	servicehelpers "pocket/x/shared/helpers"
	sharedtypes "pocket/x/shared/types"
)

const TypeMsgStakeApplication = "stake_application"

var _ sdk.Msg = &MsgStakeApplication{}

func NewMsgStakeApplication(
	address string,
	stake types.Coin,
	serviceIds []string,
) *MsgStakeApplication {
	// Convert the serviceIds to the proper ApplicationServiceConfig type (enables future expansion)
	appServiceConfigs := make([]*sharedtypes.ApplicationServiceConfig, len(serviceIds))
	for idx, serviceId := range serviceIds {
		appServiceConfigs[idx] = &sharedtypes.ApplicationServiceConfig{
			ServiceId: &sharedtypes.ServiceId{
				Id: serviceId,
			},
		}
	}

	return &MsgStakeApplication{
		Address:  address,
		Stake:    &stake,
		Services: appServiceConfigs,
	}
}

func (msg *MsgStakeApplication) Route() string {
	return RouterKey
}

func (msg *MsgStakeApplication) Type() string {
	return TypeMsgStakeApplication
}

func (msg *MsgStakeApplication) GetSigners() []sdk.AccAddress {
	address, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{address}
}

func (msg *MsgStakeApplication) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgStakeApplication) ValidateBasic() error {
	// Validate the address
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return sdkerrors.Wrapf(ErrAppInvalidAddress, "invalid application address %s; (%v)", msg.Address, err)
	}

	// Validate the stake amount
	if msg.Stake == nil {
		return sdkerrors.Wrapf(ErrAppInvalidStake, "nil application stake; (%v)", err)
	}
	stake, err := sdk.ParseCoinNormalized(msg.Stake.String())
	if !stake.IsValid() {
		return sdkerrors.Wrapf(ErrAppInvalidStake, "invalid application stake %v; (%v)", msg.Stake, stake.Validate())
	}
	if err != nil {
		return sdkerrors.Wrapf(ErrAppInvalidStake, "cannot parse application stake %v; (%v)", msg.Stake, err)
	}
	if stake.IsZero() || stake.IsNegative() {
		return sdkerrors.Wrapf(ErrAppInvalidStake, "invalid stake amount for application: %v <= 0", msg.Stake)
	}
	if stake.Denom != "upokt" {
		return sdkerrors.Wrapf(ErrAppInvalidStake, "invalid stake amount denom for application: %v", msg.Stake)
	}

	// Validate the Services
	if len(msg.Services) == 0 {
		return sdkerrors.Wrapf(ErrAppInvalidServiceConfigs, "no services configs provided for application: %v", msg.Services)
	}
	for _, serviceConfig := range msg.Services {
		if serviceConfig == nil {
			return sdkerrors.Wrapf(ErrAppInvalidServiceConfigs, "serviceConfig cannot be nil: %v", msg.Services)
		}
		if serviceConfig.ServiceId == nil {
			return sdkerrors.Wrapf(ErrAppInvalidServiceConfigs, "serviceId cannot be nil: %v", serviceConfig)
		}
		if serviceConfig.ServiceId.Id == "" {
			return sdkerrors.Wrapf(ErrAppInvalidServiceConfigs, "serviceId.Id cannot be empty: %v", serviceConfig)
		}
		if !servicehelpers.IsValidServiceId(serviceConfig.ServiceId.Id) {
			return sdkerrors.Wrapf(ErrAppInvalidServiceConfigs, "invalid serviceId.Id: %v", serviceConfig)
		}
		if !servicehelpers.IsValidServiceName(serviceConfig.ServiceId.Name) {
			return sdkerrors.Wrapf(ErrAppInvalidServiceConfigs, "invalid serviceId.Name: %v", serviceConfig)
		}
	}

	return nil
}
