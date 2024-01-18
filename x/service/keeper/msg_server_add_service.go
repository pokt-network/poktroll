package keeper

import (
	"context"
	"fmt"

	sdkerrors "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/service/types"
)

// AddService handles MsgAddService and adds a service to the network storing
// it in the service keeper's store using the provided ID from the message.
// If the message's address does not have enough funds (upokt) to cover the
// AddServiceFee parameter set in the service module it will not be able to add
// the service. If it does, the fee will be deducted and debited to the service
// module's account, then the service will be added on-chain.
func (k msgServer) AddService(
	goCtx context.Context,
	msg *types.MsgAddService,
) (*types.MsgAddServiceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	logger := k.Logger(ctx).With("method", "AddService")
	logger.Info(fmt.Sprintf("About to add a new service with msg: %v", msg))

	// Validate the message.
	if err := msg.ValidateBasic(); err != nil {
		logger.Error(fmt.Sprintf("Adding service failed basic validation: %v", err))
		return nil, err
	}

	// Check if the service already exists or not.
	if _, found := k.GetService(ctx, msg.Service.Id); found {
		logger.Error(fmt.Sprintf("Service already exists: %v", msg.Service))
		return nil, sdkerrors.Wrapf(
			types.ErrServiceAlreadyExists,
			"duplicate ID: %s", msg.Service.Id,
		)
	}

	// Retrieve the address of the actor adding the service.
	accAddr, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		logger.Error(fmt.Sprintf("could not parse address %s", msg.Address))
		return nil, sdkerrors.Wrapf(
			types.ErrServiceInvalidAddress,
			"%s is not in Bech32 format", msg.Address,
		)
	}

	// Check the actor has sufficient funds to pay for the add service fee.
	accCoins := k.bankKeeper.SpendableCoins(ctx, accAddr)
	if accCoins.Len() == 0 {
		logger.Error(fmt.Sprintf("%s doesn't have any funds to add service: %v", msg.Address, err))
		return nil, sdkerrors.Wrapf(
			types.ErrServiceNotEnoughFunds,
			"account has no spendable coins",
		)
	}
	accBalance := accCoins.AmountOf("upokt")
	if accBalance.LTE(sdkmath.NewIntFromUint64(k.GetParams(ctx).AddServiceFee)) {
		logger.Error(fmt.Sprintf("%s doesn't have enough funds to add service: %v", msg.Address, err))
		return nil, sdkerrors.Wrapf(
			types.ErrServiceNotEnoughFunds,
			"account has %d uPOKT, but the service fee is %d uPOKT",
			accBalance.Uint64(), k.GetParams(ctx).AddServiceFee,
		)
	}

	// Deduct the service fee from the actor's balance.
	serviceFee := sdk.Coins{sdk.NewCoin("upokt", sdkmath.NewIntFromUint64(k.GetParams(ctx).AddServiceFee))}
	err = k.bankKeeper.DelegateCoinsFromAccountToModule(ctx, accAddr, types.ModuleName, serviceFee)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to deduct service fee from actor's balance: %v", err))
		return nil, sdkerrors.Wrapf(
			types.ErrServiceFailedToDeductFee,
			"account has %d uPOKT, failed to deduct %d uPOKT",
			accBalance.Uint64(), k.GetParams(ctx).AddServiceFee,
		)
	}

	logger.Info(fmt.Sprintf("Adding service: %v", msg.Service))
	k.SetService(ctx, msg.Service)

	return &types.MsgAddServiceResponse{}, nil
}
