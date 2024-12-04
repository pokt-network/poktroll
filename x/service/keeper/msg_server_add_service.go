package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/service/types"
)

// AddService adds a service to the network.
// The operation checks if the signer has enough funds (upokt) to pay the AddServiceFee.
// If funds are insufficient, the service won't be added. Otherwise, the fee is transferred from
// the signer to the service module's account, afterwards the service will be present on-chain.
func (k msgServer) AddService(
	goCtx context.Context,
	msg *types.MsgAddService,
) (*types.MsgAddServiceResponse, error) {
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
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Check if the service already exists or not.
	foundService, found := k.GetService(ctx, msg.Service.Id)
	if found {
		if foundService.OwnerAddress != msg.Service.OwnerAddress {
			return nil, status.Error(
				codes.InvalidArgument,
				types.ErrServiceInvalidOwnerAddress.Wrapf(
					"existing owner address %q does not match the new owner address %q",
					foundService.OwnerAddress, msg.Service.OwnerAddress,
				).Error(),
			)
		}
		return nil, status.Error(
			codes.FailedPrecondition,
			types.ErrServiceAlreadyExists.Wrapf(
				"TODO_MAINNET(@red-0ne): This is an ephemeral state of the code. Once we s/AddService/UpdateService/g, add the business logic here for updates here.",
			).Error(),
		)
	}

	// Retrieve the address of the actor adding the service; the owner of the service.
	serviceOwnerAddr, err := sdk.AccAddressFromBech32(msg.OwnerAddress)
	if err != nil {
		return nil, status.Error(
			codes.InvalidArgument,
			types.ErrServiceInvalidAddress.Wrapf(
				"%s is not in Bech32 format", msg.OwnerAddress,
			).Error(),
		)
	}

	// Check the actor has sufficient funds to pay for the add service fee.
	accCoins := k.bankKeeper.SpendableCoins(ctx, serviceOwnerAddr)
	if accCoins.Len() == 0 {
		return nil, status.Error(
			codes.FailedPrecondition,
			types.ErrServiceNotEnoughFunds.Wrapf(
				"account has no spendable coins",
			).Error(),
		)
	}

	// Check the balance of upokt is enough to cover the AddServiceFee.
	accBalance := accCoins.AmountOf("upokt")
	addServiceFee := k.GetParams(ctx).AddServiceFee
	if accBalance.LTE(addServiceFee.Amount) {
		return nil, status.Error(
			codes.FailedPrecondition,
			types.ErrServiceNotEnoughFunds.Wrapf(
				"account has %s, but the service fee is %s",
				accBalance, k.GetParams(ctx).AddServiceFee,
			).Error(),
		)
	}

	// Deduct the service fee from the actor's balance.
	serviceFee := sdk.NewCoins(*addServiceFee)
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, serviceOwnerAddr, types.ModuleName, serviceFee)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to deduct service fee from actor's balance: %+v", err))
		return nil, status.Error(
			codes.Internal,
			types.ErrServiceFailedToDeductFee.Wrapf(
				"account has %s, failed to deduct %s",
				accBalance, k.GetParams(ctx).AddServiceFee,
			).Error(),
		)
	}

	logger.Info(fmt.Sprintf("Adding service: %v", msg.Service))
	k.SetService(ctx, msg.Service)

	isSuccessful = true
	return &types.MsgAddServiceResponse{}, nil
}
