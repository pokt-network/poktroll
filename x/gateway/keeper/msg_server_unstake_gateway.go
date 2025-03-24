package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/gateway/types"
)

func (k msgServer) UnstakeGateway(
	goCtx context.Context,
	msg *types.MsgUnstakeGateway,
) (*types.MsgUnstakeGatewayResponse, error) {
	isSuccessful := false
	defer telemetry.EventSuccessCounter(
		"unstake_gateway",
		telemetry.DefaultCounterFn,
		func() bool { return isSuccessful },
	)

	ctx := sdk.UnwrapSDKContext(goCtx)

	logger := k.Logger().With("method", "UnstakeGateway")
	logger.Info(fmt.Sprintf("About to unstake gateway with msg: %v", msg))

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Check if the gateway already exists or not
	var err error
	gateway, isGatewayFound := k.GetGateway(ctx, msg.Address)
	if !isGatewayFound {
		logger.Info(fmt.Sprintf("Gateway not found. Cannot unstake address %s", msg.Address))
		return nil, status.Error(
			codes.NotFound,
			types.ErrGatewayNotFound.Wrapf(
				"gateway with address %s", msg.Address,
			).Error(),
		)
	}
	logger.Info(fmt.Sprintf("Gateway found. Unstaking gateway for address %s", msg.Address))

	// Check if the gateway has already initiated the unstaking process.
	if gateway.IsUnbonding() {
		logger.Info(fmt.Sprintf("Gateway with address [%s] is still unbonding from previous unstaking", msg.GetAddress()))
		return nil, status.Error(
			codes.FailedPrecondition,
			types.ErrGatewayIsUnstaking.Wrapf(
				"gateway with address %q", msg.GetAddress(),
			).Error(),
		)
	}

	currentHeight := ctx.BlockHeight()
	sessionEndHeight := k.sharedKeeper.GetSessionEndHeight(ctx, currentHeight)

	// Mark the gateway as unstaking by recording its deactivation height.
	//
	// Processing rules:
	// - Gateway MAY continue processing requests until current session ends
	// - After session end: Gateway becomes inactive
	gateway.UnstakeSessionEndHeight = uint64(sessionEndHeight)
	k.SetGateway(ctx, gateway)

	sharedParams := k.sharedKeeper.GetParams(ctx)
	unbondingEndHeight := types.GetGatewayUnbondingHeight(&sharedParams, &gateway)
	unbondingBeginEvent := &types.EventGatewayUnbondingBegin{
		Gateway:            &gateway,
		SessionEndHeight:   sessionEndHeight,
		UnbondingEndHeight: unbondingEndHeight,
	}
	err = ctx.EventManager().EmitTypedEvent(unbondingBeginEvent)
	if err != nil {
		err = types.ErrGatewayEmitEvent.Wrapf("(%+v): %s", unbondingBeginEvent, err)
		logger.Error(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	isSuccessful = true
	return &types.MsgUnstakeGatewayResponse{
		Gateway: &gateway,
	}, nil
}
