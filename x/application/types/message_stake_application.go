package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/pokt-network/poktroll/x/shared/types"
)

var _ sdk.Msg = &MsgStakeApplication{}

func NewMsgStakeApplication(address string, stake *sdk.Coin, services []*types.ApplicationServiceConfig) *MsgStakeApplication {
	return &MsgStakeApplication{
		Address:  address,
		Stake:    stake,
		Services: services,
	}
}

func (msg *MsgStakeApplication) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid address address (%s)", err)
	}
	return nil
}
