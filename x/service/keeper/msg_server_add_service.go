package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/proto/types/service"
	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/service/types"
)

// AddService adds a service to the network.
// The operation checks if the signer has enough funds (upokt) to pay the AddServiceFee.
// If funds are insufficient, the service won't be added. Otherwise, the fee is transferred from
// the signer to the service module's account, afterwards the service will be present on-chain.
func (k msgServer) AddService(
	goCtx context.Context,
	msg *service.MsgAddService,
) (*service.MsgAddServiceResponse, error) {
	isSuccessful := false
	defer telemetry.EventSuccessCounter(
		"add_service",
		telemetry.DefaultCounterFn,
		func() bool { return isSuccessful },
	)

	ctx := sdk.UnwrapSDKContext(goCtx)

	logger := k.Logger().With("method", "AddService")
	logger.Info(fmt.Sprintf("About to add a new service with msg: %v", msg))

	// Validate the message.
	if err := msg.ValidateBasic(); err != nil {
		logger.Error(fmt.Sprintf("Adding service failed basic validation: %v", err))
		return nil, err
	}

	// Check if the service already exists or not.
	if _, found := k.GetService(ctx, msg.Service.Id); found {
		logger.Error(fmt.Sprintf("Service already exists: %v", msg.Service))
		return nil, service.ErrServiceAlreadyExists.Wrapf(
			"duplicate service ID: %s", msg.Service.Id,
		)
	}

	// Retrieve the address of the actor adding the service.
	accAddr, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		logger.Error(fmt.Sprintf("could not parse address %s", msg.Address))
		return nil, service.ErrServiceInvalidAddress.Wrapf(
			"%s is not in Bech32 format", msg.Address,
		)
	}

	// Check the actor has sufficient funds to pay for the add service fee.
	accCoins := k.bankKeeper.SpendableCoins(ctx, accAddr)
	if accCoins.Len() == 0 {
		logger.Error(fmt.Sprintf("%s doesn't have any funds to add service: %v", msg.Address, err))
		return nil, service.ErrServiceNotEnoughFunds.Wrapf(
			"account has no spendable coins",
		)
	}

	// Check the balance of upokt is enough to cover the AddServiceFee.
	accBalance := accCoins.AmountOf("upokt")
	addServiceFee := math.NewIntFromUint64(k.GetParams(ctx).AddServiceFee)
	if accBalance.LTE(addServiceFee) {
		logger.Error(fmt.Sprintf("%s doesn't have enough funds to add service: %v", msg.Address, err))
		return nil, service.ErrServiceNotEnoughFunds.Wrapf(
			"account has %d uPOKT, but the service fee is %d uPOKT",
			accBalance.Uint64(), k.GetParams(ctx).AddServiceFee,
		)
	}

	// Deduct the service fee from the actor's balance.
	serviceFee := sdk.Coins{sdk.NewCoin("upokt", addServiceFee)}
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, accAddr, types.ModuleName, serviceFee)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to deduct service fee from actor's balance: %v", err))
		return nil, service.ErrServiceFailedToDeductFee.Wrapf(
			"account has %d uPOKT, failed to deduct %d uPOKT",
			accBalance.Uint64(), k.GetParams(ctx).AddServiceFee,
		)
	}

	logger.Info(fmt.Sprintf("Adding service: %v", msg.Service))
	k.SetService(ctx, msg.Service)

	isSuccessful = true
	return &service.MsgAddServiceResponse{}, nil
}
